# Venom - Executor Exec

Step used to execute a script


## Input

Example

```yaml

name: Title of TestSuite
testcases:
- name: Check if exit code != 1 and echo command response in less than 1s
  steps:
  - type: exec
    script: echo 'foo'
    assertions:
    - result.code ShouldEqual 0
    - result.timeseconds ShouldBeLessThan 1

```

Multiline script:

```yaml
name: Title of TestSuite
testcases:
- name: multiline script
  steps:
  - type: exec
    script: |
            echo "Foo" \
            echo "Bar"
```

## Output

```yaml
systemout
systemoutjson
systemerr
systemerrjson
err
code
timeseconds
```

- result.timeseconds: duration of execution
- result.err: if exists, this field contains error details
- result.systemout: Standard Output of the executed script
- result.systemoutjson: Standard Output of the executed script parsed as a JSON object
- result.systemerr: Error output of the executed script
- result.systemerrjson: Error output of the executed script parsed as a JSON object
- result.code: Exit code

## Default assertion

```yaml
result.code ShouldEqual 0
```
