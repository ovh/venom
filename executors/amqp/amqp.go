package amqp

import (
	"context"
	"errors"
	"fmt"

	amqp "github.com/Azure/go-amqp"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

// Name of executor
const Name = "amqp"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Addr string `json:"addr" yaml:"addr"`

	// ClientType must be "consumer" or "producer"
	ClientType string `json:"clientType" yaml:"clientType"`

	// Used when ClientType is consumer
	// SourceAddr represents the source address from which to read incoming messages
	SourceAddr string `json:"sourceAddr" yaml:"sourceAddr"`
	// MessageLimit represents the limit of message will be read. After limit, consumer will stop reading
	MessageLimit uint `json:"messageLimit" yaml:"messageLimit"`

	// Used when ClientType is producer
	// TargetAddr represents the target address to which outgoing messages should be published
	TargetAddr string `json:"targetAddr" yaml:"targetAddr"`
	// Messages represents the messages to be sent by producer
	Messages []string `json:"messages" yaml:"messages"`
}

// Result represents a step result
type Result struct {
	Messages     []string      `json:"messages" yaml:"messages"`
	MessagesJSON []interface{} `json:"messagesjson" yaml:"messagesJSON"`
}

// ZeroValueResult returns an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	client, session, err := e.createAMQPSession(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	defer session.Close(ctx)

	switch e.ClientType {
	case "producer":
		return e.publishMessages(ctx, session)
	case "consumer":
		return e.consumeMessages(ctx, session)
	default:
		return nil, fmt.Errorf("clientType %q must be producer or consumer", e.ClientType)
	}
}

func (e Executor) createAMQPSession(ctx context.Context) (*amqp.Conn, *amqp.Session, error) {
	if e.Addr == "" {
		return nil, nil, errors.New("creating session: addr is mandatory")
	}

	client, err := amqp.Dial(e.Addr, &amqp.ConnOptions{
		SASLType: amqp.SASLTypeAnonymous(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating session: %w", err)
	}

	session, err := client.NewSession(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating session: %w", err)
	}

	return client, session, nil
}

func (e Executor) publishMessages(ctx context.Context, session *amqp.Session) (interface{}, error) {
	if e.TargetAddr == "" {
		return nil, errors.New("publishing messages: targetAddr is manatory when clientType is producer")
	}

	if len(e.Messages) < 1 {
		return nil, errors.New("publishing messages: messages length must be > 0 when clientType is producer")
	}

	sender, err := session.NewSender(ctx, e.TargetAddr, nil)
	if err != nil {
		return nil, fmt.Errorf("publishing messages: %w", err)
	}

	for _, m := range e.Messages {
		if err := sender.Send(ctx, amqp.NewMessage([]byte(m))); err != nil {
			return nil, fmt.Errorf("publishing messages: %w", err)
		}
	}

	return nil, nil
}

func (e Executor) consumeMessages(ctx context.Context, session *amqp.Session) (interface{}, error) {
	if e.SourceAddr == "" {
		return nil, errors.New("consuming messages: sourceAddr is manatory when clientType is consumer")
	}

	if e.MessageLimit < 1 {
		return nil, errors.New("consuming messages: messageLimit must be > 0 when clientType is consumer")
	}

	recv, err := session.NewReceiver(ctx, e.SourceAddr, nil)
	if err != nil {
		return nil, fmt.Errorf("consuming messages: %w", err)
	}

	output := Result{
		Messages:     make([]string, 0, e.MessageLimit),
		MessagesJSON: make([]interface{}, 0, e.MessageLimit),
	}

	for i := uint(0); i < e.MessageLimit; i++ {
		msgString, msgJSON, err := consumeMessage(ctx, recv)
		if err != nil {
			return nil, fmt.Errorf("consuming message %d: %w", i, err)
		}

		output.Messages = append(output.Messages, msgString)
		output.MessagesJSON = append(output.MessagesJSON, msgJSON)
	}

	return output, nil
}

func consumeMessage(ctx context.Context, recv *amqp.Receiver) (msgString string, msgJSON interface{}, err error) {
	msg, err := recv.Receive(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("consuming message: %w", err)
	}

	if err = recv.AcceptMessage(context.TODO(), msg); err != nil {
		return "", nil, fmt.Errorf("consuming message: %w", err)
	}

	if err := venom.JSONUnmarshal(msg.GetData(), &msgJSON); err != nil {
		return string(msg.GetData()), nil, nil
	}

	return string(msg.GetData()), msgJSON, nil
}
