package venom

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExecutorResult(t *testing.T) {
	payload := map[string]string{}
	payload["key"] = "{\"name\":\"test\"}"
	os.Setenv(LAZY_JSON_EXPANSION_FLAG, FLAG_ENABLED)
	result := GetExecutorResult(payload)
	assert.NotNil(t, result)
	assert.Equal(t, "Map", result["__type__"])
	assert.Equal(t, int64(1), result["__len__"])
	assert.Equal(t, "test", result["keyjson.name"])
	os.Unsetenv(LAZY_JSON_EXPANSION_FLAG)
}
