# Venom - Executor Redis

Step to execute command into Redis

Use case: your software need to make call to a Redis.

## Input

The follwing inputs are available:
- `commands`: an array of Redis command
- `path`: a file which contains a series of Redis command. If path property is filled, commands property will be ignored.
- `dialURL`: Redis server URL

URL should follow the draft IANA specification for the scheme (https://www.iana.org/assignments/uri-schemes/prov/redis).
If you have multiple testcases or steps that use the same Redis URL you can define the `dialURL` setting once as a testsuite variable.

```
Commands file is read line by line and each command is split by [strings.Fields](https://golang.org/pkg/strings/#Fields) method

```text
SET Foo Bar
SET bar beez
SET Bar {"foo" : "bar", "poo" : ["lol", "lil", "greez"]}
Keys *
```

```yaml
name: Redis testsuite
vars: 
  dialURL: "redis://localhost:6379/0"
  
testcases:
- name: test-commands
  steps:
  - type: redis
    commands:
        - FLUSHALL
  - type: redis
    commands:
        - SET foo bar
        - GET foo
        - KEYS *
    assertions:
        - result.commands.commands0.response ShouldEqual OK
        - result.commands.commands1.response ShouldEqual bar
        - result.commands.commands2.response.response0 ShouldEqual foo
  - type: redis
    commands:
        - KEYS *
    assertions:
        - result.commands.commands0.response.response0 ShouldEqual foo

- name: test-commands-from-file
  steps:
  - type: redis
    path: testredis/commands.txt
    dialURL: "redis://localhost:6379/0" # The global dialURL is overriden by this setting
    assertions:
        - result.commands.commands0.response ShouldEqual OK
        - result.commands.commands1.response ShouldEqual bar
        - result.commands.commands2.response.response0 ShouldEqual foo
```

## Output

The executor returns a result object that contains the executed Redis command.

- result.commands contains the list of executed Redis command
- result.commands.commandI.response represents the response of Redis command. It can be an array or a string, depends of Redis command

## Examples

More example can be used in the `tests` folder of this repository.