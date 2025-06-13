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
