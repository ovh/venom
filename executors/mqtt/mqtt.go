package mqtt

import (
	"context"
	"fmt"
	"time"

	mq "github.com/eclipse/paho.mqtt.golang"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	"github.com/pkg/errors"
)

// Name of executor
const Name = "mqtt"

const disconnectTimeoutMs = 500
const defaultExecutorTimeoutMs = 5000
const defaultConnectTimeoutMs = 5000
const mqttV311 = 4

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

type Executor struct {
	Addrs string `json:"addrs" yaml:"addrs"`

	// ClientType must be "consumer", "producer" or "persistent_queue"
	ClientType          string `json:"client_type" yaml:"clientType"`
	PersistSubscription bool   `json:"persist_subscription" yaml:"persistSubscription"`
	ClientID            string `json:"client_id" yaml:"clientId"`

	// Subscription topic
	Topics []string `json:"topics" yaml:"topics"`

	// Represents the limit of message will be read. After limit, consumer stop read message
	MessageLimit int `json:"message_limit" yaml:"messageLimit"`

	// Represents the mqtt connection timeout for reading messages. In Milliseconds. Default 5000
	ConnectTimeout int64 `json:"connect_timeout,omitempty" yaml:"connectTimeout,omitempty"`

	// Represents the timeout for reading messages. In Milliseconds. Default 5000
	Timeout int64 `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Used when ClientType is producer
	// Messages represents the message sent by producer
	Messages []Message `json:"messages" yaml:"messages"`
	QOS      byte      `json:"qos" yaml:"qos"`
}

// Message represents the object sent or received from rabbitmq
type Message struct {
	Topic    string `json:"topic" yaml:"topic"`
	QOS      byte   `json:"qos" yaml:"qos"`
	Retained bool   `json:"retained" yaml:"retained"`
	Payload  string `json:"payload" yaml:"payload"`
}

// Result represents a step result.
type Result struct {
	TimeSeconds  float64       `json:"timeseconds" yaml:"timeSeconds"`
	Topics       []string      `json:"topics" yaml:"topics"`
	Messages     []interface{} `json:"messages" yaml:"messages"`
	MessagesJSON []interface{} `json:"messagesjson" yaml:"messagesJSON"`
	Err          string        `json:"err" yaml:"error"`
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []venom.Assertion{"result.error ShouldBeEmpty"}}
}

func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	start := time.Now()
	result := Result{}

	// Default values
	if e.Addrs == "" {
		return nil, errors.New("address is mandatory")
	}
	if e.MessageLimit == 0 {
		e.MessageLimit = 1
	}
	if e.Timeout == 0 {
		e.Timeout = defaultExecutorTimeoutMs
	}
	if e.ConnectTimeout == 0 {
		e.ConnectTimeout = defaultConnectTimeoutMs
	}

	var err error
	switch e.ClientType {
	case "publisher":
		err = e.publishMessages(ctx)
		if err != nil {
			result.Err = err.Error()
		}
	case "persistent_queue":
		err = e.persistMessages(ctx)
		if err != nil {
			result.Err = err.Error()
		}
	case "subscriber":
		result.Messages, result.MessagesJSON, result.Topics, err = e.consumeMessages(ctx)
		if err != nil {
			result.Err = err.Error()
		}
	default:
		return nil, fmt.Errorf("clientType %q must be publisher, subscriber or persistent_queue", e.ClientType)
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

// session prepares a client connection returning a client and a possible error
func (e Executor) session(ctx context.Context, subscriber func(client mq.Client, message mq.Message)) (mq.Client, error) {
	venom.Debug(ctx, "creating session to %v, cleanSession: %v, clientID: %v", e.Addrs, !e.PersistSubscription, e.ClientID)

	opts := mq.NewClientOptions().
		AddBroker(e.Addrs).
		SetConnectTimeout(time.Duration(e.ConnectTimeout) * time.Millisecond).
		SetCleanSession(!e.PersistSubscription).
		SetClientID(e.ClientID).
		SetProtocolVersion(mqttV311).
		SetOnConnectHandler(func(client mq.Client) {
			venom.Debug(ctx, "connection handler called. IsConnected: %v", client.IsConnected())
		})

	client := mq.NewClient(opts)

	// MQTT may send messages prior to a subscription taking place (due to pre-existing persistent session).
	// We cannot subscribe without a connection so we register a route and subscribe later
	if subscriber != nil {
		venom.Debug(ctx, "adding routes: %v", e.Topics)
		for _, topic := range e.Topics {
			client.AddRoute(topic, subscriber)
		}
	}

	token := client.Connect()
	select {
	case <-token.Done():
		if token.Error() != nil {
			venom.Debug(ctx, "connection setup failed")
			return nil, errors.Wrap(token.Error(), "failed to connect to MQTT")
		}
		// else connection complete, all good.
	case <-time.After(time.Duration(e.Timeout) * time.Millisecond):
		venom.Debug(ctx, "connection timeout")
		return nil, errors.Wrap(token.Error(), "failed to connect to MQTT")
	case <-ctx.Done():
		venom.Debug(ctx, "Context requested cancellation in session()")
		return nil, errors.New("Context requested cancellation in session()")
	}

	venom.Debug(ctx, "connection setup completed")

	return client, nil
}

// publishMessages is a step that sends configured messages to client connection
func (e Executor) publishMessages(ctx context.Context) error {
	client, err := e.session(ctx, nil)
	if err != nil {
		venom.Debug(ctx, "Failed to create session (publishMessages)")
		return err
	}
	defer client.Disconnect(disconnectTimeoutMs)

	for i, m := range e.Messages {
		if len(m.Topic) == 0 {
			return errors.Errorf("mandatory field Topic was empty in Messages[%v](%v)", i, m)
		}

		token := client.Publish(m.Topic, m.QOS, m.Retained, m.Payload)
		select {
		case <-token.Done():
			if token.Error() != nil {
				venom.Debug(ctx, "Message publish failed")
				return errors.Wrapf(token.Error(), "Message publish failed: Messages[%v](%v)", i, m)
			}
			// else publish complete, all good.
		case <-time.After(time.Duration(e.Timeout) * time.Millisecond):
			venom.Debug(ctx, "Publish attempt timed out")
			return errors.Errorf("Publish attempt timed out on topic %v", m.Topic)
		case <-ctx.Done():
			venom.Debug(ctx, "Context requested cancellation in publishMessages()")
			return errors.New("Context requested cancellation in publishMessages()")
		}
		venom.Debug(ctx, "Message[%v] %q sent (topic: %q)", i, m.Payload, m.Topic)
	}

	return nil
}

// consumeMessages is a step to consume messages from mqtt broker using client connection
func (e Executor) consumeMessages(ctx context.Context) (messages []interface{}, messagesJSON []interface{}, topics []string, err error) {
	ch := make(chan mq.Message, 1)
	defer close(ch)
	subscriber := newSubscriber(ctx, ch)
	client, err := e.session(ctx, subscriber)
	if err != nil {
		venom.Debug(ctx, "Failed to create session (consumeMessages)")
		return nil, nil, nil, err
	}
	defer client.Disconnect(disconnectTimeoutMs)

	start := time.Now()

	for _, topic := range e.Topics {
		token := client.Subscribe(topic, e.QOS, subscriber)
		select {
		case <-token.Done():
			if token.Error() != nil {
				venom.Debug(ctx, "Failed to subscribe")
				return nil, nil, nil, errors.Wrapf(token.Error(), "failed to subscribe to topic %v", topic)
			}
			// else subscription complete, all good.
		case <-time.After(time.Duration(e.Timeout) * time.Millisecond):
			venom.Debug(ctx, "Subscription attempt timed out")
			return nil, nil, nil, errors.Errorf("Subscription attempt timed out on topic %v", topic)
		case <-ctx.Done():
			venom.Debug(ctx, "Context requested cancellation")
			return nil, nil, nil, errors.New("Context requested cancellation")
		}
	}

	messages = []interface{}{}
	messagesJSON = []interface{}{}
	topics = []string{}

	venom.Debug(ctx, "message limit %d", e.MessageLimit)
	ctx2, cancel := context.WithTimeout(ctx, time.Duration(e.Timeout)*time.Millisecond)
	defer cancel()
	for i := 0; i < e.MessageLimit; i++ {
		venom.Debug(ctx, "Reading message n° %d", i)

		var t string
		var m []byte
		select {
		case msg := <-ch:
			m = msg.Payload()
			t = msg.Topic()
		case <-ctx2.Done():
			break
		}

		messages = append(messages, m)
		topics = append(topics, t)

		s := string(m)
		venom.Debug(ctx, "message received. topic: %s len(%d), %s", t, len(m), s)

		var bodyJSONArray []interface{}
		if err := venom.JSONUnmarshal(m, &bodyJSONArray); err != nil {
			bodyJSONMap := map[string]interface{}{}
			err := venom.JSONUnmarshal(m, &bodyJSONMap)
			if err != nil {
				venom.Debug(ctx, "unable to decode message as json")
			}
			messagesJSON = append(messagesJSON, bodyJSONMap)
		} else {
			messagesJSON = append(messagesJSON, bodyJSONArray)
		}
	}
	d := time.Since(start)
	venom.Debug(ctx, "read(s) took %v msec", d.Milliseconds())

	return messages, messagesJSON, topics, nil
}

// persistMessages is a step that registers or un-registers persistent topic subscriptions against a given client id
func (e Executor) persistMessages(ctx context.Context) error {
	client, err := e.session(ctx, nil)
	if err != nil {
		venom.Debug(ctx, "Failed to create session (persistMessages)")
		return err
	}
	defer client.Disconnect(disconnectTimeoutMs)

	for _, topic := range e.Topics {
		token := client.Subscribe(topic, e.QOS, func(client mq.Client, message mq.Message) {
			venom.Debug(ctx, "msg received in persist request: %v", string(message.Payload()))
		})
		select {
		case <-token.Done():
			if token.Error() != nil {
				venom.Debug(ctx, "Failed to subscribe")
				return errors.Wrapf(token.Error(), "failed to subscribe to topic %v", topic)
			}
			// else subscription complete, all good.
		case <-time.After(time.Duration(e.Timeout) * time.Millisecond):
			venom.Debug(ctx, "Subscription attempt timed out")
			return errors.Errorf("Subscription attempt timed out on topic %v", topic)
		case <-ctx.Done():
			venom.Debug(ctx, "Context requested cancellation")
			return errors.New("Context requested cancellation")
		}
	}
	return nil
}

// newSubscriber is a topic subscription handler that forwards onto the passed channel
func newSubscriber(ctx context.Context, ch chan mq.Message) func(client mq.Client, message mq.Message) {
	return func(client mq.Client, message mq.Message) {
		t := message.Topic()
		m := message.Payload()
		venom.Debug(ctx, "rx message in subscribe handler: %s len(%d), %v", t, len(m), m)
		ch <- message
	}
}
