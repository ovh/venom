package venom

import (
	"context"
	"testing"

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
