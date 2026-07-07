package venom

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHideSensitive(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("secrets"), []string{"Joe", "Doe"})
	assert.Equal(t, "__hidden__", HideSensitive(ctx, "Joe"))
	assert.Equal(t, "__hidden__ tests something", HideSensitive(ctx, "Joe tests something"))
	assert.Equal(t, "Dave tests something", HideSensitive(ctx, "Dave tests something"))
	assert.Equal(t, "1234", HideSensitive(ctx, 1234))
	assert.Equal(t, "__hidden__!", HideSensitive(ctx, "Doe!"))
	assert.Equal(t, "__hidden__ __hidden__", HideSensitive(ctx, "Joe Doe"))
}

func TestLogFunctionsRedactSecrets(t *testing.T) {
	InitTestLogger(t)
	ctx := context.WithValue(context.Background(), ContextKey("secrets"), []string{"my_secret"})

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)
	logger.Logger.SetLevel(logrus.DebugLevel)
	defer logger.Logger.SetOutput(os.Stdout)

	Info(ctx, "step content: basic_auth_password: %s", "my_secret")
	assert.NotContains(t, buf.String(), "my_secret")
	assert.Contains(t, buf.String(), "__hidden__")

	buf.Reset()
	Debug(ctx, "with vars: %v", H{"basic_auth_password": "my_secret"})
	assert.NotContains(t, buf.String(), "my_secret")
	assert.Contains(t, buf.String(), "__hidden__")

	buf.Reset()
	Error(ctx, "interpolated step:\n%s", "basic_auth_password: my_secret\n")
	assert.NotContains(t, buf.String(), "my_secret")
	assert.Contains(t, buf.String(), "__hidden__")
}
