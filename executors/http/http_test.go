package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/venom"
)

// generateClientFile creates a TLS key and self-signed certificate inside dir.
// Returns the file names (relative to dir) so tests can resolve them through
// venom.ResolveWorkdirPath.
func generateClientFile(t *testing.T, dir string) (string, string) {
	keyPath := "TLSClientKey.key"
	certPath := "TLSClientCert.crt"
	absKey := dir + "/" + keyPath
	absCert := dir + "/" + certPath

	t.Logf("generating file %q", absKey)
	cmd := exec.Command("openssl", "genrsa", "-out", absKey, "2048")
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	require.NoError(t, err)

	t.Logf("generating file %q", absCert)
	cmd = exec.Command("openssl", "req", "-batch", "-subj", "/C=GB/ST=Yorks/L=York/O=MyCompany Ltd./OU=IT/CN=mysubdomain.mydomain.com", "-new", "-x509", "-sha256", "-key", absKey, "-out", absCert, "-days", "365")
	output, err = cmd.CombinedOutput()
	t.Log(string(output))
	require.NoError(t, err)

	return keyPath, certPath
}

func ctxWithWorkdir(workdir string) context.Context {
	return context.WithValue(context.Background(), venom.ContextKey("var.venom.testsuite.workdir"), workdir)
}

func TestExecutor_TLSOptions_From_File(t *testing.T) {
	workdir := t.TempDir()
	keyName, certName := generateClientFile(t, workdir)

	rootCA, err := os.ReadFile("../../tests/http/tls/digicert-root-ca.crt")
	require.NoError(t, err)
	rootCAPath := "digicert-root-ca.crt"
	require.NoError(t, os.WriteFile(workdir+"/"+rootCAPath, rootCA, 0o600))

	e := Executor{
		IgnoreVerifySSL: true,
		TLSClientCert:   certName,
		TLSClientKey:    keyName,
		TLSRootCA:       rootCAPath,
	}
	opts, err := e.TLSOptions(ctxWithWorkdir(workdir))
	require.NoError(t, err)
	require.Len(t, opts, 3)
}

func TestExecutor_TLSOptions_From_String(t *testing.T) {
	workdir := t.TempDir()
	keyName, certName := generateClientFile(t, workdir)

	TLSClientCert, err := os.ReadFile(workdir + "/" + certName)
	require.NoError(t, err)
	TLSClientKey, err := os.ReadFile(workdir + "/" + keyName)
	require.NoError(t, err)
	TLSRootCA, err := os.ReadFile("../../tests/http/tls/digicert-root-ca.crt")
	require.NoError(t, err)
	e := Executor{
		TLSClientCert: string(TLSClientCert),
		TLSClientKey:  string(TLSClientKey),
		TLSRootCA:     string(TLSRootCA),
	}
	opts, err := e.TLSOptions(ctxWithWorkdir(workdir))
	require.NoError(t, err)
	require.Len(t, opts, 2)
}

func TestExecutor_TLSOptions_RejectAbsolutePath(t *testing.T) {
	e := Executor{
		TLSRootCA: "/etc/ssl/certs/ca-certificates.crt",
	}
	_, err := e.TLSOptions(ctxWithWorkdir(t.TempDir()))
	require.Error(t, err)
}

func TestInterpolation_Of_String(t *testing.T) {
	e := &Executor{
		Method:           "",
		URL:              "http://example.com",
		Path:             "",
		BodyFile:         "tests/http/bodyfile_with_interpolation",
		PreserveBodyFile: false,
		MultipartForm:    nil,
		Headers:          map[string]string{},
	}
	ctx := context.Background()
	keys := make(map[string]string)
	keys["fullName"] = "{{.name}} test"
	keys["name"] = "123"

	ctx = context.WithValue(ctx, venom.ContextKey("vars"), []string{"fullName", "name"})
	for k := range keys {
		ctx = context.WithValue(ctx, venom.ContextKey(fmt.Sprintf("var.%s", k)), keys[k])
	}
	vars := venom.AllVarsFromCtx(ctx)
	fmt.Println("vars: ", vars)
	require.Len(t, vars, 2)
	r, err := e.getRequest(ctx, "../../")
	require.NoError(t, err)
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	fmt.Printf("Output")
	fmt.Println(string(b))
	require.Equal(t, "{\n    \"key\": \"123 test\"\n}", string(b))
}

func TestInterpolation_without_match_Of_String(t *testing.T) {
	e := &Executor{
		Method:           "",
		URL:              "http://example.com",
		Path:             "",
		BodyFile:         "tests/http/bodyfile_with_interpolation",
		PreserveBodyFile: false,
		MultipartForm:    nil,
		Headers:          map[string]string{},
	}
	ctx := context.Background()
	keys := make(map[string]string)
	keys["fullName"] = "{{.name}} test"

	ctx = context.WithValue(ctx, venom.ContextKey("vars"), []string{"fullName"})
	for k := range keys {
		ctx = context.WithValue(ctx, venom.ContextKey(fmt.Sprintf("var.%s", k)), keys[k])
	}

	_, err := e.getRequest(ctx, "../../")
	require.Errorf(t, err, "unable to interpolate file due to unresolved variables {{.name}}")
}

func TestCookieRedirect(t *testing.T) {
	callCount := atomic.Int32{}
	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /set", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "some-cookie",
			Value:    "some-value",
			Path:     "/",
			MaxAge:   100,
			Secure:   false,
			HttpOnly: true,
		})
		w.Header().Set("Location", "/get")
		w.WriteHeader(http.StatusSeeOther)
	})

	mux.HandleFunc("GET /get", func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		cookie, err := r.Cookie("some-cookie")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if cookie.Value != "some-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	e := &Executor{}
	res, err := e.Run(ctx, venom.TestStep{
		"method": http.MethodGet,
		"url":    srv.URL,
		"path":   "/set",
	})
	require.NoError(t, err)

	result, ok := res.(Result)
	require.True(t, ok)

	require.Equal(t, result.StatusCode, http.StatusOK)

	require.Equal(t, int32(1), callCount.Load())
}
