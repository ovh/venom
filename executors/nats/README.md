# Venom - Executor NATS

Step to publish and subscribe to NATS subjects.

## Input

### Defaults

This step includes some default values:

- `url`: defaults to `nats://localhost:4222`
- `messageLimit`: defaults to 1
- `deadline`: defaults to 1 second

### Authentication

This step allows for connection with and without TLS. Without TLS, the step does not require additional options.

To connect to a NATS server with TLS, declare:

```yaml
tls:
  selfSigned: true
  serverVerify: true
  certificatePath: "/path/to/client_certificate"
  keyPath: "/path/to/client_key"
  caPath: ""/path/to/ca_certificate""
```

Enable `selfSigned` only if the NATS server uses self-signed certificates. If enabled, `caPath` is mandatory.

Enable `serverVerify` only if the NATS server verifies the client certificates. If enabled `certificatePath` and `keyPath` are mandatory.

### publish

The publish command allows to publish a payload to a specific NATS subject. Optionally it can wait for a reply.

Full configuration example:

```yaml
- type: nats
  url: "{{.url}}" # defaults to nats://localhost:4222 if not set
  command: publish
  subject: "{{.subject}}" # mandatory
  payload: '{{.message}}'
  headers:
    customHeader:
      - "some-value"
  assertions:
    - result.error ShouldBeEmpty
```

Full configuration with reply example:

```yaml
- type: nats
  url: "{{.url}}" # defaults to nats://localhost:4222 if not set
  command: publish
  request: true
  subject: "{{.subject}}" # mandatory
  replySubject: "{{.subject}}.reply" # mandatory if `request = true`
  payload: '{{.message}}'
  assertions:
    - result.error ShouldBeEmpty
    - result.messages.__Len__ ShouldEqual 1
```

It is possible to publish to a Jetstream stream by declaring `jetstream: true` in the step.

 For example:

```yaml
- type: nats
  command: publish
  subject: "{{.subject}}.hello" # mandatory
  deadline: 2
  jetstream:
    enabled: true
  assertions:
    - result.error ShouldNotBeEmpty
```

### subscribe

The subscribe command allows to receive messages from a subject or a stream.

Full configuration example:

```yaml
- type: nats
  command: subscribe
  subject: "{{.subject}}.>" # mandatory
  messageLimit: 2 # defaults to 1
  deadline: 10 # in seconds, defaults to 1
  assertions:
    - result.error ShouldBeEmpty
    - result.messages.__Len__ ShouldEqual 2
```

Full configuration example with Jetstream:

```yaml
- type: nats
  command: subscribe
  subject: "{{.subject}}.>" # mandatory
  messageLimit: 2 # defaults to 1
  deadline: 10 # in seconds, defaults to 1
  jetstream:
    enabled: true
    stream: TEST # mandatory, stream must exist
    filterSubjects:
      - "{{.subject}}.js.hello"
      - "{{.subject}}.js.world"
  assertions:
    - result.error ShouldBeEmpty
    - result.messages.__Len__ ShouldEqual 2
```
