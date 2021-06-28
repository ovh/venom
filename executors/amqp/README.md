# Venom - Executor AMQP

Step to publish / subscribe to AMQP 1.0 compatible broker (currently implemented by QPID and ActiveMQ, among others)

## Input

```yaml
- addr         (address of the amqp broker to connect to, in the format "amqp://<host>:<port>")
- clientType   (consumer or producer)

# Consumer Parameters
- sourceAddr   (source topic/queue to consume messages from)
- messageLimit (number of messages to read from the broker before returning result)

# Producer Parameters
- targetAddr   (topic/queue to which messages should be published)
- messages     (array of message bodies to send to the broker)
```

## Output

*Populated when ClientType is consumer*

```yaml
- result.messages     (array of strings, each containing the body of a response message)
- result.messagesJSON (if response is JSON, corresponding index will be populated with the navigable body of a response)
```
## Examples

### Publisher
```yaml
name: AMQP
testcases:
  - name: Producer
    steps:
      - type: amqp
        addr: amqp://localhost:5673
        clientType: producer
        targetAddr: amqp-test
        messages:
          - '{"key1":"value1","key2":"value2"}'
          - '{"key3":"value3","key4":"value4"}'
          - 'not json'
          - '["value5","value6"]'
```

### Consumer
```yaml
name: AMQP
testcases:
  - name: Consumer
    steps:
     - type: amqp
        addr: amqp://localhost:5673
        clientType: consumer
        sourceAddr: amqp-test
        messageLimit: 4
        assertions:
          - result.messages.__len__ ShouldEqual 4
          - result.messages.messages0 ShouldEqual '{"key1":"value1","key2":"value2"}'
          - result.messages.messages1 ShouldEqual '{"key3":"value3","key4":"value4"}'
          - result.messages.messages2 ShouldEqual 'not json'
          - result.messages.messages3 ShouldEqual '["value5","value6"]'
          - result.messagesjson.__len__ ShouldEqual 4
          - result.messagesjson.messagesjson0.key1 ShouldEqual value1
          - result.messagesjson.messagesjson0.key2 ShouldEqual value2
          - result.messagesjson.messagesjson1.key3 ShouldEqual value3
          - result.messagesjson.messagesjson1.key4 ShouldEqual value4
          - result.messagesjson.messagesjson3.messagesjson30 ShouldEqual value5
          - result.messagesjson.messagesjson3.messagesjson31 ShouldEqual value6
```
