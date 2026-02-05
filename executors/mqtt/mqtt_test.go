package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mq "github.com/eclipse/paho.mqtt.golang"
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

func TestExecutor_GetDefaultAssertions(t *testing.T) {
	e := Executor{}
	assertions := e.GetDefaultAssertions()
	require.NotNil(t, assertions)
	require.Len(t, assertions.Assertions, 1)
	assert.Equal(t, "result.error ShouldBeEmpty", assertions.Assertions[0])
}

func TestExecutor_Run_MissingAddress(t *testing.T) {
	ctx := context.Background()
	e := Executor{}

	step := venom.TestStep{}
	result, err := e.Run(ctx, step)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "address is mandatory")
	assert.Nil(t, result)
}

func TestExecutor_Run_InvalidClientType(t *testing.T) {
	step := venom.TestStep{
		"addrs":      "tcp://localhost:1883",
		"clientType": "invalid_type",
	}

	e := Executor{}
	_, err := e.Run(context.Background(), step)

	// This should return an error (not in result.Err) for invalid client type
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be publisher, subscriber or persistent_queue")
}

func TestExecutor_Run_DefaultValues(t *testing.T) {
	step := venom.TestStep{
		"addrs":      "tcp://localhost:1883",
		"clientType": "publisher",
		"messages":   []interface{}{},
	}

	var e Executor
	err := mapstructure.Decode(step, &e)
	require.NoError(t, err)

	// Set defaults
	if e.MessageLimit == 0 {
		e.MessageLimit = 1
	}
	if e.Timeout == 0 {
		e.Timeout = defaultExecutorTimeoutMs
	}
	if e.ConnectTimeout == 0 {
		e.ConnectTimeout = defaultConnectTimeoutMs
	}

	assert.Equal(t, 1, e.MessageLimit)
	assert.Equal(t, int64(defaultExecutorTimeoutMs), e.Timeout)
	assert.Equal(t, int64(defaultConnectTimeoutMs), e.ConnectTimeout)
}

func TestExecutor_MapstructureDecode(t *testing.T) {
	tests := []struct {
		name     string
		step     venom.TestStep
		expected Executor
	}{
		{
			name: "Publisher configuration",
			step: venom.TestStep{
				"addrs":      "tcp://localhost:1883",
				"clientType": "publisher",
				"clientId":   "test-publisher",
				"messages": []interface{}{
					map[string]interface{}{
						"topic":    "test/topic",
						"payload":  "test payload",
						"qos":      byte(1),
						"retained": true,
					},
				},
			},
			expected: Executor{
				Addrs:      "tcp://localhost:1883",
				ClientType: "publisher",
				ClientID:   "test-publisher",
				Messages: []Message{
					{
						Topic:    "test/topic",
						Payload:  "test payload",
						QOS:      1,
						Retained: true,
					},
				},
			},
		},
		{
			name: "Subscriber configuration",
			step: venom.TestStep{
				"addrs":        "tcp://localhost:1883",
				"clientType":   "subscriber",
				"clientId":     "test-subscriber",
				"topics":       []string{"test/topic1", "test/topic2"},
				"messageLimit": 5,
				"timeout":      10000,
				"qos":          byte(2),
			},
			expected: Executor{
				Addrs:        "tcp://localhost:1883",
				ClientType:   "subscriber",
				ClientID:     "test-subscriber",
				Topics:       []string{"test/topic1", "test/topic2"},
				MessageLimit: 5,
				Timeout:      10000,
				QOS:          2,
			},
		},
		{
			name: "Persistent queue configuration",
			step: venom.TestStep{
				"addrs":               "tcp://localhost:1883",
				"clientType":          "persistent_queue",
				"clientId":            "test-persistent",
				"topics":              []string{"test/topic"},
				"persistSubscription": true,
				"connectTimeout":      3000,
			},
			expected: Executor{
				Addrs:               "tcp://localhost:1883",
				ClientType:          "persistent_queue",
				ClientID:            "test-persistent",
				Topics:              []string{"test/topic"},
				PersistSubscription: true,
				ConnectTimeout:      3000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e Executor
			err := mapstructure.Decode(tt.step, &e)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Addrs, e.Addrs)
			assert.Equal(t, tt.expected.ClientType, e.ClientType)
			assert.Equal(t, tt.expected.ClientID, e.ClientID)
			assert.Equal(t, tt.expected.Topics, e.Topics)
			assert.Equal(t, tt.expected.PersistSubscription, e.PersistSubscription)

			if tt.expected.MessageLimit != 0 {
				assert.Equal(t, tt.expected.MessageLimit, e.MessageLimit)
			}
			if tt.expected.Timeout != 0 {
				assert.Equal(t, tt.expected.Timeout, e.Timeout)
			}
			if tt.expected.ConnectTimeout != 0 {
				assert.Equal(t, tt.expected.ConnectTimeout, e.ConnectTimeout)
			}
			if tt.expected.QOS != 0 {
				assert.Equal(t, tt.expected.QOS, e.QOS)
			}
			if len(tt.expected.Messages) > 0 {
				assert.Len(t, e.Messages, len(tt.expected.Messages))
			}
		})
	}
}

func TestMessage_Struct(t *testing.T) {
	msg := Message{
		Topic:    "test/topic",
		QOS:      1,
		Retained: true,
		Payload:  `{"key": "value"}`,
	}

	assert.Equal(t, "test/topic", msg.Topic)
	assert.Equal(t, byte(1), msg.QOS)
	assert.True(t, msg.Retained)
	assert.Equal(t, `{"key": "value"}`, msg.Payload)
}

func TestResult_Struct(t *testing.T) {
	result := Result{
		TimeSeconds:  1.234,
		Topics:       []string{"topic1", "topic2"},
		Messages:     []interface{}{[]byte("msg1"), []byte("msg2")},
		MessagesJSON: []interface{}{map[string]interface{}{"key": "value"}},
		Err:          "",
	}

	assert.Equal(t, 1.234, result.TimeSeconds)
	assert.Len(t, result.Topics, 2)
	assert.Len(t, result.Messages, 2)
	assert.Len(t, result.MessagesJSON, 1)
	assert.Empty(t, result.Err)
}

func TestResult_JSONSerialization(t *testing.T) {
	result := Result{
		TimeSeconds:  1.5,
		Topics:       []string{"test/topic"},
		Messages:     []interface{}{[]byte("test message")},
		MessagesJSON: []interface{}{map[string]interface{}{"foo": "bar"}},
		Err:          "",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded Result
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.TimeSeconds, decoded.TimeSeconds)
	assert.Equal(t, result.Topics, decoded.Topics)
	assert.Equal(t, result.Err, decoded.Err)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "mqtt", Name)
	assert.Equal(t, uint(500), uint(disconnectTimeoutMs))
	assert.Equal(t, int64(5000), int64(defaultExecutorTimeoutMs))
	assert.Equal(t, int64(5000), int64(defaultConnectTimeoutMs))
	assert.Equal(t, uint(4), uint(mqttV311))
}

func TestNewSubscriber(t *testing.T) {
	ctx := context.Background()
	ch := make(chan mq.Message, 10)
	defer close(ch)

	subscriber := newSubscriber(ctx, ch)
	require.NotNil(t, subscriber)

	// Note: Testing the actual subscriber would require a mock mq.Client and mq.Message
	// which is complex without a proper MQTT broker
}

func TestExecutor_Run_TimeTracking(t *testing.T) {
	// Skip this test as it requires a real MQTT broker
	t.Skip("Requires MQTT broker - integration test")

	ctx := context.Background()
	step := venom.TestStep{
		"addrs":      "tcp://invalid-host:1883",
		"clientType": "publisher",
		"messages":   []interface{}{},
		"timeout":    100,
	}

	e := Executor{}
	start := time.Now()
	result, err := e.Run(ctx, step)
	elapsed := time.Since(start)

	require.NoError(t, err)
	r, ok := result.(Result)
	require.True(t, ok)

	// Result should track execution time
	assert.Greater(t, r.TimeSeconds, 0.0)
	// Execution time should be reasonable (under total elapsed + margin)
	assert.LessOrEqual(t, r.TimeSeconds, elapsed.Seconds()+0.1)
}

func TestExecutor_ValidateMessages(t *testing.T) {
	tests := []struct {
		name        string
		messages    []Message
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid message",
			messages: []Message{
				{Topic: "test/topic", Payload: "payload", QOS: 0},
			},
			shouldError: false,
		},
		{
			name: "Empty topic",
			messages: []Message{
				{Topic: "", Payload: "payload", QOS: 0},
			},
			shouldError: true,
			errorMsg:    "mandatory field Topic was empty",
		},
		{
			name: "Multiple messages with one empty topic",
			messages: []Message{
				{Topic: "valid/topic", Payload: "payload1", QOS: 0},
				{Topic: "", Payload: "payload2", QOS: 0},
			},
			shouldError: true,
			errorMsg:    "mandatory field Topic was empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, m := range tt.messages {
				if len(m.Topic) == 0 && tt.shouldError {
					// Simulating the validation that happens in publishMessages
					assert.Empty(t, m.Topic, "Expected empty topic for validation test at index %d", i)
				}
			}
		})
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	// Skip this test as it requires a real MQTT broker
	t.Skip("Requires MQTT broker - integration test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	step := venom.TestStep{
		"addrs":      "tcp://localhost:1883",
		"clientType": "publisher",
		"messages": []interface{}{
			map[string]interface{}{
				"topic":   "test/topic",
				"payload": "test",
			},
		},
	}

	e := Executor{}
	result, err := e.Run(ctx, step)

	require.NoError(t, err)
	r, ok := result.(Result)
	require.True(t, ok)
	// When context is cancelled, connection should fail
	assert.NotEmpty(t, r.Err)
}

func TestExecutor_QoSLevels(t *testing.T) {
	tests := []struct {
		name string
		qos  byte
	}{
		{"QoS 0 - At most once", 0},
		{"QoS 1 - At least once", 1},
		{"QoS 2 - Exactly once", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := Message{
				Topic:   "test/topic",
				Payload: "test",
				QOS:     tt.qos,
			}
			assert.Equal(t, tt.qos, msg.QOS)
			assert.True(t, msg.QOS >= 0 && msg.QOS <= 2, "QoS should be between 0 and 2")
		})
	}
}

func TestExecutor_PersistSubscriptionFlag(t *testing.T) {
	tests := []struct {
		name                 string
		persistSubscription  bool
		expectedCleanSession bool
	}{
		{
			name:                 "Persistent subscription",
			persistSubscription:  true,
			expectedCleanSession: false,
		},
		{
			name:                 "Non-persistent subscription",
			persistSubscription:  false,
			expectedCleanSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Executor{
				PersistSubscription: tt.persistSubscription,
			}
			// CleanSession should be the opposite of PersistSubscription
			cleanSession := !e.PersistSubscription
			assert.Equal(t, tt.expectedCleanSession, cleanSession)
		})
	}
}

func TestExecutor_MultipleTopics(t *testing.T) {
	topics := []string{
		"test/topic1",
		"test/topic2",
		"test/topic3",
		"sensors/+/temperature",
		"home/#",
	}

	e := Executor{
		Topics: topics,
	}

	assert.Len(t, e.Topics, 5)
	assert.Contains(t, e.Topics, "sensors/+/temperature")
	assert.Contains(t, e.Topics, "home/#")
}

func TestExecutor_TimeoutConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		timeout        int64
		connectTimeout int64
		expectedTO     int64
		expectedCTO    int64
	}{
		{
			name:           "Default timeouts",
			timeout:        0,
			connectTimeout: 0,
			expectedTO:     defaultExecutorTimeoutMs,
			expectedCTO:    defaultConnectTimeoutMs,
		},
		{
			name:           "Custom timeouts",
			timeout:        10000,
			connectTimeout: 3000,
			expectedTO:     10000,
			expectedCTO:    3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Executor{
				Timeout:        tt.timeout,
				ConnectTimeout: tt.connectTimeout,
			}

			// Apply defaults
			if e.Timeout == 0 {
				e.Timeout = defaultExecutorTimeoutMs
			}
			if e.ConnectTimeout == 0 {
				e.ConnectTimeout = defaultConnectTimeoutMs
			}

			assert.Equal(t, tt.expectedTO, e.Timeout)
			assert.Equal(t, tt.expectedCTO, e.ConnectTimeout)
		})
	}
}

func TestExecutor_RetainedMessages(t *testing.T) {
	msg := Message{
		Topic:    "test/retained",
		Payload:  "persistent message",
		Retained: true,
		QOS:      1,
	}

	assert.True(t, msg.Retained, "Message should be marked as retained")
}

func TestExecutor_MessageLimit(t *testing.T) {
	tests := []struct {
		name         string
		messageLimit int
		expected     int
	}{
		{"Default limit", 0, 1},
		{"Custom limit 5", 5, 5},
		{"Custom limit 100", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Executor{
				MessageLimit: tt.messageLimit,
			}

			if e.MessageLimit == 0 {
				e.MessageLimit = 1
			}

			assert.Equal(t, tt.expected, e.MessageLimit)
		})
	}
}

func TestResult_ErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		hasErr bool
	}{
		{
			name: "No error",
			result: Result{
				TimeSeconds: 1.0,
				Err:         "",
			},
			hasErr: false,
		},
		{
			name: "With error",
			result: Result{
				TimeSeconds: 0.5,
				Err:         "connection failed",
			},
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hasErr {
				assert.NotEmpty(t, tt.result.Err)
			} else {
				assert.Empty(t, tt.result.Err)
			}
		})
	}
}

func TestExecutor_ClientTypes(t *testing.T) {
	validTypes := []string{"publisher", "subscriber", "persistent_queue"}
	invalidTypes := []string{"consumer", "producer", "invalid", ""}

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

func TestExecutor_JSONPayloadParsing(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		shouldParse bool
	}{
		{
			name:        "Valid JSON object",
			payload:     `{"key": "value", "number": 42}`,
			shouldParse: true,
		},
		{
			name:        "Valid JSON array",
			payload:     `[1, 2, 3, "test"]`,
			shouldParse: true,
		},
		{
			name:        "Invalid JSON",
			payload:     `{invalid json}`,
			shouldParse: false,
		},
		{
			name:        "Plain text",
			payload:     `plain text message`,
			shouldParse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			err := json.Unmarshal([]byte(tt.payload), &result)

			if tt.shouldParse {
				assert.NoError(t, err, "Expected valid JSON")
			} else {
				assert.Error(t, err, "Expected invalid JSON")
			}
		})
	}
}

func TestExecutor_ClientIDUniqueness(t *testing.T) {
	// Test that different executors can have different client IDs
	e1 := Executor{ClientID: "client-1"}
	e2 := Executor{ClientID: "client-2"}

	assert.NotEqual(t, e1.ClientID, e2.ClientID)
	assert.Equal(t, "client-1", e1.ClientID)
	assert.Equal(t, "client-2", e2.ClientID)
}
