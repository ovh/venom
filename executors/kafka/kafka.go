package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

// Name of executor
const Name = "kafka"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

//Message represents the object sended or received from kafka
type Message struct {
	Topic string
	Value string
}

//MessageJSON represents the object sended or received from kafka
type MessageJSON struct {
	Topic string
	Value interface{}
}

// Executor represents a Test Exec
type Executor struct {
	Addrs              []string `json:"addrs,omitempty" yaml:"addrs,omitempty"`
	WithTLS            bool     `json:"with_tls,omitempty" yaml:"withTLS,omitempty"`
	WithSASL           bool     `json:"with_sasl,omitempty" yaml:"withSASL,omitempty"`
	WithSASLHandshaked bool     `json:"with_sasl_handshaked,omitempty" yaml:"withSASLHandshaked,omitempty"`
	User               string   `json:"user,omitempty" yaml:"user,omitempty"`
	Password           string   `json:"password,omitempty" yaml:"password,omitempty"`

	//ClientType must be "consumer" or "producer"
	ClientType string `json:"client_type,omitempty" yaml:"clientType,omitempty"`

	//Used when ClientType is consumer
	GroupID string   `json:"group_id,omitempty" yaml:"groupID,omitempty"`
	Topics  []string `json:"topics,omitempty" yaml:"topics,omitempty"`
	//Represents the timeout for reading messages. In Milliseconds. Default 5000
	Timeout int64 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	//Represents the limit of message will be read. After limit, consumer stop read message
	MessageLimit int `json:"message_limit,omitempty" yaml:"messageLimit,omitempty"`
	//InitialOffset represents the initial offset for the consumer. Possible value : newest, oldest. default: newest
	InitialOffset string `json:"initial_offset,omitempty" yaml:"initialOffset,omitempty"`
	//MarkOffset allows to mark offset when consuming message
	MarkOffset bool `json:"mark_offset,omitempty" yaml:"markOffset,omitempty"`

	//Used when ClientType is producer
	//Messages represents the message sended by producer
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`

	//MessagesFile represents the messages into the file sended by producer (messages field would be ignored)
	MessagesFile string `json:"messages_file,omitempty" yaml:"messages_file,omitempty"`

	// Kafka version, default is 0.10.2.0
	KafkaVersion string `json:"kafka_version,omitempty" yaml:"kafka_version,omitempty"`
}

// Result represents a step result.
type Result struct {
	TimeSeconds  float64       `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman    string        `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
	Messages     []Message     `json:"messages,omitempty" yaml:"messages,omitempty"`
	MessagesJSON []interface{} `json:"messagesJSON,omitempty" yaml:"messagesJSON,omitempty"`
	Err          string        `json:"error" yaml:"error"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep, workdir string) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	start := time.Now()

	result := Result{}

	if e.Timeout == 0 {
		e.Timeout = 5000
	}
	switch e.ClientType {
	case "producer":
		err := e.produceMessages(workdir)
		if err != nil {
			result.Err = err.Error()
		}
	case "consumer":
		var err error
		result.Messages, result.MessagesJSON, err = e.consumeMessages(ctx)
		if err != nil {
			result.Err = err.Error()
		}
	default:
		return nil, fmt.Errorf("type must be a consumer or a producer")
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = elapsed.String()

	return result, nil
}

func (e Executor) produceMessages(workdir string) error {
	if len(e.Messages) == 0 && e.MessagesFile == "" {
		return fmt.Errorf("At least messages or messagesFile property must be setted")
	}

	config, err := e.getKafkaConfig()
	if err != nil {
		return err
	}

	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true

	sp, err := sarama.NewSyncProducer(e.Addrs, config)
	if err != nil {
		return err
	}
	defer sp.Close()

	messages := []*sarama.ProducerMessage{}

	if e.MessagesFile != "" {
		path := filepath.Join(workdir, string(e.MessagesFile))
		if _, err = os.Stat(path); err == nil {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			messages := []Message{}
			err = json.Unmarshal(content, &messages)
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
		messages = append(messages, &sarama.ProducerMessage{
			Topic: message.Topic,
			Value: sarama.ByteEncoder([]byte(message.Value)),
		})
	}
	return sp.SendMessages(messages)
}

func (e Executor) consumeMessages(ctx context.Context) ([]Message, []interface{}, error) {
	if len(e.Topics) == 0 {
		return nil, nil, fmt.Errorf("You must provide topics")
	}

	config, err := e.getKafkaConfig()
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(e.InitialOffset) == "oldest" {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	consumerGroup, err := sarama.NewConsumerGroup(e.Addrs, e.GroupID, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error instanciate consumer err:%s", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.Timeout)*time.Millisecond)
	defer cancel()

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			venom.Error(ctx, "error on consume:%s", err)
		}
	}()

	h := &handler{
		messages:     []Message{},
		messagesJSON: []interface{}{},
		markOffset:   e.MarkOffset,
		messageLimit: e.MessageLimit,
	}

	if err := consumerGroup.Consume(ctx, e.Topics, h); err != nil {
		venom.Error(ctx, "error on consume:%s", err)
	}

	return h.messages, h.messagesJSON, nil
}

func (e Executor) getKafkaConfig() (*sarama.Config, error) {
	config := sarama.NewConfig()
	config.Net.TLS.Enable = e.WithTLS
	config.Net.SASL.Enable = e.WithSASL
	config.Net.SASL.User = e.User
	config.Net.SASL.Password = e.Password
	config.Consumer.Return.Errors = true
	config.Net.DialTimeout = 10 * time.Second

	if e.KafkaVersion != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(e.KafkaVersion)
		if err != nil {
			return config, fmt.Errorf("error parsing Kafka version %v err:%s", kafkaVersion, err)
		}
		config.Version = kafkaVersion
	} else {
		config.Version = sarama.V0_10_2_0
	}

	return config, nil
}

// handler represents a Sarama consumer group consumer
type handler struct {
	messages     []Message
	messagesJSON []interface{}
	markOffset   bool
	messageLimit int
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *handler) Setup(s sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		h.messages = append(h.messages, Message{
			Topic: message.Topic,
			Value: string(message.Value),
		})
		messageJSONArray := []MessageJSON{}
		if err := json.Unmarshal(message.Value, &messageJSONArray); err != nil {
			messageJSONMap := map[string]interface{}{}
			if err2 := json.Unmarshal(message.Value, &messageJSONMap); err2 == nil {
				h.messagesJSON = append(h.messagesJSON, MessageJSON{
					Topic: message.Topic,
					Value: messageJSONMap,
				})
			} else {
				h.messagesJSON = append(h.messagesJSON, MessageJSON{
					Topic: message.Topic,
					Value: string(message.Value),
				})
			}
		} else {
			h.messagesJSON = append(h.messagesJSON, MessageJSON{
				Topic: message.Topic,
				Value: messageJSONArray,
			})
		}
		if h.markOffset {
			session.MarkMessage(message, "")
		}
		if h.messageLimit > 0 && len(h.messages) >= h.messageLimit {
			venom.Info(context.Background(), "message limit reached")
			return nil
		}
		session.MarkMessage(message, "delivered")
	}
	return nil
}
