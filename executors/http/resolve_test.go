// Copyright 2026 Venom contributors.
// Licensed under the Apache License, Version 2.0 (see LICENSE).

package http_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/venom"
	httpexecutor "github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/ovhapi"
)

var resolveExecutors = []struct {
	name string
	impl venom.Executor
}{
	{"http", httpexecutor.Executor{}},
	{"ovhapi", ovhapi.Executor{}},
}

type resolveRequest struct {
	Method, URL, Host, SNI string
}

func echoResolveRequest(w http.ResponseWriter, r *http.Request) {
	request := resolveRequest{Method: r.Method, URL: r.URL.String(), Host: r.Host}
	if r.TLS != nil {
		request.SNI = r.TLS.ServerName
	}
	w.Header().Set("Connection", "close")
	_ = json.NewEncoder(w).Encode(request)
}

// The returned IPv4 address has the same port as the IPv6-only listener, but
// cannot reach it. Numeric loopback also bypasses environment proxies.
func newIPv6Server(t *testing.T, handler http.HandlerFunc, config *tls.Config) (*httptest.Server, string) {
	t.Helper()
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	srv := httptest.NewUnstartedServer(handler)
	require.NoError(t, srv.Listener.Close())
	srv.Listener, srv.TLS = listener, config
	if config == nil {
		srv.Start()
	} else {
		srv.StartTLS()
	}
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	return srv, net.JoinHostPort("127.0.0.1", port)
}

func resolveCertificate(t *testing.T, hostname string) (*tls.Config, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(hostname); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{hostname}
	}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	require.NoError(t, err)
	config := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{cert}, PrivateKey: privateKey}}}
	return config, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}))
}

func resolveStep(target string) venom.TestStep {
	return venom.TestStep{
		"url": target, "endpoint": target, "path": "/healthz",
		"applicationKey": "unused", "applicationSecret": "unused", "noAuth": true,
	}
}

func runResolve(t *testing.T, executor venom.Executor, step venom.TestStep) (resolveRequest, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := executor.Run(ctx, step)
	var request resolveRequest
	if err != nil {
		return request, err
	}
	var body string
	switch result := res.(type) {
	case httpexecutor.Result:
		require.Equal(t, http.StatusOK, result.StatusCode)
		body = result.Body
	case ovhapi.Result:
		require.Equal(t, http.StatusOK, result.StatusCode)
		body = result.Body
	default:
		t.Fatalf("unexpected result %T", res)
	}
	require.NoError(t, json.Unmarshal([]byte(body), &request))
	return request, nil
}

func requestWithResolve(t *testing.T, executor venom.Executor, step venom.TestStep, address, destination string) (resolveRequest, error) {
	t.Helper()
	// Require a dial failure at the original address, not a DNS, proxy, or TLS error.
	_, err := runResolve(t, executor, step)
	require.ErrorIs(t, err, syscall.ECONNREFUSED)
	var dialError *net.OpError
	require.ErrorAs(t, err, &dialError)
	if dialError.Op == "proxyconnect" {
		require.ErrorAs(t, dialError.Err, &dialError)
	}
	require.Equal(t, "dial", dialError.Op)
	require.Equal(t, address, dialError.Addr.String())

	step["resolve"] = []string{address + ":" + destination}
	return runResolve(t, executor, step)
}

func TestExecutors_Resolve(t *testing.T) {
	for _, executor := range resolveExecutors {
		t.Run(executor.name, func(t *testing.T) {
			for _, destination := range []string{"::1", "[::1]"} {
				t.Run(destination, func(t *testing.T) {
					_, address := newIPv6Server(t, echoResolveRequest, nil)
					step := resolveStep("http://" + address)
					request, err := requestWithResolve(t, executor.impl, step, address, destination)
					require.NoError(t, err)
					require.Equal(t, resolveRequest{Method: "GET", URL: "/healthz", Host: address}, request)

					step["resolve"] = []string{address + ":[::1"}
					_, err = runResolve(t, executor.impl, step)
					require.ErrorContains(t, err, "invalid IPv6 destination")
				})
			}
		})
	}
}

func TestExecutors_ResolveTLS(t *testing.T) {
	for _, executor := range resolveExecutors {
		t.Run(executor.name, func(t *testing.T) {
			for _, tt := range []struct{ name, certificate string }{
				{"verified IP", "127.0.0.1"},
				{"hostname mismatch", "wrong.example"},
				{"untrusted certificate", "127.0.0.1"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					config, rootCA := resolveCertificate(t, tt.certificate)
					_, address := newIPv6Server(t, echoResolveRequest, config)
					step := resolveStep("https://" + address)
					if tt.name != "untrusted certificate" {
						step["tls_root_ca"], step["tlsRootCA"] = rootCA, rootCA
					}
					request, err := requestWithResolve(t, executor.impl, step, address, "::1")
					switch tt.name {
					case "hostname mismatch":
						var hostnameError x509.HostnameError
						require.ErrorAs(t, err, &hostnameError)
						require.Equal(t, "127.0.0.1", hostnameError.Host)
					case "untrusted certificate":
						var authorityError x509.UnknownAuthorityError
						require.ErrorAs(t, err, &authorityError)
					default:
						require.NoError(t, err)
						require.Equal(t, resolveRequest{Method: "GET", URL: "/healthz", Host: address}, request)
					}
				})
			}
		})
	}
}

func TestExecutors_ResolveProxy(t *testing.T) {
	for _, executor := range resolveExecutors {
		t.Run(executor.name, func(t *testing.T) {
			for _, tt := range []struct{ scheme, hostHeader string }{
				{"http", ""}, {"https", ""}, {"https", "custom.example"},
			} {
				t.Run(tt.scheme+tt.hostHeader, func(t *testing.T) {
					step := resolveStep(tt.scheme + "://collector.local:8443")
					want := resolveRequest{Method: "GET", URL: "/healthz", Host: "collector.local:8443"}
					var proxyHandler http.HandlerFunc = echoResolveRequest
					if tt.scheme == "https" {
						config, rootCA := resolveCertificate(t, "collector.local")
						srv, _ := newIPv6Server(t, echoResolveRequest, config)
						proxyHandler = resolveTunnel(srv.Listener.Addr().String())
						step["tls_root_ca"], step["tlsRootCA"] = rootCA, rootCA
						want.SNI = "collector.local"
					} else {
						want.URL = "http://collector.local:8443/healthz"
					}
					if tt.hostHeader != "" {
						step["headers"] = map[string]string{"Host": tt.hostHeader}
						want.Host = tt.hostHeader
					}
					_, address := newIPv6Server(t, proxyHandler, nil)
					step["proxy"] = "http://" + address // Overrides all environment proxy settings.
					request, err := requestWithResolve(t, executor.impl, step, address, "::1")
					require.NoError(t, err)
					require.Equal(t, want, request)
				})
			}
		})
	}
}

// CONNECT forwards to a known local address without looking up the hostname.
func resolveTunnel(address string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Host != "collector.local:8443" {
			http.Error(w, "unexpected CONNECT target", http.StatusBadRequest)
			return
		}
		upstream, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		conn, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		_ = upstream.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := buffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			return
		}
		if err := buffer.Flush(); err != nil {
			return
		}
		go func() {
			_, _ = io.Copy(upstream, buffer)
			_ = upstream.Close()
		}()
		_, _ = io.Copy(conn, upstream)
	}
}
