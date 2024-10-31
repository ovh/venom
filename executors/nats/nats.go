package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/nats-io/nats.go"
	"github.com/ovh/venom"
)

const Name = "nats"

const (
	defaultUrl            = "nats://localhost:4222"
	defaultConnectTimeout = 5 * time.Second
	defaultReconnectTime  = 1 * time.Second
	defaultClientName     = "Venom"
)

const (
	defaultMessageLimit = 1
	defaultDeadline     = 5
)

type Executor struct {
	Command      string              `json:"command,omitempty" yaml:"command,omitempty"`
	Url          string              `json:"url,omitempty" yaml:"url,omitempty"`
	Subject      string              `json:"subject,omitempty" yaml:"subject,omitempty"`
	Payload      string              `json:"payload,omitempty" yaml:"payload,omitempty"`
	Header       map[string][]string `json:"header,omitempty" yaml:"header,omitempty"`
	MessageLimit int                 `json:"message_limit,omitempty" yaml:"messageLimit,omitempty"`
	Deadline     int                 `json:"deadline,omitempty" yaml:"deadline,omitempty"`
	ReplySubject string              `json:"reply_subject,omitempty" yaml:"replySubject,omitempty"`
	Request      bool                `json:"request,omitempty" yaml:"request,omitempty"`
}

type Message struct {
	Data         interface{}         `json:"data,omitempty" yaml:"data,omitempty"`
	Header       map[string][]string `json:"header,omitempty" yaml:"header,omitempty"`
	Subject      string              `json:"subject,omitempty" yaml:"subject,omitempty"`
	ReplySubject string              `json:"reply_subject,omitempty" yaml:"replySubject,omitempty"`
}

type Result struct {
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`
	Error    string    `json:"error,omitempty" yaml:"error,omitempty"`
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
		reply, cmdErr := e.publish(ctx, session)
		if cmdErr != nil {
			result.Error = cmdErr.Error()
		} else {
			result.Messages = []Message{*reply}
			venom.Debug(ctx, "Received reply message %+v", result.Messages)
		}
	case "subscribe":
		msgs, cmdErr := e.subscribe(ctx, session)
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
	}
}

func (e Executor) session(ctx context.Context) (*nats.Conn, error) {
	if e.Url == "" {
		venom.Warning(ctx, "No URL provided, using default %q", defaultUrl)
		e.Url = defaultUrl
	}

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

func (e Executor) publish(ctx context.Context, session *nats.Conn) (*Message, error) {
	if e.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	venom.Debug(ctx, "Publishing message to subject %q with payload %q", e.Subject, e.Payload)

	msg := nats.Msg{
		Subject: e.Subject,
		Data:    []byte(e.Payload),
		Header:  e.Header,
	}

	var result Message
	if e.Request {
		if e.ReplySubject == "" {
			return nil, fmt.Errorf("reply subject is required for request command")
		}
		msg.Reply = e.ReplySubject

		replyMsg, err := session.RequestMsg(&msg, time.Duration(5)*time.Second)
		if err != nil {
			return nil, err
		}

		result = Message{
			Data:         string(replyMsg.Data),
			Header:       replyMsg.Header,
			Subject:      msg.Subject,
			ReplySubject: replyMsg.Subject,
		}
	} else {
		err := session.PublishMsg(&msg)
		if err != nil {
			return nil, err
		}
	}

	venom.Debug(ctx, "Message published to subject %q", e.Subject)

	return &result, nil
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
