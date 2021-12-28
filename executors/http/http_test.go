package http

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutor_TLSOptions_From_File(t *testing.T) {
	e := Executor{
		IgnoreVerifySSL: true,
		TLSClientCert:   "../../tests/http/tls/http-client-tls.crt",
		TLSClientKey:    "../../tests/http/tls/http-client-tls.key",
		TLSRootCA:       "../../tests/http/tls/digicert-root-ca.crt",
	}
	opts, err := e.TLSOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, opts, 3)
}

func TestExecutor_TLSOptions_From_String(t *testing.T) {
	TLSClientCert, err := os.ReadFile("../../tests/http/tls/http-client-tls.crt")
	require.NoError(t, err)
	TLSClientKey, err := os.ReadFile("../../tests/http/tls/http-client-tls.key")
	require.NoError(t, err)
	TLSRootCA, err := os.ReadFile("../../tests/http/tls/digicert-root-ca.crt")
	require.NoError(t, err)
	e := Executor{
		TLSClientCert: string(TLSClientCert),
		TLSClientKey:  string(TLSClientKey),
		TLSRootCA:     string(TLSRootCA),
	}
	opts, err := e.TLSOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, opts, 2)
}
