# Venom - Executor SMTP

Step for sending SMTP

Use case: you software have to check mails for doing something with them?
Venom can send mail then execute some tests on your software.

## Input

```yaml
name: TestSuite with SMTP Steps
testcases:
- name: TestCase SMTP
  steps:
  - type: smtp
    withtls: false
    host: localhost
    port: 25 # 465 if using TLS
    user: yourSMTPUsername # Optional, only works with TLS
    password: yourSMTPPassword # Optional, only works with TLS
    from: venom@smtp.net
    to: destinationa@yourdomain.com,destinationb@yourdomain.com
    subject: Title of mail
    body: Body of mail
    assertions:
    - result.err ShouldBeEmpty
```

## Output

Nothing, except result.err if there is an error.

## Default assertion

```yaml
result.err ShouldBeEmpty
```
