# Venom - Executor IMAP

Use case: you have a mail server and you want to test IMAP commands ?

Venom IMAP executor implements a few IMAP commands such as FETCH, APPEND, MOVE, STORE...

## How to test
### Authentication

```yaml
auth:
  withtls: false
  host: yourimaphost
  port: 143 # Most probably 993 if using TLS
  user: imap@venom.com
  password: imapvenompassword
```

### Commands

As of today, these are the available commands: 
- **append**: create a new mail at the end of a mailbox
- **create**: create a new mailbox
- **clear**: delete every message in a mailbox from the mail server
- **delete**: delete a message
- **fetch**: retrieve a mail
- **flag**: add, remove or set the flags of a mail
- **move**: move a mail from one mailbox to another

Multiple commands can be stated as once: they will be executed in the order they are presented under the `commands` field.

Each command consists of a `name` field which allows to specify the command to execute.
Then they are composed of the `search` field and/or the `args` field.

The `search` field always has the same syntax. It retrieves and returns the first mail found through the search criteria:
```yaml
search:
  mailbox: testBox # The mailbox in which to search the mail
  uid: 1 # The UID to search for
  from: .*@your-domain.localhost # The "From" header field to search for
  to: you@company.tld # The "To" header field to search for
  subject: Title of mail with * # The "Subject" header field to search for
  body: .*a body content.* # The "Body" field to search for
```
It allows us to specify which mail the command will act upon.
> ⚠️ Be careful as search criteria are regular expressions. If you wish to search for special characters, don't forget to escape them: \
> ❌ `subject: [IMPORTANT] READ THIS` \
> ✅ `subject: \[IMPORTANT\] READ THIS`

Often combined to the search field, the `args` field allows us to specify the arguments to the command.
Its syntax depends on the command.

In this section, we will describe each command syntax.

#### Append command

The append command creates a new mail at the end of a mailbox.

```yaml
commands:
  - name: append
    args:
      mailbox: mailboxName # MANDATORY: The mailbox in which to create the mail
      from: origin-of@-my-new-mail.com # MANDATORY: the "From" header field of the new mail
      to: recipient-of@my-new-mail.com # OPTIONAL: the "To" header field of the new mail
      subject: Subject of my new mail # OPTIONAL: the "Subject" header field of the new mail
      body: Body of my new mail # OPTIONAL: the body of the new mail
      flags: # OPTIONAL: the flags of the new mail (note that the "\Recent" flag will most probably be added as well)
        - Flag1
        - Flag2
        ...
```

#### Create command

The create command creates a new mailbox.

```yaml
commands:
  - name: create
    args:
      mailbox: mailboxName # MANDATORY: The name of the new mailbox to create
```

#### Clear command

The clear command deletes every message in a mailbox. It can be called in two different ways: 
- First, by listing every mailbox to clear:
```yaml
commands:
  - name: clear
    args:
      mailboxes:
        - Mailbox1
        - Mailbox2
        ...
```
- Second, by using the wildcard argument to clean all the existing mailboxes:
```yaml
commands:
  - name: clear
    args:
      mailboxes:
        - "*"
```

#### Delete command

The delete command permanently deletes a message retrieved through the search field.

```yaml
commands:
  - name: delete
    search:
      [...]
```

#### Fetch command

The fetch command retrieves a message through the search field.

```yaml
commands:
  - name: fetch
    search:
      [...]
```

#### Flag command

The flag command modifies the flags of a message. It is made up of both `search` and `args` fields.
The `args` field can be composed of up to 3 different fields:
- **add**: flags to add to the mail
- **remove**: flags to remove from the mail
- **set**: the mail flags to set (overrides any existing flag)

All these fields can be called at once but, if specified, only `set` will be taken into account.
They all have the same syntax.
```yaml
commands:
  - name: flag
    args:
      add:
        - Flag1
        - Flag2
        ...
      remove:
        - Flag1
        - Flag2
        ...
      set:
        - Flag1
        - Flag2
        ...
```

#### Move command

The move command moves a mail from one mailbox to another. It is made up of both `search` and `args` fields.


```yaml
commands:
  - name: move
    search:
      [...]
    args:
      mailbox: mailboxname # The mailbox to move the mail to
```

### Assertions

Each execution of a command produces a `result` Its content depends on the command and its execution.\
Taking that multiple commands can be executed in a single testcase, the `result` object follows this syntax:
```json
{
  "result": {
    "commands": [
      {
        // Search represents the result of the command's search field
        "search": {
          "mailbox": "INBOX",
          "from": "from@mail-before-command-execution.com",
          "to": "to@mail-before-command-execution.com",
          "subject": "Title of mail BEFORE command execution",
          "body": "Body content of mail BEFORE command execution",
          "flags": [
            "Flag1",
            "Flag2"
          ]
        },
        // Mail represents the state of the searched mail after the command was executed
        "mail": { 
          "from": "from@mail-after-command-execution.com",
          "to": "to@mail-after-command-execution.com",
          "subject": "Title of mail AFTER command execution",
          "body": "Body content of mail AFTER command execution",
          "flags": [
            "Flag1",
            "Flag2",
            "Flag3"
          ]
        },
        "err": "Error of the command", // Err represents the command error message, supposing the command failed
        "timeseconds": 1.5 // TimeSeconds represents the duration of the command execution
      }
    ]
  }
}
```
Where each command has its own results that can be asserted.

As an example, here is a list of possible assertions after execution of two commands in the same testcase:
```yaml
assertions:
  # First ommand
  - result.commands.commands0.err ShouldBeEmpty
  # State of the mail before command execution (search)
  - result.commands.commands0.search.from ShouldEqual "from@mail-before-command-execution.com"
  # Mail as a result of command execution
  - result.commands.commands0.mail.from ShouldEqual "to@mail-after-command-execution.com"
  # Second command
  - result.commands.commands1.err ShouldBeEmpty
  - result.commands.commands1.search.from ShouldEqual ...
  - result.commands.commands1.mail.from ShouldEqual ...
  - result.commands.commands1.mail.flags.flags0 ShouldEqual ...
  - result.commands.commands1.mail.flags.flags1 ShouldEqual ...
```

The commands that do not modify a mail content (headers and body) won't have the `mail` field filled up.\
As an example, here is the typical result of a `fetch`, `delete` or `move` command:
```json
{
  "result": {
    "commands": [
      {
        "search": {
          "mailbox": "INBOX",
          "from": "from@mail-before-command-execution.com",
          "to": "to@mail.com",
          "subject": "Title of retrieved mail",
          "body": "Body content of retrieved mail",
          "flags": [
            "Flag1",
            "Flag2"
          ]
        },
        "err": "",
        "timeseconds": 1.5
      }
    ]
  }
}
```

As another example, here is the typical result of a `create` or `clear` command:
```json
{
  "result": {
    "commands": [
      {
        "search": {},
        "err": "",
        "timeseconds": 1.5
      }
    ]
  }
}
```

## Testsuite example

```yml
name: IMAP testsuite
vars:
  withTLS: false
  host: localhost
  port: 1143
  user: address@example.org
  password: pass
testcases:
  - name: Clear a mailbox
    steps:
      - type: imap
        auth:
          host: "{{.host}}"
          port: "{{.port}}"
          user: "{{.user}}"
          password: "{{.password}}"
        commands:
          - name: clear
            args:
              mailboxes:
                - INBOX
        assertions:
          # As multiple commands can be executed in a single testcase, we need to specify which command result we want to assert
          - result.commands.commands0.err ShouldBeEmpty
```

To see more examples, check the [IMAP executor test file](../../tests/imap.yml).
