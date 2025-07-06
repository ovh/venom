package venom

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteEscapingLogic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON with escaped quotes should be properly escaped",
			input:    `{"errors":[{"message":"ERROR: conflicting key value violates exclusion constraint \"age_group_id_range_unique\" (SQLSTATE 23P01)","path":["ageGroup"]}],"data":null}`,
			expected: `{\"errors\":[{\"message\":\"ERROR: conflicting key value violates exclusion constraint \\\"age_group_id_range_unique\\\" (SQLSTATE 23P01)\",\"path\":[\"ageGroup\"]}],\"data\":null}`,
		},
		{
			name:     "Simple quoted text should be escaped",
			input:    `simple "quoted" text`,
			expected: `simple \"quoted\" text`,
		},
		{
			name:     "Text without quotes should remain unchanged",
			input:    `no quotes here`,
			expected: `no quotes here`,
		},
		{
			name:     "Empty string should remain unchanged",
			input:    ``,
			expected: ``,
		},
		{
			name:     "Complex JSON response with multiple nested quotes",
			input:    `{"data":{"user":{"name":"John \"Johnny\" Doe","description":"A user with \"special\" characters"}}}`,
			expected: `{\"data\":{\"user\":{\"name\":\"John \\\"Johnny\\\" Doe\",\"description\":\"A user with \\\"special\\\" characters\"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeQuotes(tt.input)
			assert.Equal(t, tt.expected, result, "Quote escaping should work correctly")
		})
	}
}

func TestUserExecutorQuoteEscaping(t *testing.T) {
	InitTestLogger(t)

	// Test that our fix works in the context of a user executor
	tests := []struct {
		name           string
		computedVars   map[string]string
		expectedResult map[string]string
	}{
		{
			name: "GraphQL response with escaped quotes",
			computedVars: map[string]string{
				"body":       `{"errors":[{"message":"ERROR: conflicting key value violates exclusion constraint \"age_group_id_range_unique\" (SQLSTATE 23P01)","path":["ageGroup"]}],"data":null}`,
				"statuscode": "200",
				"headers":    `{"Content-Type":"application/json"}`,
			},
			expectedResult: map[string]string{
				"body":       `{\"errors\":[{\"message\":\"ERROR: conflicting key value violates exclusion constraint \\\"age_group_id_range_unique\\\" (SQLSTATE 23P01)\",\"path\":[\"ageGroup\"]}],\"data\":null}`,
				"statuscode": "200",
				"headers":    `{\"Content-Type\":\"application/json\"}`,
			},
		},
		{
			name: "Variables without quotes should remain unchanged",
			computedVars: map[string]string{
				"statuscode": "200",
				"simple":     "no quotes here",
			},
			expectedResult: map[string]string{
				"statuscode": "200",
				"simple":     "no quotes here",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same quote escaping logic as in types_executor.go
			result := make(map[string]string)
			for k, v := range tt.computedVars {
				result[k] = escapeQuotes(v)
			}

			assert.Equal(t, tt.expectedResult, result, "User executor quote escaping should work correctly")
		})
	}
}

func TestParseTestCaseQuoteEscaping(t *testing.T) {
	InitTestLogger(t)

	// Test that our fix works in the context of parseTestCase
	tests := []struct {
		name           string
		dvars          map[string]string
		expectedResult map[string]string
	}{
		{
			name: "Test case variables with quotes",
			dvars: map[string]string{
				"json_response": `{"message":"Error with \"quotes\" in message"}`,
				"simple_var":    "no quotes",
			},
			expectedResult: map[string]string{
				"json_response": `{\"message\":\"Error with \\\"quotes\\\" in message\"}`,
				"simple_var":    "no quotes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same quote escaping logic as in process_testcase.go
			result := make(map[string]string)
			for k, v := range tt.dvars {
				result[k] = escapeQuotes(v)
			}

			assert.Equal(t, tt.expectedResult, result, "Parse test case quote escaping should work correctly")
		})
	}
}

func TestUserExecutorYAMLUnmarshalWithQuotedJSON(t *testing.T) {
	InitTestLogger(t)

	// Test that our quote escaping fix actually allows YAML unmarshalling to succeed
	// This simulates the scenario where a GraphQL response causes YAML parsing errors
	tests := []struct {
		name          string
		jsonResponse  string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "GraphQL error response with escaped quotes",
			jsonResponse:  `{"errors":[{"message":"ERROR: conflicting key value violates exclusion constraint \"age_group_id_range_unique\" (SQLSTATE 23P01)","path":["ageGroup"]}],"data":null}`,
			shouldSucceed: true,
			description:   "Should successfully parse GraphQL error response with escaped quotes",
		},
		{
			name:          "Simple JSON response",
			jsonResponse:  `{"data":{"message":"success"}}`,
			shouldSucceed: true,
			description:   "Should successfully parse simple JSON response",
		},
		{
			name:          "JSON with nested quotes",
			jsonResponse:  `{"user":{"name":"John \"Johnny\" Doe"}}`,
			shouldSucceed: true,
			description:   "Should successfully parse JSON with nested quotes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the YAML content that would be created in user executor output processing
			computedVars := map[string]string{
				"body": tt.jsonResponse,
			}

			computedVars["body"] = escapeQuotes(computedVars["body"])

			// Create a sample output template like what would be in a user executor
			outputTemplate := `{"result":{"body":"{{.body}}"}}`

			// Simulate interpolation (simplified version)
			interpolated := strings.ReplaceAll(outputTemplate, "{{.body}}", computedVars["body"])

			// Try to unmarshal as JSON first to validate structure
			var jsonResult interface{}
			err := json.Unmarshal([]byte(interpolated), &jsonResult)
			if tt.shouldSucceed {
				assert.NoError(t, err, "JSON unmarshalling should succeed: %s", tt.description)
			}

			// Then try YAML unmarshalling (which is what actually happens in Venom)
			var yamlResult interface{}
			err = yaml.Unmarshal([]byte(interpolated), &yamlResult)
			if tt.shouldSucceed {
				assert.NoError(t, err, "YAML unmarshalling should succeed: %s", tt.description)
				assert.NotNil(t, yamlResult, "YAML result should not be nil")
			}
		})
	}
}

// TestRegressionGraphQLQuoteEscaping tests the specific scenario that was failing
// before our fix - GraphQL responses with escaped quotes in error messages
func TestRegressionGraphQLQuoteEscaping(t *testing.T) {
	InitTestLogger(t)

	// This is the exact GraphQL response that was causing the issue
	problematicResponse := `{"errors":[{"message":"ERROR: conflicting key value violates exclusion constraint \"age_group_id_range_unique\" (SQLSTATE 23P01)","path":["ageGroup"]}],"data":null}`

	t.Run("problematic GraphQL response should not cause YAML errors", func(t *testing.T) {
		// Apply our quote escaping fix
		escapedResponse := escapeQuotes(problematicResponse)

		// Create YAML content similar to what would be generated in user executor
		yamlContent := `
result:
  body: "` + escapedResponse + `"
  statuscode: 200
`

		// This should not fail with our fix
		var result interface{}
		err := yaml.Unmarshal([]byte(yamlContent), &result)
		require.NoError(t, err, "YAML unmarshalling should succeed with properly escaped quotes")

		// Verify the structure is correct

		resultMap, ok := result.(map[interface{}]interface{})
		if !ok {
			// Try string map instead
			if strMap, ok := result.(map[string]interface{}); ok {
				resultData, ok := strMap["result"].(map[string]interface{})
				require.True(t, ok, "Result data should be a map")
				body, ok := resultData["body"].(string)
				require.True(t, ok, "Body should be a string")
				assert.Contains(t, body, "age_group_id_range_unique", "Body should contain the original content")
				return
			}
			require.True(t, ok, "Result should be a map, got: %T", result)
		}

		resultData, ok := resultMap["result"].(map[interface{}]interface{})
		require.True(t, ok, "Result data should be a map")

		body, ok := resultData["body"].(string)
		require.True(t, ok, "Body should be a string")
		assert.Contains(t, body, "age_group_id_range_unique", "Body should contain the original content")
	})
}
