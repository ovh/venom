# Venom - Executor Kafka

Step to use read / write on a Kafka topic. We also have possibility to use Avro schema to encode message in Kafka Topic.

## Input

In your yaml file, you can use:

```yaml
  - addrs mandatory
  - with_tls optional
  - with_sasl optional
  - with_sasl_handshaked optional
  - with_avro optional - describes if this test should expect Avro schema to be used. NOTE if you used it for consumer, you will have to use it for Producer too.
  - user optional
  - password optional
  - kafka_version optional, defaut is 0.10.2.0

  - client_type mandator: producer or consumer

  # for consumer client type:
  - group_id mandatory
  - topics mandatory
  - timeout optional
  - message_limit optional
  - initial_offset optional - Sarama default is newest
  - mark_offset optional

  # for producer client type:
  - messages
  - messages.topic - Topic where to post message
  - messages.value - Value for message
  - messages.valueFile - Take value for message from file provided here
  - messages.avroSchemaFile - Specify Avro schema file. messages.valueFile or messages.value should have value, which can be encoded with that schema
```

Example without Avro:

```yaml
name: My Kafka testsuite
version: "2"
testcases:
- name: Kafka test
  description: Test Kafka
  steps:
  - type: kafka
    clientType: producer
    withSASL: true
    withTLS: true
    user: "{{.kafkaUser}}"
    password: "{{.kafkaPwd}}"
    addrs:
      - "{{.kafkaHost}}:{{.kafkaPort}}"
    messages:
    - topic: test-topic
      value: '{"hello":"bar"}'
  - type: kafka
    clientType: consumer
    withTLS: true
    withSASL: true
    user: "{{.kafkaUser}}"
    password: "{{.kafkaPwd}}"
    markOffset: true
    initialOffset: oldest
    messageLimit: 1
    groupID: venom
    addrs:
      - "{{.kafkaHost}}:{{.kafkaPort}}"
    topics:
      - test-topic
    assertions:
    - result.messagesjson.messagesjson0.value.hello ShouldEqual bar
    - result.messages.__len__ ShouldEqual 1

```

Example with Avro:

```yaml
name: My Kafka testsuite
version: "2"
testcases:
- name: Kafka test
  description: Test Kafka
  steps:
  - type: kafka
    clientType: producer
    withSASL: true
    withTLS: true
    user: "{{.kafkaUser}}"
    password: "{{.kafkaPwd}}"
    addrs:
      - "{{.kafkaHost}}:{{.kafkaPort}}"
    messages:
    - topic: test-topic
      valueFile: "kafka/values/message2.json"
      avroSchemaFile: "kafka/schemas/message.avsc"
  - type: kafka
    clientType: consumer
    withTLS: true
    withSASL: true
    user: "{{.kafkaUser}}"
    password: "{{.kafkaPwd}}"
    markOffset: true
    initialOffset: oldest
    messageLimit: 1
    groupID: venom
    addrs:
      - "{{.kafkaHost}}:{{.kafkaPort}}"
    topics:
      - test-topic
    assertions:
    - result.messagesjson.messagesjson0.value.id ShouldEqual 1
    - result.messagesjson.messagesjson0.value.message ShouldEqual "Some test"
    - result.messages.__len__ ShouldEqual 1
```
