package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

const (
	// Name of executor
	Name                          = "kafka"
	defaultExecutorTimeoutSeconds = 5
	defaultProducerMaxRetries     = 10
	defaultDialTimeout            = 10 * time.Second
)

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

type (
	// Message represents the object sended or received from kafka
	Message struct {
		Topic          string `json:"topic" yaml:"topic"`
		Key            string `json:"key" yaml:"key"`
		Value          string `json:"value,omitempty" yaml:"value,omitempty"`
		ValueFile      string `json:"valueFile,omitempty" yaml:"valueFile,omitempty"`
		AvroSchemaFile string `json:"avroSchemaFile,omitempty" yaml:"avroSchemaFile,omitempty"`
	}

	// MessageJSON represents the object sended or received from kafka
	MessageJSON struct {
		Topic string
		Key   interface{}
		Value interface{}
	}

	// Executor represents a Test Exec
	Executor struct {
		Addrs []string `json:"addrs,omitempty" yaml:"addrs,omitempty"`
		// Registry schema address
		SchemaRegistryAddr string `json:"schema_registry_addr,omitempty" yaml:"schemaRegistryAddr,omitempty"`
		WithAVRO           bool   `json:"with_avro,omitempty" yaml:"withAVRO,omitempty"`
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
		// Represents the timeout for reading messages. In Seconds. Default 5
		Timeout int `json:"timeout,omitempty" yaml:"timeout,omitempty"`
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

		schemaReg SchemaRegistry
	}

	// Result represents a step result.
	Result struct {
		TimeSeconds  float64       `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
		Messages     []Message     `json:"messages,omitempty" yaml:"messages,omitempty"`
		MessagesJSON []interface{} `json:"messagesJSON,omitempty" yaml:"messagesJSON,omitempty"`
		Err          string        `json:"error" yaml:"error"`
	}
	consumeFunc = func(message *sarama.ConsumerMessage) (Message, interface{}, error)
)

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	start := time.Now()

	result := Result{}
	if e.WithAVRO && len(e.SchemaRegistryAddr) != 0 {
		var err error
		e.schemaReg, err = NewSchemaRegistry(e.SchemaRegistryAddr)
		if err != nil {
			return nil, fmt.Errorf("can't create SchemaRegistry: %s", err)
		}
	}

	if e.Timeout == 0 {
		e.Timeout = defaultExecutorTimeoutSeconds
	}
	switch e.ClientType {
	case "producer":
		workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")
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

	return result, nil
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
		path := filepath.Join(workdir, e.MessagesFile)
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
		value, err := e.getMessageValue(&message, workdir)
		if err != nil {
			return err
		}
		messages = append(messages, &sarama.ProducerMessage{
			Topic: message.Topic,
			Key:   sarama.ByteEncoder([]byte(message.Key)),
			Value: sarama.ByteEncoder(value),
		})
	}

	return sp.SendMessages(messages)
}

func (e Executor) getMessageValue(m *Message, workdir string) ([]byte, error) {
	value, err := e.getRAWMessageValue(m, workdir)
	if err != nil {
		return nil, fmt.Errorf("can't get value: %w", err)
	}
	if !e.WithAVRO {
		// This is test without AVRO - value is all we need to have
		return value, nil
	}
	// This is test with Avro
	// 1. Read schema from file
	schemaFile := strings.Trim(m.AvroSchemaFile, " ")
	if len(schemaFile) == 0 {
		return nil, fmt.Errorf("no AVRO schema file specified")
	}
	shemaPath := path.Join(workdir, m.AvroSchemaFile)
	schema, err := ioutil.ReadFile(shemaPath)
	if err != nil {
		return nil, fmt.Errorf("can't read from %s: %w", shemaPath, err)
	}
	// 2. Encode Value with schema
	avroMsg, err := Convert2Avro(value, string(schema))
	if err != nil {
		return nil, fmt.Errorf("can't convert value 2 avro with schema: %w", err)
	}
	// 3. Push schema 2 Schema Registry
	schemaID, err := e.schemaReg.RegisterNewSchema(fmt.Sprintf("%s-value", m.Topic), string(schema))
	if err != nil {
		return nil, fmt.Errorf("can't register new schame in SchemaRegistry: %s", err)
	}
	// 4. Create Kafka message with majic byte and schema ID
	encodedAvroMsg, err := CreateMessage(avroMsg, schemaID)
	if err != nil {
		return nil, fmt.Errorf("can't encode avro message with schemaID: %s", err)
	}
	return encodedAvroMsg, nil
}

func (e Executor) getRAWMessageValue(m *Message, workdir string) ([]byte, error) {
	// We have 2 fields Value and ValueFile from where we can get value, we prefer Value
	if len(m.Value) != 0 {
		// Most easiest scenario - Value is present
		return []byte(m.Value), nil
	}
	// Read from file
	s := path.Join(workdir, m.ValueFile)
	value, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, fmt.Errorf("can't read from %s: %w", s, err)
	}
	return value, nil
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
		return nil, nil, fmt.Errorf("error instanciate consumer err: %w", err)
	}
	defer func() { _ = consumerGroup.Close() }()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.Timeout)*time.Second)
	defer cancel()

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			venom.Error(ctx, "error on consume:%s", err)
		}
	}()

	h := &handler{
		withAVRO:     e.WithAVRO,
		messages:     []Message{},
		messagesJSON: []interface{}{},
		markOffset:   e.MarkOffset,
		messageLimit: e.MessageLimit,
		schemaReg:    e.schemaReg,
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
	withAVRO     bool
	messages     []Message
	messagesJSON []interface{}
	markOffset   bool
	messageLimit int
	schemaReg    SchemaRegistry
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
		consumeFunction := h.consumeJSON
		if h.withAVRO {
			consumeFunction = h.consumeAVRO
		}
		msg, msgJSON, err := consumeFunction(message)
		if err != nil {
			return err
		}
		h.messages = append(h.messages, msg)
		h.messagesJSON = append(h.messagesJSON, msgJSON)

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

func (h *handler) consumeJSON(message *sarama.ConsumerMessage) (Message, interface{}, error) {
	msg := Message{
		Topic: message.Topic,
		Key:   string(message.Key),
		Value: string(message.Value),
	}
	msgJSON := MessageJSON{
		Topic: message.Topic,
	}
	convertFromMessage2JSON(&msg, &msgJSON)

	return msg, msgJSON, nil
}

func (h *handler) consumeAVRO(message *sarama.ConsumerMessage) (Message, interface{}, error) {
	msg := Message{
		Topic: message.Topic,
		Key:   string(message.Key),
	}
	msgJSON := MessageJSON{
		Topic: message.Topic,
	}
	// 1. Get Schema ID
	avroMsg, schemaID := GetMessageAvroID(message.Value)
	schema, err := h.schemaReg.GetSchemaByID(schemaID)
	if err != nil {
		return msg, nil, fmt.Errorf("can't get Schema with ID %d: %w", schemaID, err)
	}
	// 2. Decode Avro Msg
	value, err := ConvertFromAvro(avroMsg, schema)
	if err != nil {
		return msg, nil, fmt.Errorf("can't get value from Avro message: %w", err)
	}
	msg.Value = value
	convertFromMessage2JSON(&msg, &msgJSON)
	return msg, msgJSON, nil
}

func convertFromMessage2JSON(message *Message, msgJSON *MessageJSON) {
	// unmarshall the message.Value
	listMessageJSON := []MessageJSON{}
	// try to unmarshall into an array
	if err := json.Unmarshal([]byte(message.Value), &listMessageJSON); err != nil {
		// try to unmarshall into a map
		mapMessageJSON := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(message.Value), &mapMessageJSON); err2 != nil {
			// try to unmarshall into a string
			msgJSON.Value = message.Value
		} else {
			msgJSON.Value = mapMessageJSON
		}
	} else {
		msgJSON.Value = listMessageJSON
	}

	// unmarshall the message.Key
	listMessageJSON = []MessageJSON{}
	// try to unmarshall into an array
	if err := json.Unmarshal([]byte(message.Key), &listMessageJSON); err != nil {
		// try to unmarshall into a map
		mapMessageJSON := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(message.Key), &mapMessageJSON); err2 != nil {
			// try to unmarshall into a string
			msgJSON.Key = message.Key
		} else {
			msgJSON.Key = mapMessageJSON
		}
	} else {
		msgJSON.Key = listMessageJSON
	}
}
