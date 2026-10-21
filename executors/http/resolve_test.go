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
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/venom"
	httpexecutor "github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/ovhapi"
)

func TestExecutors_ResolveTLS(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	require.NoError(t, err)
	rootCA := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}))
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"host": r.Host, "sni": r.TLS.ServerName})
	}))
	require.NoError(t, srv.Listener.Close())
	srv.Listener = listener
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{cert}, PrivateKey: privateKey}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	for _, executor := range []struct {
		name string
		run  func(context.Context, venom.TestStep) (interface{}, error)
	}{
		{"http", (httpexecutor.Executor{}).Run},
		{"ovhapi", (ovhapi.Executor{}).Run},
	} {
		t.Run(executor.name, func(t *testing.T) {
			for _, tt := range []struct {
				name, source, hostHeader string
			}{
				{"verified hostname", "localhost", ""},
				{"custom Host preserves SNI", "localhost", "custom.example"},
				{"hostname verification fails", "127.0.0.1", ""},
			} {
				t.Run(tt.name, func(t *testing.T) {
					host := net.JoinHostPort(tt.source, port)
					step := venom.TestStep{
						"url":               "https://" + host,
						"endpoint":          "https://" + host,
						"path":              "/healthz",
						"applicationKey":    "unused",
						"applicationSecret": "unused",
						"noAuth":            true,
						"resolve":           []string{host + ":::1"},
						"tls_root_ca":       rootCA,
						"tlsRootCA":         rootCA,
					}
					if tt.hostHeader != "" {
						step["headers"] = map[string]string{"Host": tt.hostHeader}
					}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					res, err := executor.run(ctx, step)
					if tt.source == "127.0.0.1" {
						var hostnameError x509.HostnameError
						require.ErrorAs(t, err, &hostnameError)
						require.Equal(t, "127.0.0.1", hostnameError.Host)
						return
					}
					require.NoError(t, err)
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
					var response map[string]string
					require.NoError(t, json.Unmarshal([]byte(body), &response))
					wantHost := host
					if tt.hostHeader != "" {
						wantHost = tt.hostHeader
					}
					require.Equal(t, wantHost, response["host"])
					require.Equal(t, "localhost", response["sni"])
				})
			}
		})
	}
}

func TestExecutors_ResolveProxy(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	requests := make(chan [2]string, 2)
	proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- [2]string{r.URL.String(), r.Host}
		_, _ = w.Write([]byte(`{"proxied":true}`))
	}))
	require.NoError(t, proxy.Listener.Close())
	proxy.Listener = listener
	proxy.Start()
	t.Cleanup(proxy.Close)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	proxyHost := net.JoinHostPort("localhost", port)

	for _, executor := range []struct {
		name string
		run  func(context.Context, venom.TestStep) (interface{}, error)
	}{
		{"http", (httpexecutor.Executor{}).Run},
		{"ovhapi", (ovhapi.Executor{}).Run},
	} {
		t.Run(executor.name, func(t *testing.T) {
			step := venom.TestStep{
				"url":               "http://collector.local",
				"endpoint":          "http://collector.local",
				"path":              "/healthz",
				"applicationKey":    "unused",
				"applicationSecret": "unused",
				"noAuth":            true,
				"proxy":             "http://" + proxyHost,
				"resolve":           []string{"collector.local:80:192.0.2.1", proxyHost + ":::1"},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := executor.run(ctx, step)
			require.NoError(t, err)
			require.Equal(t, [2]string{"http://collector.local/healthz", "collector.local"}, <-requests)

			step["resolve"] = []string{"collector.local:80:[::1"}
			_, err = executor.run(ctx, step)
			require.ErrorContains(t, err, "invalid value for resolve attribute:")
			require.ErrorContains(t, err, "invalid IPv6 destination")
		})
	}
}
