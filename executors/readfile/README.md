# Venom - Executor Read file

Step used to read a file

Use case: your application writes a file on disk. Venom can check that this file is produced, read it,
and return its content. Content can be used by another steps of testsuite.

Path can contain a fullpath, a wildcard or a directory:

```
- path: /a/full/path/file.txt
- path: afile.txt
- path: *.yml
- path: a_directory/
- path: ./foo/b*/**/z*.txt
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
    - result.err ShouldBeEmpty
```

## Output

```yaml
  result.timeseconds
  result.content
  result.contentjson
  result.size.filename
  result.md5sum.filename
  result.modtime.filename
  result.mod.filename
```

- result.timeseconds: execution duration
- result.err: if the file does not exist, this field contains an error
- result.content: content of the read file
- result.contentjson: content of the read file if it's a json file. You can access json data as result.contentjson.yourkey for example
- result.size.filename: size of the file 'filename'
- result.md5sum.filename: md5 of the file 'fifename'
- result.modtime.filename: modification date of the file 'filename', example: 1487698253
- result.mod.filename: rights on file 'filename', example: -rw-r--r--

Note: the value of 'filename' is equal to the path, where all letters are in lower case and where all '/' have been replaced by '_'.

## Default assertion

```yaml
result.err ShouldBeEmpty
```

## Example

testa.txt file:

```
simple content
multilines
```

testa.json file:

```json
{
  "foo": "bar"
}
```

testb.json file:

```json
[
  {
    "foo": "bar",
    "foo2": "bar2"
  }
]

```

venom test file:

```yaml
name: TestSuite Read File
testcases:
- name: TestCase Read File
  steps:
  - type: readfile
    path: testa.json
    assertions:
      - result.contentjson.foo ShouldEqual bar

  - type: readfile
    path: testb.json
    assertions:
      - result.contentjson.contentjson0.foo2 ShouldEqual bar2

  - type: readfile
    path: test.txt
    assertions:
      - result.content ShouldContainSubstring multilines
```
