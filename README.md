# 🐍 Venom

Venom execute "executors" (script, HTTP Request, etc. ) and assertions.
It can also generate xUnit result files.

<img src="./venom.gif" alt="Venom Demonstration">

* [Command Line](#command-line)
* [Docker image](#docker-image)
* [TestSuites](#testsuites)
* [Executors](#executors)
  * [User defined executors](#user-defined-executors)
* [Variables](#variables)
  * [Testsuite variables](#testsuite-variables)
    * [Variable on Command Line](#variable-on-command-line)
    * [Variable Definitions Files](#variable-definitions-files)
    * [Environment Variables](#environment-variables)
    * [Variable helpers](#variable-helpers)
  * [How to use outputs from a test step as input of another test step](#how-to-use-outputs-from-a-test-step-as-input-of-another-test-step)
  * [Builtin venom variables](#builtin-venom-variables)
* [Tests Report](#tests-report)
* [Assertion](#assertion)
  * [Keywords](#keywords)
* [Advanced usage](#advanced-usage)
  * [Debug your testsuites](#debug-your-testsuites)
  * [Skip testcase](#skip-testcase)
  * [Iterating over data](#iterating-over-data)
* [Use venom in CI](#use-venom-in-ci)
* [Hacking](#hacking)
* [License](#license)

# Command Line

Download latest binary release from https://github.com/ovh/venom/releases

```bash
$ venom run -h

$ venom run *.yml

Notice that variables initialized with -var-from-file argument can be overrided with -var argument.

Usage:
  venom run [flags]

Flags:
      --format string           --format:yaml, json, xml, tap (default "xml")
  -h, --help                    help for run
      --lib-dir string          Lib Directory: this directory can contain user executors. This overrides the default lib folder directory
      --output-dir string       Output Directory: create tests results file inside this directory
      --stop-on-failure         Stop running Test Suite on first Test Case failure
      --var stringArray         --var cds='cds -f config.json' --var cds2='cds -f config.json'
      --var-from-file strings   --var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionnary
  -v, --verbose count           verbose. -vv to very verbose and -vvv to very verbose with CPU Profiling
```

Globstar support: `venom run ./foo/b*/**/z*.yml`

You can define the arguments with environment variables:

```bash
venom run my-test-suite.yml --format=json
# is the same as
VENOM_FORMAT=json venom run my-test-suite.yml
```

```
      --format           -  example: VENOM_FORMAT=json
      --output-dir       -  example: VENOM_OUTPUT_DIR=.
      --lib-dir          -  example: VENOM_LIB_DIR=/etc/venom/lib:$HOME/venom.d/lib
      --stop-on-failure  -  example: VENOM_STOP_ON_FAILURE=true
      --var              -  example: VENOM_VAR="foo=bar"
      --var-from-file    -  example: VENOM_VAR_FROM_FILE="fileA.yml fileB.yml"
      -v                 -  example: VENOM_VERBOSE=2 is the same as -vv
```

You can define the venom settings using a configuration file `.venomrc`. This configuration file should be placed in the current directory or in the home directory.

```yml
variables: 
  - foo=bar
variables_files:
  - my_var_file.yaml
stop_on_failure: true
format: xml
output_dir: output
lib_dir: lib
verbosity: 3
```

Please note that command line flags overrides the configuration file. Configuration file overrides the environment variables.

# Docker image

Venom can be started inside a docker image with:
```bash
$ git clone git@github.com:ovh/venom.git
$ cd venom
$ docker run -it $(docker build -q .) --rm -v $(pwd)/outputs:/outputs -v $(pwd):/tests run /tests/testsuite.yaml
```

# TestSuites

A test suite is a collection of test cases that are intended to be used to test a software program to show that it has a specified set of behaviors.
A test case is a specification of the inputs, execution conditions, testing procedure, and expected results that define a single test to be executed to achieve a particular software testing objective, such as to exercise a particular program path or to verify compliance with a specific requirement.

In `venom` the testcases are executed sequentially within a testsuite. Each testcase is an ordered set of steps. Each step is based on an `executor` that enable some specific kind of behavior.

In `venom` a testsuite is written in one `yaml` file respecting the following structure.

```yaml

name: Title of TestSuite
testcases:
- name: TestCase with default value, exec cmd. Check if exit code != 1
  steps:
  - script: echo 'foo'
    type: exec

- name: Title of First TestCase
  steps:
  - script: echo 'foo'
    assertions:
    - result.code ShouldEqual 0
  - script: echo 'bar'
    assertions:
    - result.systemout ShouldNotContainSubstring foo
    - result.timeseconds ShouldBeLessThan 1

- name: GET http testcase, with 5 seconds timeout
  steps:
  - type: http
    method: GET
    url: https://eu.api.ovh.com/1.0/
    timeout: 5
    assertions:
    - result.body ShouldContainSubstring /dedicated/server
    - result.body ShouldContainSubstring /ipLoadbalancing
    - result.statuscode ShouldEqual 200
    - result.timeseconds ShouldBeLessThan 1

- name: Test with retries and delay in seconds between each try
  steps:
  - type: http
    method: GET
    url: https://eu.api.ovh.com/1.0/
    retry: 3
    delay: 2
    assertions:
    - result.statuscode ShouldEqual 200

```

# Executors

* **amqp**: https://github.com/ovh/venom/tree/master/executors/amqp
* **dbfixtures**: https://github.com/ovh/venom/tree/master/executors/dbfixtures
* **exec**: https://github.com/ovh/venom/tree/master/executors/exec `exec` is the default type for a step
* **grpc**: https://github.com/ovh/venom/tree/master/executors/grpc
* **http**: https://github.com/ovh/venom/tree/master/executors/http
* **imap**: https://github.com/ovh/venom/tree/master/executors/imap
* **kafka** https://github.com/ovh/venom/tree/master/executors/kafka
* **mqtt** https://github.com/ovh/venom/tree/master/executors/mqtt
* **odbc**: https://github.com/ovh/venom/tree/master/executors/plugins/odbc
* **ovhapi**: https://github.com/ovh/venom/tree/master/executors/ovhapi
* **rabbitmq**: https://github.com/ovh/venom/tree/master/executors/rabbitmq
* **readfile**: https://github.com/ovh/venom/tree/master/executors/readfile
* **redis**: https://github.com/ovh/venom/tree/master/executors/redis
* **smtp**: https://github.com/ovh/venom/tree/master/executors/smtp
* **sql**: https://github.com/ovh/venom/tree/master/executors/sql
* **ssh**: https://github.com/ovh/venom/tree/master/executors/ssh
* **web**: https://github.com/ovh/venom/tree/master/executors/web

## User defined executors

You can define an executor with a single yaml file. This is a good way to abstract technical or functional behaviors and reuse them in complex testsuites.

Example:

file `lib/customA.yml`
```yml
executor: hello
input:
  myarg: {}
steps:
- script: echo "{\"hello\":\"{{.input.myarg}}\"}"
  assertions:
  - result.code ShouldEqual 0
output:
  display:
    hello: "{{.result.systemoutjson.hello}}"
  all: "{{.result.systemoutjson}}"
```

file `testsuite.yml`
```yml
name: testsuite with a user executor
testcases:
- name: testA
  steps:
  - type: hello
    myarg: World
    assertions:
    - result.display.hello ShouldContainSubstring World
    - result.alljson.hello ShouldContainSubstring World
```

Notice the variable `alljson`. All variables declared in output are automatically converted in a json format with the suffix `json`. In the example above, two implicit variables are available: `displayjson.hello` and `alljson`.

Venom will load user's executors from the directory `lib/` relative to the testsuite path. You add executors source path using the flag `--lib-dir`. 
Note that all folders listed with `--lib-dir` will be scanned recursively to find `.yml` files as user executors.

```bash
# lib/*.yml files will be loaded as executors.
$ venom run testsuite.yml 

# executors will be loaded from /etc/venom/lib, $HOME/venom.d/lib and lib/ directory relative to testsuite.yml file.
$ venom run --lib-dir=/etc/venom/lib:$HOME/venom.d/lib testsuite.yml 
```

# Variables

## Testsuite variables

You can define variable on the `testsuite` level.

```yaml
name: myTestSuite
vars:
  foo: foo
  biz:
    bar: bar
  aString: '{"foo": "bar"}'

testcases:
- name: first-test-case
  steps:
  - type: exec
    script: echo '{{.foo}} {{.biz.bar}}'
    assertions:
    - result.code ShouldEqual 0
    - result.systemout ShouldEqual "foo bar"

- name: foobar
  steps:
  - script: echo '{{.aString}}'
    info: value of aString is {{.aString}}
    assertions:
    - result.systemoutjson.foo ShouldEqual bar
...
```

Each user variable used in testsuite must be declared in this section. You can override its value at runtime in a number of ways:
- Individually, with the `--var` command line option.
- In variable definitions files, either specified on the command line `--var-from-file`.
- As environment variables.


### Variable on Command Line

To specify individual variables on the command line, use the `--var` option when running the `venom run` commands:

```
venom run --var="foo=bar"
venom run --var='foo_list=["biz","buz"]'
venom run --var='foo={"biz":"bar","biz":"barr"}'
```

The `--var` option can be used many times in a single command.

### Variable Definitions Files

To set lots of variables, it is more convenient to specify their values in a variable definitions file. This file is a YAML dictionnary and you have specify that file on the command line with `--var-from-file`

### Environment Variables

As a fallback for the other ways of defining variables, `venom` searches the environment of its own process for environment variables named `VENOM_VAR_` followed by the name of a declared variable.

```bash
$ export VENOM_VAR_foo=bar
$ venom run *.yml
```

### Variable helpers

Available helpers and some examples:

- `abbrev`
- `abbrevboth`
- `trunc`
- `trim`
- `upper`: {{.myvar | upper}}
- `lower`: {{.myvar | lower}}
- `title`
- `untitle`
- `substr`
- `repeat`
- `trimall`
- `trimAll`
- `trimSuffix`
- `trimPrefix`
- `nospace`
- `initials`
- `randAlphaNum`
- `randAlpha`
- `randASCII`
- `randNumeric`
- `swapcase`
- `shuffle`
- `snakecase`
- `camelcase`
- `quote`
- `squote`
- `indent`
- `nindent`
- `replace`: {{.myvar | replace "_" "."}}
- `plural`
- `default`: {{.myvar | default ""}}
- `empty`
- `coalesce`
- `toJSON`
- `toPrettyJSON`
- `b64enc`
- `b64dec` {{.result.bodyjson | b64enc}}
- `escape`: replace ‘_‘, ‘/’, ‘.’ by ‘-’


## How to use outputs from a test step as input of another test step

To be able to reuse a property from a teststep in a following testcase or step, you have to extract the variable, as the following example. 

After the first step execution, `venom` extracts a value using a regular expression `foo with a ([a-z]+) here` from the content of the `result.systemout` property returned by the `executor`.
Then it is able to reuse this variable with the name `testA.myvariable` with `testA` corresponding to the name of the testcase.

```yaml
name: MyTestSuite
testcases:
- name: testA
  steps:
  - type: exec
    script: echo 'foo with a bar here'
    vars:
      myvariable:
        from: result.systemout
        regex: foo with a ([a-z]+) here

- name: testB
  steps:
  - type: exec
    script: echo {{.testA.myvariable}}
    assertions:
    - result.code ShouldEqual 0
    - result.systemout ShouldContainSubstring bar
```

## Builtin venom variables

```yaml
name: MyTestSuite
testcases:
- name: testA
  steps:
  - type: exec
    script: echo '{{.venom.testsuite}} {{.venom.testsuite.filename}} {{.venom.testcase}} {{.venom.teststep.number}} {{.venom.datetime}} {{.venom.timestamp}}'
    # will display something as: MyTestSuite MyTestSuiteWithVenomBuiltinVar.yml testA 0 2018-08-05T21:38:24+02:00 1533497904

```

Builtin variables:

* {{.venom.testsuite}}
* {{.venom.testsuite.filename}}
* {{.venom.testsuite.shortName}}
* {{.venom.testsuite.workdir}}
* {{.venom.testcase}}
* {{.venom.teststep.number}}
* {{.venom.datetime}}
* {{.venom.timestamp}}

# Tests Report

```bash
venom run --format=xml --output-dir="."
```

Available formats: jUnit (xml), json, yaml, tap reports

# Assertion

## Keywords

* ShouldEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldEqual.yml)
* ShouldNotEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotEqual.yml)
* ShouldAlmostEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldAlmostEqual.yml)
* ShouldNotAlmostEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotAlmostEqual.yml)
* ShouldBeNil - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeNil.yml)
* ShouldNotBeNil - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotBeNil.yml)
* ShouldBeTrue - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeTrue.yml)
* ShouldBeFalse - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeFalse.yml)
* ShouldBeZeroValue - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeZeroValue.yml)
* ShouldBeGreaterThan - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeGreaterThan.yml)
* ShouldBeGreaterThanOrEqualTo - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeGreaterThanOrEqualTo.yml)
* ShouldBeLessThan - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeLessThan.yml)
* ShouldBeLessThanOrEqualTo - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeLessThanOrEqualTo.yml)
* ShouldBeBetween - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeBetween.yml)
* ShouldNotBeBetween - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotBeBetween.yml)
* ShouldBeBetweenOrEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeBetweenOrEqual.yml)
* ShouldNotBeBetweenOrEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotBeBetweenOrEqual.yml)
* ShouldContain - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldContain.yml)
* ShouldNotContain - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotContain.yml)
* ShouldContainKey - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldContainKey.yml)
* ShouldNotContainKey - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotContainKey.yml)
* ShouldBeIn - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeIn.yml)
* ShouldNotBeIn - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotBeIn.yml)
* ShouldBeEmpty - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeEmpty.yml)
* ShouldNotBeEmpty - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotBeEmpty.yml)
* ShouldHaveLength - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldHaveLength.yml)
* ShouldStartWith - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldStartWith.yml)
* ShouldNotStartWith - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotStartWith.yml)
* ShouldEndWith - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldEndWith.yml)
* ShouldNotEndWith - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotEndWith.yml)
* ShouldBeBlank - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldBeBlank.yml)
* ShouldNotBeBlank - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotBeBlank.yml)
* ShouldContainSubstring - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldContainSubstring.yml)
* ShouldNotContainSubstring - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotContainSubstring.yml)
* ShouldEqualTrimSpace - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldEqualTrimSpace.yml)
* ShouldNotExist - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldNotExist.yml)
* ShouldHappenBefore - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldHappenBefore.yml)
* ShouldHappenOnOrBefore - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldHappenOnOrBefore.yml)
* ShouldHappenAfter - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldHappenAfter.yml)
* ShouldHappenOnOrAfter - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldHappenOnOrAfter.yml)
* ShouldHappenBetween - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldHappenBetween.yml)

# Advanced usage
## Debug your testsuites

There is two ways to debug a testsuite:
 - use `-v` flag on venom binary.
   - `$ venom run -v test.yml` will output a venom.log file
   - `$ venom run -vv test.yml` will output a venom.log file and dump.json files for each teststep.
 - use `info` keyword your teststep:
`test.yml` file:
```yml
name: Exec testsuite
testcases:
- name: testA
  steps:
  - type: exec
    script: echo 'foo with a bar here'
    info:
      - this a first info
      - and a second...
- name: cat json
  steps:
  - script: cat exec/testa.json
    info: "the value of result.systemoutjson is {{.result.systemoutjson}}"
    assertions:
    - result.systemoutjson.foo ShouldContainSubstrin bar
```

```bash
$ venom run test.yml

# output:

 • Exec testsuite (exec.yml)
 	• testA SUCCESS
	  [info] this a first info (exec.yml:8)
	  [info] and a second... (exec.yml:9)
 	• testB SUCCESS
 	• sleep 1 SUCCESS
 	• cat json SUCCESS
	  [info] the value of result.systemoutjson is map[foo:bar] (exec.yml:34)
```

## Skip testcase

It is possible to skip `testcase` according to some `assertions`. For instance, the following example will skip the last testcase.

```yaml
name: "Skip testsuite"
vars:
  foo: bar

testcases:
- name: init
  steps:
  - type: exec
    script: echo {{.foo}}
    assertions:
    - result.code ShouldEqual 0
    - result.systemout ShouldContainSubstring bar

- name: do-not-skip-this
  skip: 
  - foo ShouldNotBeEmpty
  steps:
  - type: exec
    script: exit 0

- name: skip-this
  skip: 
    - foo ShouldBeEmpty
  steps:
  - type: exec
    script: command_not_found
    assertions:
    - result.code ShouldEqual 0

```

## Iterating over data

It is possible to iterate over data using `range` attribute.

The following data types are supported, each exposing contexted variables `.index`, `.key` and `.value`:

- An array where each value will be iterated over (`[]interface{}`)
  - `.index`/`.key`: current iteration index
  - `.value`: current iteration item value
- A map where each key will be iterated over (`map[string]interface{}`)
  - `.index`: current iteration index
  - `.key`: current iteration item key
  - `.value`: current iteration item value
- An integer to perform target step `n` times (`int`)
  - `.index`/`.key`/`.value`: current iteration index
- A templated string which results in one of the above typing (`string`)
  - It can be either inherited from vars file, or interpolated from a previous step result

For instance, the following example will iterate over an array of two items containing maps:
```yaml
- name: range with harcoded array
  steps:
  - type: exec
    range:
      - actual: hello
        expected: hello
      - actual: world
        expected: world
    script: echo "{{.value.actual}}"
    assertions:
    - result.code ShouldEqual 0
    - result.systemout ShouldEqual "{{.value.expected}}"
```

More examples are available in [`tests/ranged.yml`](/tests/ranged.yml).

# FAQ

## Common errors with quotes

If you have this kind of error:

```
err:unable to parse file "foo.yaml": error converting YAML to JSON: yaml: line 8: did not find expected key
```

this is probably because you try to use a json value instead of a string. You should have more details in venom.log file.

Wrong:

```yml
...
vars:
  body: >-
      {
        "the-attribute": "the-value"
      }
...
steps:
- type: http
  body: "{{.body}}"
...
```

OK:


```yml
...
vars:
  body: >-
      {
        "the-attribute": "the-value"
      }
...
steps:
- type: http
  body: '{{.body}}'
...
```

Note the simple quote on the value of `body`.


# Use venom in CI

Venom can be use on dev environement or your CI server.
To display correctly the venom output, you probably will have to export the environment variable `IS_TTY=true` before running venom.

# Hacking

[How to write your own executor?](https://github.com/ovh/venom/tree/master/executors#venom-executor)

How to compile?
```bash
$ make build
```

# License

Copyright 2021 OVH SAS

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
