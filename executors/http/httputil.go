package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"

	"golang.org/x/net/http2"

	"github.com/ovh/venom"
)

func GetTransport(opts ...func(*http.Transport) error) (*http.Transport, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	for _, o := range opts {
		if err := o(tr); err != nil {
			return tr, err
		}
	}

	_ = http2.ConfigureTransport(tr)
	return tr, nil
}

func WithTLSInsecureSkipVerify(v bool) func(*http.Transport) error {
	return func(t *http.Transport) error {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}

		t.TLSClientConfig.InsecureSkipVerify = v
		return nil
	}
}

func WithTLSClientAuth(cert tls.Certificate) func(*http.Transport) error {
	return func(t *http.Transport) error {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}

		t.TLSClientConfig.Certificates = append(t.TLSClientConfig.Certificates, cert)
		return nil
	}
}

// WithTLSRootCA should be called only once, with multiple PEM encoded certificates as input if needed.
func WithTLSRootCA(ctx context.Context, caCert []byte) func(*http.Transport) error {
	return func(t *http.Transport) error {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			venom.Warning(ctx, "http: tls: failed to load default system cert pool, fallback to an empty cert pool")
			caCertPool = x509.NewCertPool()
		}

		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return errors.New("WithTLSRootCA: failed to add a certificate to the cert pool")
		}

		t.TLSClientConfig.RootCAs = caCertPool
		return nil
	}
}

func WithProxyFromEnv() func(*http.Transport) error {
	return func(t *http.Transport) error {
		t.Proxy = http.ProxyFromEnvironment
		return nil
	}
}
