package kafka

import (
	"bytes"
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
	"github.com/ovh/venom/executors"
	"github.com/ovh/venom/executors/kafka/avro"
)

// Name of executor
const Name = "kafka"

const (
	defaultExecutorTimeoutMs  = 5000
	defaultProducerMaxRetries = 10
	defaultDialTimeout        = 10 * time.Second
)

type consumerEncoding string

const (
	jsonEncoding = consumerEncoding("JSON")
	avroEncoding = consumerEncoding("AVRO")
)

var mapConsumerEncoding = map[string]consumerEncoding{
	"":     jsonEncoding,
	"AVRO": avroEncoding,
	"JSON": jsonEncoding,
}

type consumeFunc = func(message *sarama.ConsumerMessage) (Message, interface{})

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Message represents the object sended or received from kafka
type Message struct {
	Topic string
	Key   string
	Value string
}

// MessageJSON represents the object sended or received from kafka
type MessageJSON struct {
	Topic string
	Key   interface{}
	Value interface{}
}

// Executor represents a Test Exec
type Executor struct {
	Addrs []string `json:"addrs,omitempty" yaml:"addrs,omitempty"`
	// Registry schema address
	SchemaRegistryAddr string `json:"schema_registry_addr,omitempty" yaml:"schemaRegistryAddr,omitempty"`
	WithTLS            bool   `json:"with_tls,omitempty" yaml:"withTLS,omitempty"`
	WithSASL           bool   `json:"with_sasl,omitempty" yaml:"withSASL,omitempty"`
	WithSASLHandshaked bool   `json:"with_sasl_handshaked,omitempty" yaml:"withSASLHandshaked,omitempty"`
	User               string `json:"user,omitempty" yaml:"user,omitempty"`
	Password           string `json:"password,omitempty" yaml:"password,omitempty"`

	// ClientType must be "consumer" or "producer"
	ClientType string `json:"client_type,omitempty" yaml:"clientType,omitempty"`

	// Used when ClientType is consumer
	GroupID string   `json:"group_id,omitempty" yaml:"groupID,omitempty"`
	Topics  []string `json:"topics,omitempty" yaml:"topics,omitempty"`
	// Represents the timeout for reading messages. In Milliseconds. Default 5000
	Timeout int64 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// Represents the limit of message will be read. After limit, consumer stop read message
	MessageLimit int `json:"message_limit,omitempty" yaml:"messageLimit,omitempty"`
	// InitialOffset represents the initial offset for the consumer. Possible value : newest, oldest. default: newest
	InitialOffset string `json:"initial_offset,omitempty" yaml:"initialOffset,omitempty"`
	// MarkOffset allows to mark offset when consuming message
	MarkOffset bool `json:"mark_offset,omitempty" yaml:"markOffset,omitempty"`

	// Only one of JSON or Avro are currently supported
	ConsumerEncoding string `json:"consumer_encoding,omitempty" yaml:"consumerEncoding,omitempty"`

	// Used when ClientType is producer
	// Messages represents the message sended by producer
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`

	// MessagesFile represents the messages into the file sended by producer (messages field would be ignored)
	MessagesFile string `json:"messages_file,omitempty" yaml:"messages_file,omitempty"`

	// Kafka version, default is 0.10.2.0
	KafkaVersion string `json:"kafka_version,omitempty" yaml:"kafka_version,omitempty"`
}

// Result represents a step result.
type Result struct {
	Executor     Executor      `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds  float64       `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman    string        `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
	Messages     []Message     `json:"messages,omitempty" yaml:"messages,omitempty"`
	MessagesJSON []interface{} `json:"messagesJSON,omitempty" yaml:"messagesJSON,omitempty"`
	Err          string        `json:"error" yaml:"error"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	start := time.Now()

	result := Result{Executor: e}

	if e.Timeout == 0 {
		e.Timeout = defaultExecutorTimeoutMs
	}
	if e.ClientType == "producer" {
		err := e.produceMessages(workdir)
		if err != nil {
			result.Err = err.Error()
		}
	} else if e.ClientType == "consumer" {
		var err error
		result.Messages, result.MessagesJSON, err = e.consumeMessages(l)
		if err != nil {
			result.Err = err.Error()
		}
	} else {
		return nil, fmt.Errorf("type must be a consumer or a producer")
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = elapsed.String()
	result.Executor.Password = "****hidden****" // do not output password

	return executors.Dump(result)
}

func (e Executor) produceMessages(workdir string) error {
	if len(e.Messages) == 0 && e.MessagesFile == "" {
		return fmt.Errorf("Either one of `messages` or `messagesFile` field must be set")
	}

	config, err := e.getKafkaConfig()
	if err != nil {
		return err
	}

	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Retry.Max = defaultProducerMaxRetries
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
			Key:   sarama.ByteEncoder([]byte(message.Key)),
			Value: sarama.ByteEncoder([]byte(message.Value)),
		})
	}

	return sp.SendMessages(messages)
}

func (e Executor) consumeMessages(l venom.Logger) ([]Message, []interface{}, error) {
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
		return nil, nil, fmt.Errorf("error instanciate consumer err: %w", err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.Timeout)*time.Millisecond)
	defer cancel()

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			l.Errorf("error on consume: %w", err)
		}
	}()

	encoding, ok := mapConsumerEncoding[e.ConsumerEncoding]
	if !ok {
		encoding = mapConsumerEncoding[""]
	}

	h := &handler{
		messages:     []Message{},
		messagesJSON: []interface{}{},
		markOffset:   e.MarkOffset,
		messageLimit: e.MessageLimit,
		logger:       l,
		encoding:     encoding,
	}

	if err := consumerGroup.Consume(ctx, e.Topics, h); err != nil {
		l.Errorf("error on consume: %w", err)
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
	config.Net.DialTimeout = defaultDialTimeout
	config.Version = sarama.V0_10_2_0

	if e.KafkaVersion != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(e.KafkaVersion)
		if err != nil {
			return config, fmt.Errorf("error parsing Kafka version %v err: %w", kafkaVersion, err)
		}
		config.Version = kafkaVersion
	}

	return config, nil
}

// handler represents a Sarama consumer group consumer
type handler struct {
	messages     []Message
	messagesJSON []interface{}
	markOffset   bool
	messageLimit int
	logger       venom.Logger
	encoding     consumerEncoding
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
		var mapConsumeConsumeFunc = map[consumerEncoding]consumeFunc{
			jsonEncoding: h.consumeJSON,
			avroEncoding: h.consumeAvro,
		}
		consumeFunc := mapConsumeConsumeFunc[h.encoding]
		msg, msgJSON := consumeFunc(message)
		h.messages = append(h.messages, msg)
		h.messagesJSON = append(h.messagesJSON, msgJSON)

		if h.markOffset {
			session.MarkMessage(message, "")
		}
		if h.messageLimit > 0 && len(h.messages) >= h.messageLimit {
			h.logger.Infof("message limit reached")
			return nil
		}
		session.MarkMessage(message, "delivered")
	}
	return nil
}

func (h *handler) consumeJSON(message *sarama.ConsumerMessage) (Message, interface{}) {
	msg := Message{
		Topic: message.Topic,
		Key:   string(message.Key),
		Value: string(message.Value),
	}
	msgJSON := MessageJSON{
		Topic: message.Topic,
	}

	// unmarshall the message.Value
	listMessageJSON := []MessageJSON{}
	// try to unmarshall into an array
	if err := json.Unmarshal(message.Value, &listMessageJSON); err != nil {
		// try to unmarshall into a map
		mapMessageJSON := map[string]interface{}{}
		if err2 := json.Unmarshal(message.Value, &mapMessageJSON); err2 != nil {
			// try to unmarshall into a string
			msgJSON.Value = string(message.Value)
		} else {
			msgJSON.Value = mapMessageJSON
		}
	} else {
		msgJSON.Value = listMessageJSON
	}

	// unmarshall the message.Key
	listMessageJSON = []MessageJSON{}
	// try to unmarshall into an array
	if err := json.Unmarshal(message.Key, &listMessageJSON); err != nil {
		// try to unmarshall into a map
		mapMessageJSON := map[string]interface{}{}
		if err2 := json.Unmarshal(message.Key, &mapMessageJSON); err2 != nil {
			// try to unmarshall into a string
			msgJSON.Key = string(message.Key)
		} else {
			msgJSON.Key = mapMessageJSON
		}
	} else {
		msgJSON.Key = listMessageJSON
	}

	return msg, msgJSON
}

func (h *handler) consumeAvro(message *sarama.ConsumerMessage) (Message, interface{}) {

	// _, schemaID := getMessageByte(message.Value)

	kMsg := avro.NewMessage()
	err := kMsg.FromKafka(message)
	if err != nil {
		h.logger.Errorf(
			"error getting Avro msg from Sarama msg: %w. Topic: %s, Partition: %s, Offset: %d",
			err,
			message.Topic,
			message.Partition,
			message.Offset,
		)
	}
	buffer := bytes.NewBuffer(nil)
	err = kMsg.Serialize(buffer)
	if err != nil {
		h.logger.Errorf(
			"error avro serialize: %w. Topic: %s, Partition: %s, Offset: %d",
			err,
			message.Topic,
			message.Partition,
			message.Offset,
		)
	}

	msg := Message{
		Topic: message.Topic,
		Key:   string(kMsg.Key),
		Value: string(kMsg.Value),
	}
	msgJSON := MessageJSON{
		Topic: message.Topic,
		Key:   string(kMsg.Key),
		Value: string(kMsg.Value),
	}

	return msg, msgJSON
}
