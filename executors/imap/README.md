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
    mbox: INBOX
    mboxifsuccess: mailsMatches
    searchsubject: Title of mail
    assertions:
    - result.err ShouldNotExist
```

* mbox: optional, default is INBOX
* mboxifsuccess: optional. If not empty, move found mail (matching criteria) to another mbox.

## Output

* result.err is there is an arror.
* result.subject: subject of searched mail
* result.body: body of searched mail

## Default assertion

```yaml
result.err ShouldNotExist
```
