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

func TestLogFunctionsPreserveNonStringFormatting(t *testing.T) {
	InitTestLogger(t)
	ctx := context.WithValue(context.Background(), ContextKey("secrets"), []string{"my_secret"})

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)
	logger.Logger.SetLevel(logrus.DebugLevel)
	defer logger.Logger.SetOutput(os.Stdout)

	Info(ctx, "Step #%d-%d content is: %s", 3, 7, "my_secret")
	out := buf.String()
	assert.NotContains(t, out, "%!d")
	assert.Contains(t, out, "Step #3-7 content is: __hidden__")

	buf.Reset()
	Debug(ctx, "count=%d ratio=%.2f ok=%t", 42, 1.5, true)
	out = buf.String()
	assert.NotContains(t, out, "%!")
	assert.Contains(t, out, "count=42 ratio=1.50 ok=true")
}
