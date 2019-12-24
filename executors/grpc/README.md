# Venom - Executor Grpc

Step for execute GRPC Request

Based on `grpcurl`, see [grpcurl](https://github.com/fullstorydev/grpcurl) for more information.
This executor relies on the gRPC server reflection, which should be enabled on the server as described 
[here](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md). 
gRPC server reflection is not enabled by default and not implemented for every gRPC library,
make sure your library of choice supports reflection before implementing tests using this executor.
gRPC server reflection also does not properly work with `gogo/protobuf`: grpc/grpc-go#1873

## Tests

Results of test are parsed as json and saved in `bodyjson`. Status codes correspond 
to the official status codes of gRPC.
You can find what individual return codes mean [here](https://github.com/grpc/grpc/blob/master/doc/statuscodes.md).

## Input

In your yaml file, you can use:

```yaml
  - url mandatory
  - service mandatory: service to call
  - method mandatory: list, describe, or method of the endpoint
  - plaintext optional: use plaintext protocol instead of TLS
  - data optional: data to marshal to json and send as a request
  - headers optional: data to send as additional headers
  - connect_timeout optional: The maximum time, in seconds, to wait for connection to be established. Defaults to 10 seconds.
  - default_fields optional: whether json formatter should emit default fields
  - include_text_separator optional: when protobuf string formatter is invoked to format multiple messages, all messages after the first one will be prefixed with character (0x1E).
```

Example:

```yaml

name: Title of TestSuite
testcases:

- name: request GRPC
  steps:
  - type: grpc
    url: serverUrlWithoutHttp:8090
    plaintext: true # skip TLS
    data:
      foo: bar
    service: coolService.api
    method: GetAllFoos
    assertions:
    - result.code ShouldEqual 0
    - result.systemoutjson.foo ShouldEqual bar
  - type: grpc
    url: serverUrlWithoutHttp:8090
    plaintext: true # skip TLS
    stream: foo.json
    service: coolService.api
    method: StreamAllFoos
    assertions:
    - result.code ShouldEqual 0
    - result.err ShouldBeEmpty

```

## Output

```yaml
executor
systemout
systemoutjson
systemerr
systemerrjson
err
code
timeseconds
timehuman
```

- result.timeseconds & result.timehuman: time of execution
- result.executor.executor.script: script executed
- result.err: if exists, this field contains error
- result.systemout: Standard Output of executed script
- result.systemerr: Error Output of executed script
- result.code: Exit Code

## Streaming

Stream JSON file must be an a array of chunks. For instance, if your proto message is:

```
message BodyChunk {
  bytes Foo = 1;
}
```

then, you json file must be:

```json
[
  {
    "Foo": "first part of stream"
  },
  {
    "Foo": "second part of stream"
  }
]
```

