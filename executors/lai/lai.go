package lai

import (
	"context"
	"encoding/json"
	"fmt"
	"git.orcatech.org/infrastructure/data/backend"
	"git.orcatech.org/infrastructure/data/backend/conn/onprem"
	"git.orcatech.org/sdks/golang/config"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Message represents the object sent or received from kafka
type Message struct {
	Topic     string `json:"topic" yaml:"topic"`
	Key       string `json:"key" yaml:"key"`
	Value     string `json:"value,omitempty" yaml:"value,omitempty"`
	ValueFile string `json:"valueFile,omitempty" yaml:"valueFile,omitempty"`
}

type MessageJSON struct {
	Topic string
	Key   interface{}
	Value interface{}
}

// Result represents a step result.
type Result struct {
	TimeSeconds  float64       `json:"timeseconds,omitempty" yaml:"timeSeconds,omitempty"`
	Messages     []Message     `json:"messages,omitempty" yaml:"messages,omitempty"`
	MessagesJSON []interface{} `json:"messagesjson,omitempty" yaml:"messagesJSON,omitempty"`
	Err          string        `json:"err" yaml:"error"`
}

type Executor struct {
	Command string

	ParamsFile string
	// ClientType must be "consumer" or "producer"
	ClientType string `json:"client_type,omitempty" yaml:"clientType,omitempty"`
	// Used when ClientType is consumer
	Topics []string `json:"topics,omitempty" yaml:"topics,omitempty"`

	// Used when ClientType is producer
	// Messages represents the message sended by producer
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`

	// MessagesFile represents the messages into the file sended by producer (messages field would be ignored)
	MessagesFile string `json:"messages_file,omitempty" yaml:"messages_file,omitempty"`
}

const Name = "lai"

func New() venom.Executor {
	return &Executor{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []venom.Assertion{"result.err ShouldBeEmpty"}}
}

var localOption *OptionSet

func (e Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// transform step to Executor Instance
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	start := time.Now()
	result := Result{}

	localContext, _ := context.WithTimeout(ctx, 10*time.Second)
	workdir := venom.StringVarFromCtx(localContext, "venom.testsuite.workdir")

	options := NewOptionSet()
	if localOption == nil {
		localOption = options

		config.ParseFilename("", "", workdir+"/"+e.ParamsFile, options.Configs()...)
	} else {
		options = localOption
	}

	manager := onprem.NewConnectionManager(options.OnPremisesOptionSet)
	conn, _ := manager.DialContext(localContext)
	defer backend.Close(conn)

	switch e.ClientType {
	case "producer":
		err := e.produceMessages(workdir, localContext, conn, e.Messages[0].Topic)
		if err != nil {
			result.Err = err.Error()
		}
	case "consumer":
		var err error
		result, err = e.consumeMessages(localContext, conn)
		if err != nil {
			result.Err = err.Error()
		}
	default:
		return nil, fmt.Errorf("type must be a consumer or a producer")
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func (e Executor) produceMessages(workdir string, ctx context.Context, conn *backend.DatabaseConnections, topic string) error {
	log.Info("produce")
	if len(e.Messages) == 0 && e.MessagesFile == "" {
		return fmt.Errorf("Either one of `messages` or `messagesFile` field must be set")
	}
	if e.MessagesFile != "" {
		path := filepath.Join(workdir, e.MessagesFile)
		if _, err := os.Stat(path); err == nil {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			messages := []Message{}
			err = venom.JSONUnmarshal(content, &messages)
			if err != nil {
				return err
			}
			e.Messages = messages
		} else {
			return err
		}
	}

	for i := range e.Messages {
		message := e.Messages[i]
		value, err := e.getMessageValue(&message, workdir)
		if err != nil {
			return err
		}

		record := backend.NewObservationVendor()
		err = record.UnmarshalJSON(value)
		if err != nil {
			return err
		}

		_, _, err = backend.Produce(ctx, conn, record)
		if err != nil {
			return err
		}
	}

	return nil
}

func OnStartNoOp(_ string) error { return nil }

func (e Executor) consumeMessages(ctx context.Context, conn *backend.DatabaseConnections) (Result, error) {
	log.Info("consume")
	if len(e.Topics) == 0 {
		return Result{}, fmt.Errorf("You must provide topics")
	}

	outResult := Result{}
	currentTopic := e.Topics[0]
	err := backend.Consume(ctx, conn, e.Topics, OnStartNoOp, func(ctx context.Context, record backend.Record) error {
		newMessage := &Message{}
		newMessageJSON := &MessageJSON{}

		var emptyJSON interface{}
		jsonBits, _ := json.Marshal(record)
		err := json.Unmarshal(jsonBits, &emptyJSON)
		if err != nil {
			log.Error(err)
			return err
		}

		newMessageJSON.Key = "key"
		newMessageJSON.Value = emptyJSON
		newMessageJSON.Topic = currentTopic
		outResult.Messages = append(outResult.Messages, *newMessage)
		outResult.MessagesJSON = append(outResult.MessagesJSON, *newMessageJSON)
		return nil
	})
	if err != nil {
		log.Error(err)
		return Result{}, err
	}

	return outResult, err
}

func (e Executor) getMessageValue(m *Message, workdir string) ([]byte, error) {
	// We have 2 fields Value and ValueFile from where we can get value, we prefer Value
	if len(m.Value) != 0 {
		// Most easiest scenario - Value is present
		return []byte(m.Value), nil
	}
	// Read from file
	s := path.Join(workdir, m.ValueFile)
	value, err := os.ReadFile(s)
	if err != nil {
		return nil, fmt.Errorf("can't read from %s: %w", s, err)
	}
	return value, nil
}
