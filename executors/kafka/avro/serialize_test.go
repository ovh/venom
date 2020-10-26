package avro_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ovh/venom/executors/kafka/avro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestMessage struct {
	Messages []*sarama.ConsumerMessage
	Error    error
}

type MessageSuite struct {
	suite.Suite
	Tests []TestMessage
}

// before each test
func (suite *MessageSuite) SetupTest() {
	suite.Tests = []TestMessage{
		{
			Messages: []*sarama.ConsumerMessage{
				{
					Headers: []*sarama.RecordHeader{
						{
							Key:   []byte{'T', 'e', 's', 't'},
							Value: []byte{0x1, 0x2},
						},
					},
					Timestamp: time.Now(),
					Value:     []byte{0x0, 0x1, 0x2},
					Key:       []byte{0x3, 0x4, 0x5},

					Topic:     "test",
					Partition: 0,
					Offset:    1,
				},
			},
			Error: nil,
		},
		{
			Messages: []*sarama.ConsumerMessage{
				{
					Headers:   []*sarama.RecordHeader{},
					Timestamp: time.Now(),
					Value:     []byte{0x0, 0x1, 0x2},
					Key:       []byte{0x3, 0x4, 0x5},

					Topic:     "test",
					Partition: 1,
					Offset:    0,
				},
			},
			Error: nil,
		},
	}
}

func (suite *MessageSuite) TestSerialize() {
	for _, test := range suite.Tests {
		for _, message := range test.Messages {
			kMsg := avro.NewMessage()
			err := kMsg.FromKafka(message)
			if test.Error != nil {
				assert.EqualError(suite.T(), err, fmt.Sprintf("%v", test.Error))
				continue
			}
			assert.NoError(suite.T(), err)
			buffer := bytes.NewBuffer(nil)
			err = kMsg.Serialize(buffer)
			assert.NoError(suite.T(), err)
			kMsg2, err := avro.DeserializeMessage(buffer)
			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), kMsg.Key, kMsg2.Key)
			assert.Equal(suite.T(), kMsg.Value, kMsg2.Value)
			assert.Equal(suite.T(), kMsg.Timestamp, kMsg2.Timestamp)
			assert.Equal(suite.T(), kMsg.Topic, kMsg2.Topic)
			assert.Equal(suite.T(), kMsg.Partition, kMsg2.Partition)
			assert.Equal(suite.T(), kMsg.Offset, kMsg2.Offset)
		}
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMessageTestSuite(t *testing.T) {
	suite.Run(t, new(MessageSuite))
}
