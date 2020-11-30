# Venom - Executor HTTP

Step for execute a HTTP Request

## Input
In your yaml file, you can use:

```yaml
  - method optional, default value: GET
  - url mandatory
  - unix_sock optional
  - path optional
  - body optional
  - bodyFile optional
  - headers optional
  - proxy optional: set to use a proxy server for connection to url
  - ignore_verify_ssl optional: set to true if you use a self-signed SSL on remote for example
  - basic_auth_user optional: username to use for HTTP basic authentification
  - basic_auth_password optional: password to use for HTTP basic authentification
  - no_follow_redirect optional: indicates that you don't want to follow Location if server returns a Redirect (301/302/...)
  - skip_body: skip the body and bodyjson result
  - skip_headers: skip the headers result
  - tls_client_cert optional: a chain of certificates to identify the caller, first certificate in the chain is considered as the leaf, followed by intermediates. Setting it enable mutual TLS authentication
  - tls_client_key optional: private key corresponding to the certificate.
  _ tls_root_ca optional: defines additional root CAs to perform the call. can contains multiple CAs concatenated together

```

```yaml

name: HTTP testsuite
testcases:
- name: get http testcase
  steps:
  - type: http
    method: GET
    url: https://eu.api.ovh.com/1.0/
    assertions:
    - result.body ShouldContainSubstring /dedicated/server
    - result.body ShouldContainSubstring /ipLoadbalancing
    - result.statuscode ShouldEqual 200
    - result.bodyjson.api ShouldBeNil
    - result.bodyjson.apis ShouldNotBeEmpty
    - result.bodyjson.apis.apis0 ShouldNotBeNil
    - result.bodyjson.apis.apis0.path ShouldEqual /allDom

- name: post http multipart
  steps:
  - type: http
    method: POST
    url: https://eu.api.ovh.com/1.0/auth/logout
    multipart_form:
      file: '@./venom.gif'
    assertions:
    - result.statuscode ShouldEqual 401

- name: post http enhanced assertions
  steps:
  - type: http
    method: POST
    url: https://eu.api.ovh.com/1.0/newAccount/rules
    assertions:
      - result.statuscode ShouldEqual 200
      - result.bodyjson.__type__ ShouldEqual Array
      # Ensure a minimum of fields are present.
      - result.bodyjson.__len__ ShouldBeGreaterThanOrEqualTo 8
      # Ensure fields have the right keys.
      - result.bodyjson.bodyjson0 ShouldContainKey fieldName
      - result.bodyjson.bodyjson0 ShouldContainKey mandatory
      - result.bodyjson.bodyjson0 ShouldContainKey regularExpression
      - result.bodyjson.bodyjson0 ShouldContainKey prefix
      - result.bodyjson.bodyjson0 ShouldContainKey examples
      - result.bodyjson.bodyjson0 ShouldNotContainKey lol
      - result.statuscode ShouldNotEqual {{.post-http-multipart.result.statuscode}}

- name: get http (with options)
  steps:
  - type: http
    method: POST
    url: https://eu.api.ovh.com/1.0
    skip_body: true
    skip_headers: true
    info: request is {{.result.request.method}} {{.result.request.url}} {{.result.request.body}}
    assertions:
      - result.statuscode ShouldEqual 405
      - result.body ShouldBeEmpty
      - result.headers ShouldBeEmpty


```
*NB: to post a file with multipart_form, prefix the path to the file with '@'*

## Output

```
result.request
result.timeseconds
result.statuscode
result.body
result.bodyjson
result.headers
result.err
```
- result.timeseconds: execution duration
- result.request.method: HTTP method of the request
- result.request.url: HTTP URL of the request
- result.request.body: body content as string
- result.request.form: HTTP form map
- result.request.post_form: HTTP post form map
- result.err: if exists, this field contains error
- result.body: body of HTTP response
- result.bodyjson: body of HTTP response if it's a JSON. You can access json data as result.bodyjson.yourkey for example.
- result.headers: headers of HTTP response
- result.statuscode: Status Code of HTTP response

### JSON keys

JSON keys are lowercased automatically (eg. use `result.bodyjson.yourkey`, not
`result.bodyjson.YourKey`).

On top of that, if a JSON key contains special characters, they will be translate to underscores.

### JSON arrays

When a HTTP response contains a JSON array, you have to use following syntax
to access specific key of an array: `result.bodyjson.array_name.array_name_index_in_array.key`

Example if you want to get value of `path` key of *second* element in `apis` array: `result.bodyjson.apis.apis1.path`


## Default assertion

```yaml
result.statuscode ShouldEqual 200
```
