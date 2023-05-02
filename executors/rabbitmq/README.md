# Venom - Executor RabbitMQ

Step to use publish / subscribe on a RabbitMQ

## Input
In your yaml file, you can use:

```yaml

  # RabbitMQ connection
  - addrs optional        (default amqp:/localhost:5672)
  - user optional         (default guest)
  - password optional     (default guest)

  - clientType mandatory (publisher or subscriber)

  # RabbitMQ Q configuration
  - qName mandatory

  # Exchange configuration
  - routingKey optional   (default qName)
  - exchangeType optional  (default "fanout")
  - exchange optional     (default "")

  # For subscriber only
  - messageLimit optional (default 1)

  # For publisher only
  - messages
    - durable optional      (true or false) (default false)
    - contentType optional  
    - contentEncoding optional
    - persistent optional (default true)
    - headers optional
      - name: value

```

## Examples:

### Publisher (workQ)
```yaml
name: TestSuite RabbitMQ
vars:
  addrs: 'amqp://localhost:5672'
  user: 
  password: 
testcases:
  - name: RabbitMQ publish (work Q)
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        clientType: publisher
        qName: TEST
        messages: 
          - value: '{"a": "b"}'
            contentType: application/json
            contentEncoding: utf8
            persistent: false
            headers: 
              myCustomHeader: value
              myCustomHeader2: value2
```

### Subscriber (workQ)
```yaml
name: TestSuite RabbitMQ
vars:
  addrs: 'amqp://localhost:5672'
  user: 
  password: 
  - name: RabbitMQ subscribe testcase
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        clientType: subscriber
        qName: "{{.qName}}"
        durable: true
        messageLimit: 1
        assertions: 
          - result.bodyjson.bodyjson0.a ShouldEqual b   
          - result.headers.headers0.mycustomheader ShouldEqual value
          - result.headers.headers0.mycustomheader2 ShouldEqual value2
          - result.messages.messages0.contentencoding ShouldEqual utf8
          - result.messages.messages0.contenttype ShouldEqual application/json
```

### Publisher (pubsub)
```yaml
name: TestSuite RabbitMQ
vars:
  addrs: 'amqp://localhost:5672'
  user: 
  password: 
testcases:
  - name: RabbitMQ publish (work Q)
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        clientType: publisher
        exchange: exchange_test
        routingKey: pubsub_test
        messages: 
          - value: '{"a": "b"}'
            contentType: application/json
            contentEncoding: utf8
            persistent: false
            headers: 
              myCustomHeader: value
              myCustomHeader2: value2
```

### Subscriber (pubsub)
```yaml
name: TestSuite RabbitMQ
vars:
  addrs: 'amqp://localhost:5672'
  user: 
  password: 
  - name: RabbitMQ subscribe testcase
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        clientType: subscriber
        exchange: exchange_test
        routingKey: pubsub_test
        messageLimit: 1
        assertions: 
          - result.bodyjson.bodyjson0.a ShouldEqual b   
          - result.headers.headers0.mycustomheader ShouldEqual value
          - result.headers.headers0.mycustomheader2 ShouldEqual value2
          - result.messages.messages0.contentencoding ShouldEqual utf8
          - result.messages.messages0.contenttype ShouldEqual application/json
```
