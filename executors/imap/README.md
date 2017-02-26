# Venom - Executor IMAP

Use case: your software send a mail  ?
Venom can test if mail is received. Body of mail can be reused in further steps.

## Input

```yaml
name: TestSuite with IMAP Steps
testcases:
- name: TestCase IMAP
  steps:
  - type: imap
    imaphost: yourimaphost
    imapuser: yourimapuser
    imappassword: "yourimappassword"
    box: INBOX
    searchsubject: Title of mail
    assertions:
    - result.err ShouldNotExist
```

* box: default is INBOX

## Output

* result.err is there is an arror.
* result.subject: subject of searched mail
* result.body: body of searched mail

## Default assertion

```yaml
result.err ShouldNotExist
```
