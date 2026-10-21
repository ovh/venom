# Venom - Executor OVHAPI

## Step to test OVH API

Use case: your software needs to make calls to OVH API.<br>
You will need OVH credentials to make API calls. You can either use app keys authentication or OAuth2.

To use app keys authentication, please follow this tutorial: <br>
EN: https://docs.ovh.com/gb/en/customer/first-steps-with-ovh-api/

To use OAuth2, please follow this tutorial: <br>
EN: https://help.ovhcloud.com/csm/en-manage-service-account?id=kb_article_view&sysparm_article=KB0059343

## Input

The following parameters are available:

| Parameter         | Description                                                      | Default Value |
|-------------------|------------------------------------------------------------------|---------------|
| endpoint          | Optional                                                         | ovh-eu        |
| applicationKey    | Optional if `noAuth`, mandatory if using app keys authentication |               |
| applicationSecret | Optional if `noAuth`, mandatory if using app keys authentication |               |
| consumerKey       | Optional if `noAuth`, mandatory if using app keys authentication |               |
| clientID          | Optional if `noAuth`, mandatory if using OAuth2                  |               |
| clientSecret      | Optional if `noAuth`, mandatory if using OAuth2                  |               |
| noAuth            | Optional                                                         |               |
| headers           | Optional                                                         |               |
| resolve           | Optional                                                         |               |
| proxy             | Optional                                                         |               |
| tlsRootCA         | Optional                                                         |               |

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| method    | Optional    | GET           |
| path      | Mandatory   |               |
| body      | Optional    |               |
| bodyFile  | Optional    |               |
 
The first batch of parameters can also be defined inside Venom variables like this:

```yaml
vars:
  ovh.endpoint: ovh-eu
  ovh.applicationKey: foo
  ovh.applicationSecret: foo
  ovh.clientID: foo
  ovh.clientSecret: foo
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

## Custom resolution

`resolve` (or the `ovh.resolve` variable) is a list of
`hostname:port:destination` mappings. The source hostname (or IPv4 address) and
port must match the dialed address exactly; the source port is retained:

```yaml
resolve:
  - "ipv4.example:443:127.0.0.1"
  - "backend.example:443:localhost"
  - "collector.local:8443:::1"
  - "ipv6.example:443:2001:db8::1"
  - "bracketed.example:443:[::1]"
```

Destinations may be bare or enclosed in one pair of brackets. IPv6 zone
identifiers are supported, for example `service.local:443:fe80::1%eth0` or
`service.local:443:[fe80::1%eth0]`. The zone must exist on the machine running Venom.
Do not include a destination port. IPv6 literals are supported as destinations,
not as source fields in mappings. Unmatched dial addresses, including IPv6
addresses, are unchanged.

All three fields must be non-empty. Malformed IPv6 destinations produce an error;
hostname lookup and port validation are performed by Go's network dialer.
Mappings are applied in order, so a later mapping can match an earlier mapping's
destination hostname. Only the connection address changes: HTTP Host headers,
TLS certificate verification, and SNI continue to use the request's hostname
(or an explicitly configured Host header for HTTP). When using a proxy, mappings
apply to the address Venom dials, which may be the proxy.

## Example of an __ovhapi__ TestSuite

### Using App Keys authentication

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

### Using OAuth2

```yaml
name: Title of TestSuite
testcases:
- name: me
  steps:
  - type: ovhapi
    endpoint: 'ovh-eu'
    clientID: 'CLIENT_ID'
    clientSecret: 'CLIENT_SECRET'
    method: GET
    path: /me
    retry: 3
    delay: 2
    assertions:
    - result.statuscode ShouldEqual 200
    - result.bodyjson.nichandle ShouldContainSubstring MY_NICHANDLE
```

## Output

The following output fields are available:

| Field              | Description                          |
|--------------------|--------------------------------------|
| result.executor    |                                      |
| result.timeseconds | Execution duration                   |
| result.statuscode  | Status Code of HTTP response         |
| result.body        | Body of HTTP response                |
| result.bodyjson    | Body of HTTP response if it's a JSON |
| result.err         | Error message if exists              |

Note that you can access json data as `result.bodyjson.yourkey` for example.

## Default assertion

```yaml
result.statuscode ShouldEqual 200
```
