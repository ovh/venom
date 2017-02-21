# Venom - Executor SMTP

Step for Read file

Use case: you software write a file. Venom checks that file is produced, read it,
and return content. Content can be used by another steps of testsuite.

path can contains a fullpath, a wildcard or a directory:

```
- path: /a/full/path/file.txt
- path: afile.txt
- path: *.yml
- path: a_directory/
```

## Input

```yaml
name: TestSuite Read File
testcases:
- name: TestCase Read File
  steps:
  - type: readfile
    path: yourfile.txt
    assertions:
    - result.err ShouldNotExist
```

## Output

```yaml
  result.executor
  result.content
  result.err
  result.timeSeconds
  result.timeHuman
```

- result.timeSeconds & result.timeHuman: time for read file
- result.executor: executor condition with file path
- result.err: if exist, this field contains error
- result.content: content of readed file


## Default assertion

```yaml
result.err ShouldNotExist
```
