package venom

import (
	"encoding/json"
	"testing"

	"github.com/ovh/venom/interpolate"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserExecutorOutputInterpolationCamelCase reproduces the bug where user-executor
// output leaves camelCase templates uninterpolated when vars are dumped with DumpString.
// Root cause: DumpString lowercases keys before interpolate.Do, but templates keep original casing.
func TestUserExecutorOutputInterpolationCamelCase(t *testing.T) {
	computedVars := H{
		"session_id":         "sess-123",
		"revision":           "rev-1",
		"deploymentLocator":  "loc-abc-xyz",
	}

	type Output struct {
		Result json.RawMessage `json:"result"`
	}
	output := Output{
		Result: json.RawMessage(`{
  "session_id": "{{.session_id}}",
  "revision": "{{.revision}}",
  "deploymentLocator": "{{.deploymentLocator}}"
}`),
	}

	outputString, err := json.Marshal(output)
	require.NoError(t, err)

	t.Run("DumpString breaks camelCase output interpolation", func(t *testing.T) {
		vars, err := DumpString(computedVars)
		require.NoError(t, err)
		for k := range vars {
			vars[k] = escapeQuotes(vars[k])
		}

		got, err := interpolate.Do(string(outputString), vars)
		require.NoError(t, err)

		var result map[string]interface{}
		require.NoError(t, yaml.Unmarshal([]byte(got), &result))
		inner := result["result"].(map[string]interface{})

		assert.Equal(t, "sess-123", inner["session_id"])
		assert.Equal(t, "rev-1", inner["revision"])
		assert.Equal(t, "{{.deploymentLocator}}", inner["deploymentLocator"])
	})

	t.Run("DumpStringPreserveCase resolves camelCase output interpolation", func(t *testing.T) {
		vars, err := DumpStringPreserveCase(computedVars)
		require.NoError(t, err)
		for k := range vars {
			vars[k] = escapeQuotes(vars[k])
		}

		got, err := interpolate.Do(string(outputString), vars)
		require.NoError(t, err)

		var result map[string]interface{}
		require.NoError(t, yaml.Unmarshal([]byte(got), &result))
		inner := result["result"].(map[string]interface{})

		assert.Equal(t, "sess-123", inner["session_id"])
		assert.Equal(t, "rev-1", inner["revision"])
		assert.Equal(t, "loc-abc-xyz", inner["deploymentLocator"])
	})
}
