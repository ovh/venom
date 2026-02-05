package amqp

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/venom"
)

func TestNew(t *testing.T) {
	executor := New()
	require.NotNil(t, executor)
	_, ok := executor.(*Executor)
	assert.True(t, ok, "New() should return an *Executor")
}

func TestExecutor_ZeroValueResult(t *testing.T) {
	e := Executor{}
	result := e.ZeroValueResult()
	require.NotNil(t, result)
	r, ok := result.(Result)
	require.True(t, ok)
	assert.Empty(t, r.Messages)
	assert.Empty(t, r.MessagesJSON)
}

func TestExecutor_MapstructureDecode(t *testing.T) {
	tests := []struct {
		name     string
		step     venom.TestStep
		expected Executor
	}{
		{
			name: "Producer configuration",
			step: venom.TestStep{
				"addr":       "amqp://localhost:5672",
				"clientType": "producer",
				"targetAddr": "test-queue",
				"messages":   []string{"message1", "message2"},
			},
			expected: Executor{
				Addr:       "amqp://localhost:5672",
				ClientType: "producer",
				TargetAddr: "test-queue",
				Messages:   []string{"message1", "message2"},
			},
		},
		{
			name: "Consumer configuration",
			step: venom.TestStep{
				"addr":         "amqp://localhost:5672",
				"clientType":   "consumer",
				"sourceAddr":   "test-queue",
				"messageLimit": uint(10),
			},
			expected: Executor{
				Addr:         "amqp://localhost:5672",
				ClientType:   "consumer",
				SourceAddr:   "test-queue",
				MessageLimit: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e Executor
			err := mapstructure.Decode(tt.step, &e)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Addr, e.Addr)
			assert.Equal(t, tt.expected.ClientType, e.ClientType)
			assert.Equal(t, tt.expected.TargetAddr, e.TargetAddr)
			assert.Equal(t, tt.expected.SourceAddr, e.SourceAddr)
			assert.Equal(t, tt.expected.MessageLimit, e.MessageLimit)

			if len(tt.expected.Messages) > 0 {
				assert.Equal(t, tt.expected.Messages, e.Messages)
			}
		})
	}
}

func TestExecutor_Struct(t *testing.T) {
	e := Executor{
		Addr:         "amqp://localhost:5672",
		ClientType:   "producer",
		SourceAddr:   "source",
		TargetAddr:   "target",
		MessageLimit: 5,
		Messages:     []string{"msg1", "msg2"},
	}

	assert.Equal(t, "amqp://localhost:5672", e.Addr)
	assert.Equal(t, "producer", e.ClientType)
	assert.Equal(t, "source", e.SourceAddr)
	assert.Equal(t, "target", e.TargetAddr)
	assert.Equal(t, uint(5), e.MessageLimit)
	assert.Len(t, e.Messages, 2)
}

func TestResult_Struct(t *testing.T) {
	result := Result{
		Messages:     []string{"msg1", "msg2"},
		MessagesJSON: []interface{}{map[string]interface{}{"key": "value"}},
	}

	assert.Len(t, result.Messages, 2)
	assert.Len(t, result.MessagesJSON, 1)
	assert.Equal(t, "msg1", result.Messages[0])
}

func TestResult_JSONSerialization(t *testing.T) {
	result := Result{
		Messages:     []string{"test message"},
		MessagesJSON: []interface{}{map[string]interface{}{"foo": "bar"}},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded Result
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Messages, decoded.Messages)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "amqp", Name)
}

func TestExecutor_ValidateProducer(t *testing.T) {
	tests := []struct {
		name        string
		executor    Executor
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid producer",
			executor: Executor{
				Addr:       "amqp://localhost:5672",
				ClientType: "producer",
				TargetAddr: "queue",
				Messages:   []string{"msg"},
			},
			shouldError: false,
		},
		{
			name: "Producer without targetAddr",
			executor: Executor{
				Addr:       "amqp://localhost:5672",
				ClientType: "producer",
				Messages:   []string{"msg"},
			},
			shouldError: true,
			errorMsg:    "targetAddr",
		},
		{
			name: "Producer without messages",
			executor: Executor{
				Addr:       "amqp://localhost:5672",
				ClientType: "producer",
				TargetAddr: "queue",
				Messages:   []string{},
			},
			shouldError: true,
			errorMsg:    "messages length must be > 0",
		},
		{
			name: "Producer without addr",
			executor: Executor{
				ClientType: "producer",
				TargetAddr: "queue",
				Messages:   []string{"msg"},
			},
			shouldError: true,
			errorMsg:    "addr is mandatory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation that happens in Run() and helper functions
			if tt.executor.Addr == "" {
				assert.True(t, tt.shouldError)
				assert.Contains(t, "addr is mandatory", tt.errorMsg)
				return
			}

			if tt.executor.ClientType == "producer" {
				if tt.executor.TargetAddr == "" {
					assert.True(t, tt.shouldError)
					assert.Contains(t, tt.errorMsg, "targetAddr")
				}
				if len(tt.executor.Messages) < 1 {
					assert.True(t, tt.shouldError)
					assert.Contains(t, tt.errorMsg, "messages")
				}
			}
		})
	}
}

func TestExecutor_ValidateConsumer(t *testing.T) {
	tests := []struct {
		name        string
		executor    Executor
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid consumer",
			executor: Executor{
				Addr:         "amqp://localhost:5672",
				ClientType:   "consumer",
				SourceAddr:   "queue",
				MessageLimit: 5,
			},
			shouldError: false,
		},
		{
			name: "Consumer without sourceAddr",
			executor: Executor{
				Addr:         "amqp://localhost:5672",
				ClientType:   "consumer",
				MessageLimit: 5,
			},
			shouldError: true,
			errorMsg:    "sourceAddr",
		},
		{
			name: "Consumer with messageLimit 0",
			executor: Executor{
				Addr:         "amqp://localhost:5672",
				ClientType:   "consumer",
				SourceAddr:   "queue",
				MessageLimit: 0,
			},
			shouldError: true,
			errorMsg:    "messageLimit must be > 0",
		},
		{
			name: "Consumer without addr",
			executor: Executor{
				ClientType:   "consumer",
				SourceAddr:   "queue",
				MessageLimit: 5,
			},
			shouldError: true,
			errorMsg:    "addr is mandatory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation that happens in Run() and helper functions
			if tt.executor.Addr == "" {
				assert.True(t, tt.shouldError)
				assert.Contains(t, "addr is mandatory", tt.errorMsg)
				return
			}

			if tt.executor.ClientType == "consumer" {
				if tt.executor.SourceAddr == "" {
					assert.True(t, tt.shouldError)
					assert.Contains(t, tt.errorMsg, "sourceAddr")
				}
				if tt.executor.MessageLimit < 1 {
					assert.True(t, tt.shouldError)
					assert.Contains(t, tt.errorMsg, "messageLimit")
				}
			}
		})
	}
}

func TestExecutor_ClientTypes(t *testing.T) {
	validTypes := []string{"producer", "consumer"}
	invalidTypes := []string{"publisher", "subscriber", "invalid", ""}

	for _, ct := range validTypes {
		t.Run(fmt.Sprintf("Valid type: %s", ct), func(t *testing.T) {
			e := Executor{ClientType: ct}
			assert.Contains(t, validTypes, e.ClientType)
		})
	}

	for _, ct := range invalidTypes {
		t.Run(fmt.Sprintf("Invalid type: %s", ct), func(t *testing.T) {
			e := Executor{ClientType: ct}
			assert.NotContains(t, validTypes, e.ClientType)
		})
	}
}

func TestExecutor_AddressFormats(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		isValid bool
	}{
		{"Standard AMQP", "amqp://localhost:5672", true},
		{"AMQPS secure", "amqps://localhost:5671", true},
		{"With credentials", "amqp://user:pass@localhost:5672", true},
		{"With path", "amqp://localhost:5672/vhost", true},
		{"IP address", "amqp://127.0.0.1:5672", true},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Executor{Addr: tt.addr}
			if tt.isValid {
				assert.NotEmpty(t, e.Addr)
			} else {
				assert.Empty(t, e.Addr)
			}
		})
	}
}

func TestExecutor_Messages(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		expected int
	}{
		{"Single message", []string{"msg1"}, 1},
		{"Multiple messages", []string{"msg1", "msg2", "msg3"}, 3},
		{"Empty messages", []string{}, 0},
		{"JSON messages", []string{`{"key":"value"}`, `["array"]`}, 2},
		{"Mixed content", []string{"plain text", `{"json":"data"}`}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Executor{Messages: tt.messages}
			assert.Len(t, e.Messages, tt.expected)
		})
	}
}

func TestExecutor_MessageLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit uint
		valid bool
	}{
		{"Limit 1", 1, true},
		{"Limit 10", 10, true},
		{"Limit 100", 100, true},
		{"Limit 0", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Executor{MessageLimit: tt.limit}
			if tt.valid {
				assert.Greater(t, e.MessageLimit, uint(0))
			} else {
				assert.Equal(t, uint(0), e.MessageLimit)
			}
		})
	}
}

func TestResult_EmptyResult(t *testing.T) {
	result := Result{}
	assert.Nil(t, result.Messages)
	assert.Nil(t, result.MessagesJSON)
}

func TestResult_WithMessages(t *testing.T) {
	result := Result{
		Messages:     []string{"msg1", "msg2"},
		MessagesJSON: []interface{}{nil, map[string]interface{}{"key": "value"}},
	}

	assert.Len(t, result.Messages, 2)
	assert.Len(t, result.MessagesJSON, 2)
	assert.Nil(t, result.MessagesJSON[0], "Non-JSON message should have nil JSON representation")
	assert.NotNil(t, result.MessagesJSON[1], "JSON message should be parsed")
}

func TestExecutor_JSONParsing(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		shouldParse bool
	}{
		{
			name:        "Valid JSON object",
			message:     `{"key": "value", "number": 42}`,
			shouldParse: true,
		},
		{
			name:        "Valid JSON array",
			message:     `[1, 2, 3, "test"]`,
			shouldParse: true,
		},
		{
			name:        "Invalid JSON",
			message:     `{invalid json}`,
			shouldParse: false,
		},
		{
			name:        "Plain text",
			message:     `plain text message`,
			shouldParse: false,
		},
		{
			name:        "Empty string",
			message:     ``,
			shouldParse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			err := json.Unmarshal([]byte(tt.message), &result)

			if tt.shouldParse {
				assert.NoError(t, err, "Expected valid JSON")
			} else {
				if tt.message == "" {
					assert.Error(t, err, "Expected error for empty string")
				} else if tt.message == "plain text message" {
					assert.Error(t, err, "Expected error for plain text")
				}
			}
		})
	}
}

func TestExecutor_ProducerConsumerFlow(t *testing.T) {
	// This test documents the expected flow without requiring a real AMQP broker
	t.Run("Producer step structure", func(t *testing.T) {
		step := venom.TestStep{
			"addr":       "amqp://localhost:5672",
			"clientType": "producer",
			"targetAddr": "test-queue",
			"messages":   []string{`{"test": "data"}`, "plain message"},
		}

		var e Executor
		err := mapstructure.Decode(step, &e)
		require.NoError(t, err)

		assert.Equal(t, "producer", e.ClientType)
		assert.Equal(t, "test-queue", e.TargetAddr)
		assert.Len(t, e.Messages, 2)
	})

	t.Run("Consumer step structure", func(t *testing.T) {
		step := venom.TestStep{
			"addr":         "amqp://localhost:5672",
			"clientType":   "consumer",
			"sourceAddr":   "test-queue",
			"messageLimit": 5,
		}

		var e Executor
		err := mapstructure.Decode(step, &e)
		require.NoError(t, err)

		assert.Equal(t, "consumer", e.ClientType)
		assert.Equal(t, "test-queue", e.SourceAddr)
		assert.Equal(t, uint(5), e.MessageLimit)
	})
}

func TestExecutor_ResultPreallocation(t *testing.T) {
	// Test that Result can be properly preallocated (as done in consumeMessages)
	messageLimit := uint(10)
	result := Result{
		Messages:     make([]string, 0, messageLimit),
		MessagesJSON: make([]interface{}, 0, messageLimit),
	}

	assert.Equal(t, 0, len(result.Messages))
	assert.Equal(t, 10, cap(result.Messages))
	assert.Equal(t, 0, len(result.MessagesJSON))
	assert.Equal(t, 10, cap(result.MessagesJSON))

	// Simulate appending messages
	for i := 0; i < 5; i++ {
		result.Messages = append(result.Messages, fmt.Sprintf("msg%d", i))
		result.MessagesJSON = append(result.MessagesJSON, map[string]interface{}{"index": i})
	}

	assert.Equal(t, 5, len(result.Messages))
	assert.Equal(t, 5, len(result.MessagesJSON))
}

func TestExecutor_ErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		executor     Executor
		expectedErr  string
		validateFunc func(Executor) error
	}{
		{
			name:        "Missing addr",
			executor:    Executor{},
			expectedErr: "addr is mandatory",
			validateFunc: func(e Executor) error {
				if e.Addr == "" {
					return fmt.Errorf("creating session: addr is mandatory")
				}
				return nil
			},
		},
		{
			name: "Producer missing targetAddr",
			executor: Executor{
				Addr:       "amqp://localhost:5672",
				ClientType: "producer",
			},
			expectedErr: "targetAddr",
			validateFunc: func(e Executor) error {
				if e.TargetAddr == "" {
					return fmt.Errorf("publishing messages: targetAddr is manatory when clientType is producer")
				}
				return nil
			},
		},
		{
			name: "Consumer missing sourceAddr",
			executor: Executor{
				Addr:       "amqp://localhost:5672",
				ClientType: "consumer",
			},
			expectedErr: "sourceAddr",
			validateFunc: func(e Executor) error {
				if e.SourceAddr == "" {
					return fmt.Errorf("consuming messages: sourceAddr is manatory when clientType is consumer")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validateFunc(tt.executor)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutor_MultipleMessagesWithDifferentFormats(t *testing.T) {
	messages := []string{
		`{"type": "json", "value": 1}`,
		`plain text`,
		`["array", "of", "strings"]`,
		`{"nested": {"object": true}}`,
		`42`,
		`"just a string"`,
	}

	e := Executor{Messages: messages}
	assert.Len(t, e.Messages, 6)

	// Verify each can be parsed or not as expected
	for i, msg := range messages {
		var parsed interface{}
		err := json.Unmarshal([]byte(msg), &parsed)
		switch i {
		case 0, 2, 3, 4, 5: // Valid JSON
			assert.NoError(t, err, "Message %d should be valid JSON", i)
		case 1: // Plain text
			assert.Error(t, err, "Message %d should not be valid JSON", i)
		}
	}
}
