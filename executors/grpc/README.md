# Venom - Executor Grpc

Step for execute GRPC Request

Based on `grpcurl`, see [grpcurl](https://github.com/fullstorydev/grpcurl) for more information.
This executor relies on the gRPC server reflection, which should be enabled on the server as described 
[here](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md). 
gRPC server reflection is not enabled by default and not implemented for every gRPC library,
make sure your library of choice supports reflection before implementing tests using this executor.
gRPC server reflection also does not properly work with `gogo/protobuf`: grpc/grpc-go#1873

## Tests

Results of test are parsed as json and saved in `systemoutjson`. Status codes correspond 
to the official status codes of gRPC.
You can find what individual return codes mean [here](https://github.com/grpc/grpc/blob/master/doc/statuscodes.md).

## Input

In your yaml file, you can use:

```yaml
  - url mandatory
  - service mandatory: service to call
  - method mandatory: list, describe, or method of the endpoint
  - data optional: data to marshal to json and send as a request
  - headers optional: data to send as additional headers
  - connect_timeout optional: The maximum time, in seconds, to wait for connection to be established. Defaults to 10 seconds
  - default_fields optional: whether json formatter should emit default fields
  - include_text_separator optional: when protobuf string formatter is invoked to format multiple messages, all messages after the first one will be prefixed with character (0x1E)
  - tls_client_cert optional: a chain of certificates to identify the caller, first certificate in the chain is considered as the leaf, followed by intermediates. Setting it enable mutual TLS authentication. Set the PEM content or the path to the PEM file.
  - tls_client_key optional: private key corresponding to the certificate. Set the PEM content or the path to the PEM file.
  - tls_root_ca optional: defines additional root CAs to perform the call. can contains multiple CAs concatenated together. Set the PEM content or the path to the PEM file.
  - ignore_verify_ssl optional: set to true if you use a self-signed SSL on remote for example
```

Example:

```yaml

name: Title of TestSuite
testcases:

- name: request GRPC
  steps:
  - type: grpc
    url: serverUrlWithoutHttp:8090
    data:
      foo: bar
    service: coolService.api
    method: GetAllFoos
    assertions:
    - result.code ShouldEqual 0
    - result.systemoutjson.foo ShouldEqual bar
```

Example TLS:

```yaml

name: Title of TestSuite
testcases:

- name: request GRPC
  steps:
  - type: grpc
    url: serverUrlWithoutHttp:8090
    tls_root_ca: |-
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    ignore_verify_ssl: true # true for self signed certificates
    data:
      foo: bar
    service: coolService.api
    method: GetAllFoos
    assertions:
    - result.code ShouldEqual 0
    - result.systemoutjson.foo ShouldEqual bar
```

Example mutual TLS:

```yaml

name: Title of TestSuite
testcases:

- name: request GRPC
  steps:
  - type: grpc
    url: serverUrlWithoutHttp:8090
    tls_root_ca: |-
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    tls_client_cert: |-
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    tls_client_key: |-
      -----BEGIN PRIVATE KEY-----
      ...
      -----END PRIVATE KEY-----
    ignore_verify_ssl: true # true for self signed certificates
    data:
      foo: bar
    service: coolService.api
    method: GetAllFoos
    assertions:
    - result.code ShouldEqual 0
    - result.systemoutjson.foo ShouldEqual bar
```

## Output

```yaml
executor
systemout
systemerr
err
code
timeseconds
```

- result.timeseconds: execution duration
- result.executor.executor.script: script executed
- result.err: if exists, this field contains error
- result.systemout: Standard Output of executed script
- result.systemerr: Error Output of executed script
- result.code: Exit Code
