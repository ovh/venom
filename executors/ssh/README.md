# Venom - Executor SSH

Step for execute a script on remote server via SSH


## Input

In your yaml file, you can use:

```yaml
  - host mandatory
  - command mandatory
  - user optional (default is OS username)
  - password optional (mandatory if no privatekey is found)
  - privatekey optional (default is $HOME/.ssh/id_rsa)
  - sudo optional
  - sudopassword optional (default to password)
```

Example

```yaml

name: Title of TestSuite
testcases:
- name: Check if exit code != 1 and echo command response in less than 1s
  steps:
  - type: ssh
    host: localhost:2222
    command: echo 'foo'
    assertions:
    - result.code ShouldEqual 0
    - result.timeseconds ShouldBeLessThan 1

- name: Use specific privatekey
  steps:
  - type: ssh
    host: 10.0.1.5:2222
    command: echo 'foo'
    user: bar
    privatekey: /home/foo/.ssh/id_rsa
    assertions:
    - result.code ShouldEqual 0

- name: Execute command as another user than bar
  steps:
  - type: ssh
    host: 10.0.1.5:2222
    command: echo 'foo'
    user: bar
    sudo: root
    sudopassword: '{{.mypassword}}'
    assertions:
    - result.code ShouldEqual 0

```
*NB: Sudo option uses a pseudotty*

## Output

```yaml
systemout
systemerr
err
code
timeseconds
```

- result.timeseconds: time of execution
- result.err: if exists, this field contains error
- result.systemout: Standard Output of executed script
- result.systemerr: Error Output of executed script
- result.code: Exit Code

## Default assertion

```yaml
result.code ShouldEqual 0
```
