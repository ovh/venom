package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"time"

	mq "github.com/eclipse/paho.mqtt.golang"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

// TODO: needs AddRoute rather than global default handler as we don't check the topic otherwise

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
	ClientId            string `json:"client_id" yaml:"clientId"`

	// Subscription topic
	Topic string `json:"topic" yaml:"topic"`

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
	TimeSeconds float64 `json:"timeSeconds" yaml:"timeSeconds"`
	//Body        []string      `json:"body" yaml:"body"`
	Topics       []string      `json:"topics" yaml:"topics"`
	Messages     []interface{} `json:"messages" yaml:"messages"`
	MessagesJSON []interface{} `json:"messagesJSON" yaml:"messagesJSON"`
	Err          string        `json:"error" yaml:"error"`
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.error ShouldBeEmpty"}}
}

func (Executor) Run(ctx context.Context, step venom.TestStep, _ string) (interface{}, error) {
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
	if len(e.Topic) != 0 {

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

func (e Executor) session(ctx context.Context, subscriber func(client mq.Client, message mq.Message)) (mq.Client, error) {
	venom.Debug(ctx, "creating session to %v, cleansession: %v, clientid: %v", e.Addrs, !e.PersistSubscription, e.ClientId)

	opts := mq.NewClientOptions().
		AddBroker(e.Addrs).
		SetConnectTimeout(time.Duration(e.ConnectTimeout) * time.Millisecond).
		SetCleanSession(!e.PersistSubscription).
		SetClientID(e.ClientId).
		SetProtocolVersion(mqttV311).
		SetOnConnectHandler(func(client mq.Client) {
			venom.Debug(ctx, "connection handler called. IsConnected: %v", client.IsConnected())
		})

	if subscriber != nil {
		venom.Debug(ctx, "adding global subscriber")
		opts.SetDefaultPublishHandler(subscriber)
	}

	client := mq.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		venom.Debug(ctx, "connection setup failed")
		return nil, errors.Wrap(token.Error(), "failed to connect to MQTT")
	}
	venom.Debug(ctx, "connection setup completed")

	return client, nil
}

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
		if token.Wait() && token.Error() != nil {
			return errors.Wrapf(token.Error(), "failed to publish message: Messages[%v](%v)", i, m)
		}
		venom.Debug(ctx, "Message[%v] %q sent (topic: %q)", i, m.Payload, m.Topic)
	}

	return nil
}

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

	messages = []interface{}{}
	messagesJSON = []interface{}{}
	topics = []string{}

	start := time.Now()

	token := client.Subscribe(e.Topic, e.QOS, subscriber)
	if token.WaitTimeout(time.Duration(e.Timeout)*time.Millisecond) && token.Error() != nil {
		venom.Debug(ctx, "Failed to subscribe during persistent queue setup")
		return nil, nil, nil, errors.Wrapf(token.Error(), "failed to subscribe to topic %v", e.Topic)
	}

	venom.Debug(ctx, "message limit %d", e.MessageLimit)
	ctx2, _ := context.WithTimeout(ctx, time.Duration(e.Timeout)*time.Millisecond)
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
		venom.Debug(ctx, "message: %t len(%d), %s", t, len(m), s)

		bodyJSONArray := []interface{}{}
		if err := json.Unmarshal(m, &bodyJSONArray); err != nil {
			bodyJSONMap := map[string]interface{}{}
			err := json.Unmarshal(m, &bodyJSONMap)
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

func (e Executor) persistMessages(ctx context.Context) error {
	client, err := e.session(ctx, nil)
	if err != nil {
		venom.Debug(ctx, "Failed to create session (persistMessages)")
		return err
	}
	defer client.Disconnect(disconnectTimeoutMs)

	token := client.Subscribe(e.Topic, e.QOS, func(client mq.Client, message mq.Message) {
		venom.Debug(ctx, "msg received in persist request: %v", string(message.Payload()))
	})
	if token.WaitTimeout(time.Duration(e.Timeout)*time.Millisecond) && token.Error() != nil {
		venom.Debug(ctx, "Failed to subscribe during persistent queue setup")
		return errors.Wrapf(token.Error(), "failed to subscribe to topic %v", e.Topic)
	}
	return nil
}

func newSubscriber(ctx context.Context, ch chan mq.Message) func(client mq.Client, message mq.Message) {
	return func(client mq.Client, message mq.Message) {
		var t string
		var m []byte
		t = message.Topic()
		m = message.Payload()
		venom.Debug(ctx, "rx message in subscribe handler: %t len(%d), %v", t, len(m), m)
		ch <- message
	}
}
