# Venom - Executor RabbitMQ

Four types of execution are supported:
- **publisher**: publish a message to a queue or to an exchange.
- **subscriber**: bind to a queue or an exchange (using routing key) and wait for message(s) to be consumed.
- **client**: publish a request message to a queue or to an exchange and wait for the reply to be received on the [reply-to](https://www.rabbitmq.com/docs/direct-reply-to) queue.
- **server**: bind to a queue or an exchange (using routing key) and send a reply over the [reply-to](https://www.rabbitmq.com/docs/direct-reply-to) queue when a request is received.

Steps to use publish / subscribe on a RabbitMQ:

## Input
In your yaml file, you can use:

```yaml

  # RabbitMQ connection
  - addrs optional        (default amqp:/localhost:5672)
  - user optional         (default guest)
  - password optional     (default guest)

  - clientType mandatory (publisher, subscriber or client)

  # RabbitMQ Q configuration
  - qName mandatory

  # Exchange configuration
  - routingKey optional   (default qName)
  - exchangeType optional  (default "fanout")
  - exchange optional     (default "")

  # For subscriber and client only
  - messageLimit optional (default 1)

  # For publisher and client only
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
  qName: 'consumer-queue'
testcases:  
  - name: RabbitMQ subscribe testcase
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        clientType: subscriber
        exchange: exchange_test
        exchangeType: direct
        durable: true
        qName: "{{.qName}}"
        routingKey: pubsub_test
        messageLimit: 1
        assertions: 
          - result.bodyjson.bodyjson0.a ShouldEqual b   
          - result.headers.headers0.mycustomheader ShouldEqual value
          - result.headers.headers0.mycustomheader2 ShouldEqual value2
          - result.messages.messages0.contentencoding ShouldEqual utf8
          - result.messages.messages0.contenttype ShouldEqual application/json
```

### Client (pubsub RPC)
```yaml
name: TestSuite RabbitMQ
vars:
  addrs: 'amqp://localhost:5672'
  user: 
  password: 
testcases:
  - name: RabbitMQ request/reply
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        clientType: client
        exchange: exchange_test
        exchangeType: direct
        durable: true
        routingKey: order-query
        messages: 
          - value: '{"OrderId": "ORDER-12345"}'
            contentType: application/json
            contentEncoding: utf8
            persistent: false
            headers: 
              myCustomHeader: value
              myCustomHeader2: value2
        messageLimit: 1
        assertions: 
          - result.bodyjson.bodyjson0 ShouldContainKey OrderStatus
          - result.bodyjson.bodyjson0.OrderStatus ShouldEqual Pending
```

### Server (pubsub RPC)
Use the _assertions_ to validate the request. Note the reply will be sent regardless of validation result.
Use the _messages_ to define reply payload(s).
For convenience, each reply message includes an _x-request-messageid_ header populated with the _MessageId_ property of the request message.
```yaml
name: TestSuite RabbitMQ
vars:
  addrs: 'amqp://localhost:5672'
  user: 
  password: 
  qName: 'order-query-handler'
testcases:
  - name: RabbitMQ request/reply
    steps:
      - type: rabbitmq
        addrs: "{{.addrs}}"
        user: "{{.user}}"
        password: "{{.password}}"
        qName: "{{.qName}}"
        clientType: server
        exchange: exchange_test
        exchangeType: direct
        durable: true
        routingKey: order-query
        messages: 
          - value: '{"Status": "OK", "OrderDate": "2024/11/07", "OrderStatus": "Pending"}'
            contentType: application/json
            contentEncoding: utf8
            persistent: false
            headers: 
              myCustomHeader: value
              myCustomHeader2: value2
        messageLimit: 1
        assertions: 
          - result.bodyjson.bodyjson0 ShouldContainKey OrderId
```
