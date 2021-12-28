package http

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateClientFile(t *testing.T) (string, string) {
	TLSClientKey, err := os.CreateTemp(os.TempDir(), "TLSClientKey.*.key")
	require.NoError(t, err)
	TLSClientKeyFileName := TLSClientKey.Name()
	t.Logf("generating file %q", TLSClientKeyFileName)
	cmd := exec.Command("openssl", "genrsa", "-out", TLSClientKeyFileName, "2048")
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	require.NoError(t, err)

	TLSClientCert, err := os.CreateTemp(os.TempDir(), "TLSClientCert.*.crt")
	require.NoError(t, err)
	TLSClientCertFilename := TLSClientCert.Name()
	t.Logf("generating file %q", TLSClientCertFilename)
	cmd = exec.Command("openssl", "req", "-batch", "-subj", "/C=GB/ST=Yorks/L=York/O=MyCompany Ltd./OU=IT/CN=mysubdomain.mydomain.com", "-new", "-x509", "-sha256", "-key", TLSClientKeyFileName, "-out", TLSClientCertFilename, "-days", "365")
	output, err = cmd.CombinedOutput()
	t.Log(string(output))
	require.NoError(t, err)

	return TLSClientKeyFileName, TLSClientCertFilename
}

func TestExecutor_TLSOptions_From_File(t *testing.T) {
	TLSClientKeyFileName, TLSClientCertFilename := generateClientFile(t)

	e := Executor{
		IgnoreVerifySSL: true,
		TLSClientCert:   TLSClientCertFilename,
		TLSClientKey:    TLSClientKeyFileName,
		TLSRootCA:       "../../tests/http/tls/digicert-root-ca.crt",
	}
	opts, err := e.TLSOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, opts, 3)
}

func TestExecutor_TLSOptions_From_String(t *testing.T) {
	TLSClientKeyFileName, TLSClientCertFilename := generateClientFile(t)

	TLSClientCert, err := os.ReadFile(TLSClientCertFilename)
	require.NoError(t, err)
	TLSClientKey, err := os.ReadFile(TLSClientKeyFileName)
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
