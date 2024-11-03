package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ovh/venom"
)

const Name = "nats"

const (
	defaultUrl            = "nats://localhost:4222"
	defaultConnectTimeout = 5 * time.Second
	defaultReconnectTime  = 1 * time.Second
	defaultClientName     = "Venom"
	defaultMessageLimit   = 1
	defaultDeadline       = 5
)

type JetstreamOptions struct {
	Enabled        bool     `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Stream         string   `json:"stream,omitempty" yaml:"stream,omitempty"`     // Stream must exist before the command execution
	Consumer       string   `json:"consumer,omitempty" yaml:"consumer,omitempty"` // If set search for a durable consumer, otherwise use an ephemeral one
	FilterSubjects []string `json:"filterSubjects,omitempty" yaml:"filterSubjects,omitempty"`
}

type Executor struct {
	Command      string              `json:"command,omitempty" yaml:"command,omitempty"` // Must be publish or subscribe
	Url          string              `json:"url,omitempty" yaml:"url,omitempty"`
	Subject      string              `json:"subject,omitempty" yaml:"subject,omitempty"`
	Payload      string              `json:"payload,omitempty" yaml:"payload,omitempty"`
	Header       map[string][]string `json:"header,omitempty" yaml:"header,omitempty"`
	MessageLimit int                 `json:"message_limit,omitempty" yaml:"messageLimit,omitempty"`
	Deadline     int                 `json:"deadline,omitempty" yaml:"deadline,omitempty"` // Describes the deadline in seconds from the start of the command
	ReplySubject string              `json:"reply_subject,omitempty" yaml:"replySubject,omitempty"`
	Request      bool                `json:"request,omitempty" yaml:"request,omitempty"` // Describe that the publish command expects a reply from the NATS server
	Jetstream    JetstreamOptions    `json:"jetstream,omitempty" yaml:"jetstream,omitempty"`
}

// Message describes a NATS message received from a consumer or a request publisher
type Message struct {
	Data         interface{}         `json:"data,omitempty" yaml:"data,omitempty"`
	Header       map[string][]string `json:"header,omitempty" yaml:"header,omitempty"`
	Subject      string              `json:"subject,omitempty" yaml:"subject,omitempty"`
	ReplySubject string              `json:"reply_subject,omitempty" yaml:"replySubject,omitempty"`
}

// Resuts describes a command result
type Result struct {
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`
	Error    string    `json:"error,omitempty" yaml:"error,omitempty"`
}

func (Executor) ZeroValueMessage() Message {
	return Message{}
}

func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	session, err := e.session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	result := Result{}

	switch e.Command {
	case "publish":
		var cmdErr error
		var reply Message

		if e.Jetstream.Enabled {
			cmdErr = e.publishJetstream(ctx, session)
		} else {
			reply, cmdErr = e.publish(ctx, session)
		}

		if cmdErr != nil {
			result.Error = cmdErr.Error()
		} else {
			result.Messages = []Message{reply}
		}
	case "subscribe":
		var msgs []Message
		var cmdErr error

		if e.Jetstream.Enabled {
			msgs, cmdErr = e.subscribeJetstream(ctx, session)
		} else {
			msgs, cmdErr = e.subscribe(ctx, session)
		}

		if cmdErr != nil {
			result.Error = cmdErr.Error()
		} else {
			result.Messages = msgs
		}
	}

	return result, nil
}

func New() venom.Executor {
	return &Executor{
		MessageLimit: defaultMessageLimit,
		Deadline:     defaultDeadline,
		Url:          defaultUrl,
	}
}

func (e Executor) session(ctx context.Context) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Timeout(defaultConnectTimeout),
		nats.Name(defaultClientName),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(defaultReconnectTime),
	}

	venom.Debug(ctx, "Connecting to NATS server %q", e.Url)

	nc, err := nats.Connect(e.Url, opts...)
	if err != nil {
		return nil, err
	}

	venom.Debug(ctx, "Connected to NATS server %q", nc.ConnectedAddr())

	return nc, nil
}

func (e Executor) publish(ctx context.Context, session *nats.Conn) (Message, error) {
	if e.Subject == "" {
		return e.ZeroValueMessage(), fmt.Errorf("subject is required")
	}

	msg := nats.Msg{
		Subject: e.Subject,
		Data:    []byte(e.Payload),
		Header:  e.Header,
	}

	var result Message
	if e.Request {
		if e.ReplySubject == "" {
			return e.ZeroValueMessage(), fmt.Errorf("reply subject is required for request command")
		}
		msg.Reply = e.ReplySubject

		replyMsg, err := session.RequestMsg(&msg, time.Duration(5)*time.Second)
		if err != nil {
			return e.ZeroValueMessage(), err
		}

		result = Message{
			Data:         string(replyMsg.Data),
			Header:       replyMsg.Header,
			Subject:      msg.Subject,
			ReplySubject: replyMsg.Subject,
		}

		venom.Debug(ctx, "Received reply message %+v", result)
	} else {
		err := session.PublishMsg(&msg)
		if err != nil {
			return e.ZeroValueMessage(), err
		}
	}

	venom.Debug(ctx, "Published message to subject %q with payload %q", e.Subject, e.Payload)

	return result, nil
}

func (e Executor) publishJetstream(ctx context.Context, session *nats.Conn) error {
	if e.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	js, err := e.jetstreamSession(ctx, session)
	if err != nil {
		return err
	}

	msg := nats.Msg{
		Subject: e.Subject,
		Data:    []byte(e.Payload),
		Header:  e.Header,
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(e.Deadline)*time.Second)
	defer cancel()

	_, err = js.PublishMsg(ctxWithTimeout, &msg)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("timeout reached while waiting for ACK from NATS server")
		}
		return err
	}

	venom.Debug(ctx, "Published message to subject %q with payload %q", e.Subject, e.Payload)

	return nil
}

func (e Executor) subscribe(ctx context.Context, session *nats.Conn) ([]Message, error) {
	if e.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	venom.Debug(ctx, "Subscribing to subject %q", e.Subject)

	results := make([]Message, e.MessageLimit)

	ch := make(chan *nats.Msg)
	msgCount := 0
	sub, err := session.ChanSubscribe(e.Subject, ch)
	if err != nil {
		return nil, err
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(e.Deadline)*time.Second)
	defer cancel()

	venom.Debug(ctx, "Subscribed to subject %q with timeout %v and max messages %d", e.Subject, e.Deadline, e.MessageLimit)

	for {
		select {
		case msg := <-ch:
			venom.Debug(ctx, "Received message #%d from subject %q with data %q", msgCount, e.Subject, string(msg.Data))

			results[msgCount] = Message{
				Data:    string(msg.Data),
				Header:  msg.Header,
				Subject: msg.Subject,
			}

			msgCount++

			if msgCount >= e.MessageLimit {
				err = sub.Unsubscribe()
				if err != nil {
					return nil, err
				}
				return results, nil
			}
		case <-ctxWithTimeout.Done():
			_ = sub.Unsubscribe() // even it if fails, we are done anyway
			return nil, fmt.Errorf("timeout reached while waiting for message #%d from subject %q", msgCount, e.Subject)
		}
	}
}

func (e Executor) jetstreamSession(ctx context.Context, session *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(session)
	if err != nil {
		return nil, err
	}
	venom.Debug(ctx, "Jetstream session created")
	return js, err
}

func (e Executor) getConsumer(ctx context.Context, session *nats.Conn) (jetstream.Consumer, error) {
	js, err := e.jetstreamSession(ctx, session)
	if err != nil {
		return nil, err
	}

	stream, err := js.Stream(ctx, e.Jetstream.Stream)
	if err != nil {
		return nil, err
	}

	streamName := stream.CachedInfo().Config.Name

	venom.Debug(ctx, "Found stream %q", streamName)

	var consumer jetstream.Consumer
	var consErr error
	if e.Jetstream.Consumer != "" {
		consumer, consErr = stream.Consumer(ctx, e.Jetstream.Consumer)
		if consErr != nil {
			return nil, err
		}
		venom.Debug(ctx, "Found existing consumer %s[%s]", streamName, e.Jetstream.Consumer)
	} else {
		consumer, consErr = stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
			FilterSubjects: e.Jetstream.FilterSubjects,
			AckPolicy:      jetstream.AckAllPolicy,
		})
		if consErr != nil {
			return nil, err
		}
		venom.Warn(ctx, "Consumer %s[%s] not found. Created ephemeral consumer", streamName, e.Jetstream.Consumer)
	}

	return consumer, nil
}

func (e Executor) subscribeJetstream(ctx context.Context, session *nats.Conn) ([]Message, error) {
	if e.Jetstream.Stream == "" {
		return nil, fmt.Errorf("jetstream stream name is required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(e.Deadline)*time.Second)
	defer cancel()

	consumer, err := e.getConsumer(ctx, session)
	if err != nil {
		return nil, err
	}

	results := make([]Message, e.MessageLimit)
	msgCount := 0
	done := make(chan struct{})

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		venom.Debug(ctx, "received message from %s[%s]: %+v", consumer.CachedInfo().Stream, msg.Subject(), string(msg.Data()))
		results[msgCount] = Message{
			Data:         string(msg.Data()),
			Header:       msg.Headers(),
			Subject:      msg.Subject(),
			ReplySubject: msg.Reply(),
		}
		msgCount++
		if msgCount == e.MessageLimit {
			done <- struct{}{}
		}
	}, jetstream.PullMaxMessages(e.MessageLimit))

	defer cc.Drain()
	defer cc.Stop()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return nil, fmt.Errorf("timeout reached while waiting for message #%d from subjects %v", msgCount, e.Jetstream.FilterSubjects)
		case <-done:
			return results, nil
		}
	}
}
