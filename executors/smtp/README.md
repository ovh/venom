# Venom - Executor SMTP

Step for sending SMTP

## Input

```yaml
name: TestSuite with SMTP Steps
testcases:
- name: TestCase SMTP
  steps:
  - type: smtp
    withtls: true
    host: localhost
    port: 465
    user: yourSMTPUsername
    password: yourSMTPPassword
    to: destinationa@yourdomain.com,destinationb@yourdomain.com
    from: venom@localhost
    subject: title of mail
    body: body of mail
    assertions:
    - result.err ShouldNotExist
```

## Output

Noting, except result.err is there is an arror.

## Default assertion

```yaml
result.err ShouldNotExist
```
