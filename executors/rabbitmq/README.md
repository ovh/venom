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
  - durable optional      (true or false) (default alse)
  - contentType optional  
  - persistent optional (default true)
  - messages
  - messages_file

```

Example:

```yaml

```
