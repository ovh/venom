package venom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanUpSecrets(t *testing.T) {
	v := New()
	ts := TestSuite{
		Name:    "HTTP Auth Test",
		Secrets: []string{"basic_auth_password"},
		Vars: H{
			"url":                 "http://127.0.0.1:8000",
			"basic_auth_user":     "testuser",
			"basic_auth_password": "my_secret",
		},
		TestCases: []TestCase{
			{
				TestCaseInput: TestCaseInput{
					Name: "GET with credentials",
					Vars: H{
						"basic_auth_password": "my_secret",
					},
				},
				TestStepResults: []TestStepResult{
					{
						Name: "GET-with-credentials",
						InputVars: map[string]string{
							"basic_auth_user":     "testuser",
							"basic_auth_password": "my_secret",
						},
						Raw: []byte(`type: http
basic_auth_password: "{{.basic_auth_password}}"
`),
						Interpolated: []byte(`type: http
basic_auth_password: my_secret
basic_auth_user: testuser
method: GET
`),
						Systemout: "Authorization: Basic dGVzdHVzZXI6bXlfc2VjcmV0",
					},
				},
			},
		},
	}

	cleaned := v.CleanUpSecrets(ts)

	assert.Equal(t, "__hidden__", cleaned.Vars["basic_auth_password"])
	assert.Equal(t, "testuser", cleaned.Vars["basic_auth_user"])

	result := cleaned.TestCases[0].TestStepResults[0]
	assert.Equal(t, "__hidden__", result.InputVars["basic_auth_password"])
	assert.NotContains(t, string(result.Raw.([]byte)), "my_secret")
	assert.NotContains(t, string(result.Interpolated.([]byte)), "my_secret")
	assert.NotContains(t, result.Systemout, "my_secret")
	assert.NotContains(t, result.Systemout, "dGVzdHVzZXI6bXlfc2VjcmV0")

	data, err := json.Marshal(cleaned)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "my_secret")
}

func TestHideSensitiveBytes(t *testing.T) {
	ctx := context.WithValue(context.Background(), ContextKey("secrets"), []string{"my_secret"})

	redacted := hideSensitiveBytes(ctx, []byte("basic_auth_password: my_secret\n"))
	assert.Equal(t, "basic_auth_password: __hidden__\n", string(redacted))
}

func TestReplaceSecretsLongestFirst(t *testing.T) {
	secrets := []string{"foo", "foobar"}
	assert.Equal(t, "__hidden__", replaceSecrets("foobar", secrets))
}

func TestAppendDerivedSecretsBasicAuth(t *testing.T) {
	v := New()
	ts := TestSuite{
		Secrets: []string{"basic_auth_password"},
		Vars: H{
			"basic_auth_user":     "testuser",
			"basic_auth_password": "my_secret",
		},
	}
	ctx := v.processSecrets(context.Background(), &ts, nil)
	secrets := ctx.Value(ContextKey("secrets")).([]string)

	token := base64.StdEncoding.EncodeToString([]byte("testuser:my_secret"))
	found := false
	for _, s := range secrets {
		if s == token {
			found = true
			break
		}
	}
	assert.True(t, found, "basic auth token should be registered as a derived secret")
	assert.Equal(t, "__hidden__", HideSensitive(ctx, token))
}

func TestAppendDerivedSecretsFromTestCaseVars(t *testing.T) {
	v := New()
	ts := TestSuite{
		Secrets: []string{"basic_auth_password"},
		Vars:    H{},
		TestCases: []TestCase{
			{
				TestCaseInput: TestCaseInput{
					Vars: H{
						"basic_auth_user":     "testuser",
						"basic_auth_password": "my_secret",
					},
				},
			},
		},
	}
	ctx := v.processSecrets(context.Background(), &ts, &ts.TestCases[0])

	token := base64.StdEncoding.EncodeToString([]byte("testuser:my_secret"))
	assert.Equal(t, "__hidden__", HideSensitive(ctx, token))
	assert.Equal(t, "__hidden__", HideSensitive(ctx, base64.StdEncoding.EncodeToString([]byte("my_secret"))))
	assert.NotContains(t, HideSensitive(ctx, "Authorization: Basic "+token), token)
}

func TestHideSensitiveBasicAuthHeaderInCleanupContext(t *testing.T) {
	v := New()
	ts := TestSuite{
		Secrets: []string{"basic_auth_password"},
		Vars: H{
			"basic_auth_user":     "testuser",
			"basic_auth_password": "my_secret",
		},
		TestCases: []TestCase{
			{TestCaseInput: TestCaseInput{Vars: H{"basic_auth_password": "my_secret"}}},
		},
	}
	ctx := v.processSecrets(context.Background(), &ts, &ts.TestCases[0])
	out := HideSensitive(ctx, "Authorization: Basic dGVzdHVzZXI6bXlfc2VjcmV0")
	assert.NotContains(t, out, "dGVzdHVzZXI6bXlfc2VjcmV0")
}

func TestProcessSecretsUsesTestSuiteVars(t *testing.T) {
	v := New()
	ts := TestSuite{
		Secrets: []string{"token"},
		Vars: H{
			"token": "secret-value",
		},
	}

	ctx := v.processSecrets(context.Background(), &ts, nil)
	assert.Equal(t, "__hidden__", HideSensitive(ctx, "secret-value"))
}

// TestProcessSecretsIgnoresEmptyAndNilValues guards against a bug where a secret
// declared but left unset (nil) or empty was added to the secrets list. An empty
// value made strings.ReplaceAll insert "__hidden__" between every character, and
// a "<nil>" value redacted every literal "<nil>" in the output.
func TestProcessSecretsIgnoresEmptyAndNilValues(t *testing.T) {
	v := New()
	ts := TestSuite{
		Secrets: []string{"client_secret", "empty_secret", "real_secret"},
		Vars: H{
			"client_secret": nil,
			"empty_secret":  "",
			"real_secret":   "secret-value",
		},
	}

	ctx := v.processSecrets(context.Background(), &ts, nil)
	secrets := ctx.Value(ContextKey("secrets")).([]string)
	for _, s := range secrets {
		assert.NotEqual(t, "", s, "empty secret value must not be registered")
		assert.NotEqual(t, "<nil>", s, "nil secret value must not be registered")
	}

	content := `{"assertions":["result.statuscode ShouldEqual 200"]}`
	assert.Equal(t, content, HideSensitive(ctx, content), "no char-by-char redaction should occur")
	assert.Equal(t, "value is <nil>", HideSensitive(ctx, "value is <nil>"))
	assert.Equal(t, "__hidden__", HideSensitive(ctx, "secret-value"))
}
