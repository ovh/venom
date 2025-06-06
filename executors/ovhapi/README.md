# Venom - Executor OVHAPI

## Step to test OVH API

Use case: you software need to make call to OVH API.<br>
You will need OVH credentials to make API call. Please follow this tutorial to get all needed keys: <br>
EN: https://docs.ovh.com/gb/en/customer/first-steps-with-ovh-api/

## Input
In your yaml file, you can use:

```
  - endpoint optional, default value: ovh-eu
  - applicationKey optional, if noAuth, otherwise mandatory
  - applicationSecret optional, if noAuth, otherwise mandatory
  - consumerKey optional, if noAuth, otherwise mandatory
  - noAuth optional
  - headers optional
  - resolve optional
  - proxy optional
  - tlsRootCA optional

  - method optional, default value: GET
  - path mandatory, example "/me"
  - body optional
  - bodyFile optional
```

The first batch of parameters can also be defined inside Venom variables like this

```yaml
vars:
  ovh.endpoint: ovh-eu
  ovh.applicationKey: foo
  ovh.applicationSecret: foo
  ovh.consumerKey: foo
  ovh.noAuth: false
  ovh.headers:
    x-foo: foo
  ovh.resolve:
  - example.org:443:127.0.0.1
  ovh.proxy: localhost:8000
  ovh.tlsRootCA: |-
    -----BEGIN CERTIFICATE-----
    MIIF3jCCA8agAwIBAgIQAf1tMPyjylGoG7xkDjUDLTANBgkqhkiG9w0BAQwFADCB
    ...
    -----END CERTIFICATE-----
```

## Example of an __ovhapi__ TestSuite
```yaml
name: Title of TestSuite
testcases:
- name: me
  steps:
  - type: ovhapi
    endpoint: 'ovh-eu'
    applicationKey: 'APPLICATION_KEY'
    applicationSecret: 'APPLICATION_SECRET'
    consumerKey: 'CONSUMER_KEY'
    method: GET
    path: /me
    retry: 3
    delay: 2
    assertions:
    - result.statuscode ShouldEqual 200
    - result.bodyjson.nichandle ShouldContainSubstring MY_NICHANDLE

```

## Output

```
result.executor
result.timeseconds
result.statuscode
result.body
result.bodyjson
result.err
```
- result.timeseconds: execution duration
- result.err: if exists, this field contains error
- result.body: body of HTTP response
- result.bodyjson: body of HTTP response if it's a json. You can access json data as result.bodyjson.yourkey for example
- result.statuscode: Status Code of HTTP response

## Default assertion

```yaml
result.statuscode ShouldEqual 200
```
