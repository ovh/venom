# 🐍 Venom

Venom is a CLI (Command Line Interface) that aim to create, manage and run your integration tests with efficiency.

<a href="https://github.com/ovh/venom/releases/latest"><img alt="GitHub release" src="https://img.shields.io/github/v/release/ovh/venom.svg?logo=github&style=flat-square"></a>
[![GoDoc](https://godoc.org/github.com/ovh/venom?status.svg)](https://godoc.org/github.com/ovh/venom)
[![Go Report Card](https://goreportcard.com/badge/github.com/ovh/venom)](https://goreportcard.com/report/github.com/ovh/venom)
[![Discussions](https://img.shields.io/badge/Discussions-OVHcloud-brightgreen)](https://github.com/ovh/venom/discussions)
<a href="https://gitpod.io/#https://github.com/ovh/venom"><img src="https://img.shields.io/badge/Contribute%20with-Gitpod-908a85?logo=gitpod" alt="Contribute with Gitpod"/></a>
 
# Table of content

* [Overview](#overview)
* [Installing](#installing)
* [Updating](#updating)
* [Docker image](#docker-image)
* [CLI usage](#cli-usage)
  * [Globstar support](#globstar-support)
  * [Variables](#variables)
    * [Variable Definitions Files](#variable-definitions-files)
    * [Environment Variables](#environment-variables)
  * [Arguments](#arguments)
    * [Define arguments with environment variables](#define-arguments-with-environment-variables)
    * [Use a configuration file](#use-a-configuration-file)
* [Concepts](#concepts)
  * [TestSuites](#testsuites)
  * [Executors](#executors)
    * [User defined executors](#user-defined-executors)
  * [Variables](#variables)
    * [Testsuite variables](#testsuite-variables)
    * [Variable helpers](#variable-helpers)
  * [Use outputs from a test step as input of another test step](#use-outputs-from-a-test-step-as-input-of-another-test-step)
  * [Builtin venom variables](#builtin-venom-variables)
  * [Assertions](#assertions)
    * [Keywords](#keywords)
      * [`Must` Keywords](#must-keywords)
    * [Using logical operators](#using-logical-operators)
* [Write and run your first test suite](#write-and-run-your-first-test-suite)
* [Export tests report](#export-tests-report)
* [Advanced usage](#advanced-usage)
  * [Debug your testsuites](#debug-your-testsuites)
  * [Skip testcase and teststeps](#skip-testcase-and-teststeps)
  * [Iterating over data](#iterating-over-data)
* [FAQ](#faq)
  * [Common errors with quotes](#common-errors-with-quotes)
* [Use venom in CI/CD pipelines](#use-venom-in-cicd-pipelines)
* [Hacking](#hacking)
* [Contributing](#contributing)
* [License](#license)

# Overview

Venom allows you to handle integration tests the same way you code your application.
With Venom, testcases will be managed as code: the readability of the tests means that the tests are part of the code reviews. Thanks to that, write and execute testsuites become easier for developers and teams.

Concretely, you have to write testsuite in a YAML file.
Venom run executors (scripts, HTTP Request, web, IMAP, etc.) and apply assertions. 
It can also generate xUnit result files.

<img src="./venom.gif" alt="Venom Demonstration">

# Installing

## Install from binaries

You can find latest binary release from: https://github.com/ovh/venom/releases/latest/.

Example for Linux:

```bash
$ curl https://github.com/ovh/venom/releases/download/v1.0.1/venom.linux-amd64 -L -o /usr/local/bin/venom && chmod +x /usr/local/bin/venom
$ venom -h
```

# Updating

You can update to the latest version with `venom update` command:

```bash
$ venom update
```

The `venom update` command will download the latest version and replace the current binary:

```bash
Url to update venom: https://github.com/ovh/venom/releases/download/v1.0.1/venom.darwin-amd64
Getting latest release from: https://github.com/ovh/venom/releases/download/v1.0.1/venom.darwin-amd64 ...
Update done.
```

Check the new version with `venom version` command:

```bash
$ venom version
Version venom: v1.0.1 
```

# Docker image

Instead of installing (and updating) Venom locally, Venom can be started as a Docker image with following commands. 

Considering your testsuites are in `./tests` directory in your current directory and your test library is under `./tests/lib`, the results will be available under the `results` directory.

```bash
$ mkdir -p results
$ docker run --mount type=bind,source=$(pwd)/tests,target=/workdir/tests --mount type=bind,source=$(pwd)/results,target=/workdir/results ovhcom/venom:latest 
```

Please refer to https://hub.docker.com/r/ovhcom/venom/tags to get the available image tags.

# CLI Usage

`venom` CLI is composed of several commands:

```bash
$ venom -h
Venom - RUN Integration Tests

Usage:
  venom [command]

Available Commands:
  help        Help about any command
  run         Run Tests
  update      Update venom to the latest release version: venom update
  version     Display Version of venom: venom version

Flags:
  -h, --help   help for venom

Use "venom [command] --help" for more information about a command.
```

You can see the help of a command with `venom [command] -h`:

```bash
$ venom run -h

run integration tests

Usage:
  venom run [flags]

Examples:
  Run all testsuites containing in files ending with *.yml or *.yaml: venom run
  Run a single testsuite: venom run mytestfile.yml
  Run a single testsuite and export the result in JSON format in test/ folder: venom run mytestfile.yml --format=json --output-dir=test
  Run a single testsuite and export the result in XML and HTML formats in test/ folder: venom run mytestfile.yml --format=xml --output-dir=test --html-report
  Run a single testsuite and specify a variable: venom run mytestfile.yml --var="foo=bar"
  Run a single testsuite and load all variables from a file: venom run mytestfile.yml --var-from-file variables.yaml
  Run all testsuites containing in files ending with *.yml or *.yaml with verbosity: VENOM_VERBOSE=2 venom run
  
  Notice that variables initialized with -var-from-file argument can be overrided with -var argument
  
  More info: https://github.com/ovh/venom

Flags:
      --format string           --format:json, tap, xml, yaml (default "xml")
  -h, --help                    help for run
      --html-report             Generate HTML Report
      --lib-dir string          Lib Directory: can contain user executors. example:/etc/venom/lib:$HOME/venom.d/lib
      --output-dir string       Output Directory: create tests results file inside this directory
      --stop-on-failure         Stop running Test Suite on first Test Case failure
      --var stringArray         --var cds='cds -f config.json' --var cds2='cds -f config.json'
      --var-from-file strings   --var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionary
  -v, --verbose count           verbose. -vv to very verbose and -vvv to very verbose with CPU Profiling
```

## Run test suites in a specific order

- `venom run 01_foo.yml 02_foo.yml` will run 01 before 02. 
- `venom run 02_foo.yml 01_foo.yml` will run 02 before 01.

If you want to sort many testsuite files, you can use standard commands, example:

```bash
venom run `find . -type f -name "*.yml"|sort`
```

## Globstar support

The `venom` CLI supports globstar:

```
$ venom run ./foo/b*/**/z*.yml
```

## Variables

To specify individual variables on the command line, use the `--var` option when running the `venom run` commands:

```bash
$ venom run --var="foo=bar"
$ venom run --var='foo_list=["biz","buz"]'
$ venom run --var='foo={"biz":"bar","biz":"barr"}'
```

The `--var` option can be used many times in a single command.

### Variable Definitions Files

To set a lot of variables, it is more convenient to specify their values in a variable definitions file. This file is a YAML dictionary. You have to specify that file on the command line with `--var-from-file`:

```bash
venom run --var-from-file variables.yaml
```

### Environment Variables

As a fallback for the other ways of defining variables, `venom` tool searches the environment of its own process for environment variables named `VENOM_VAR_` followed by the name of a declared variable.

```bash
$ export VENOM_VAR_foo=bar
$ venom run *.yml
```

You can also define the environment variable and run your testsuite in one line:

```bash
$ VENOM_VAR_foo=bar venom run *.yml
```

## Arguments

You can define arguments on the command line using the flag name.

Flags are listed in the result of help command.

List of available flags for `venom run` command:

```
Flags:
      --format string           --format:json, tap, xml, yaml (default "xml")
  -h, --help                    help for run
      --html-report             Generate HTML Report
      --lib-dir string          Lib Directory: can contain user executors. example:/etc/venom/lib:$HOME/venom.d/lib
      --output-dir string       Output Directory: create tests results file inside this directory
      --stop-on-failure         Stop running Test Suite on first Test Case failure
      --var stringArray         --var cds='cds -f config.json' --var cds2='cds -f config.json'
      --var-from-file strings   --var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionary
  -v, --verbose count           verbose. -vv to very verbose and -vvv to very verbose with CPU Profiling
```

### Define arguments with environment variables

You can also define the arguments with environment variables:

```bash
# is the same as
VENOM_FORMAT=json venom run my-test-suite.yml

# is equivalent to
venom run my-test-suite.yml --format=json
```

Flags and their equivalent with environment variables usage:

- `--format="json"` flag is equivalent to `VENOM_FORMAT="json"` environment variable
- `--lib-dir="/etc/venom/lib:$HOME/venom.d/lib"` flag is equivalent to `VENOM_LIB_DIR="/etc/venom/lib"` environment variable
- `--output-dir="test-results"` flag is equivalent to `VENOM_OUTPUT_DIR="test-results"` environment variable
- `--stop-on-failure` flag is equivalent to `VENOM_STOP_ON_FAILURE=true` environment variable
- `--var foo=bar` flag is equivalent to `VENOM_VAR_foo='bar'` environment variable
- `--var-from-file fileA.yml fileB.yml` flag is equivalent to `VENOM_VAR_FROM_FILE="fileA.yml fileB.yml"` environment variable
- `-v` flag is equivalent to `VENOM_VERBOSE=1` environment variable
- `-vv` flag is equivalent to `VENOM_VERBOSE=2` environment variable

It is possible to set `NO_COLOR=1` environment variable to disable colors from output.

## Use a configuration file

You can define the Venom settings using a configuration file `.venomrc`. This configuration file should be placed in the current directory or in the `home` directory.

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

Please note that the command line flags overrides the configuration file. The configuration file overrides the environment variables.


# Concepts

## TestSuites

A test suite is a collection of test cases that are intended to be used to test a software program to show that it has a specified set of behaviors.
A test case is a specification of the inputs, execution conditions, testing procedure, and expected results that define a single test to be executed to achieve a particular software testing objective, such as to exercise a particular program path or to verify compliance with a specific requirement.

In `venom` the testcases are executed sequentially within a testsuite. Each testcase is an ordered set of steps. Each step is based on an `executor` that enable some specific kind of behavior.

In `venom` a testsuite is written in one `YAML` file respecting the following structure:

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
    retry_if: # (optional, lets you early break unrecoverable errors)
    - result.statuscode ShouldNotEqual 403
    delay: 2
    assertions:
    - result.statuscode ShouldEqual 200

```

## Executors

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

### User defined executors

You can define an executor with a single YAML file. This is a good way to abstract technical or functional behaviors and reuse them in complex testsuites.

Example:

file `lib/customA.yml`:

```yml
executor: hello
input:
  myarg: {}
steps:
- script: echo "{\"hello\":\"{{.input.myarg}}\"}"
  assertions:
  - result.code ShouldEqual 0
  vars:
    hello:
      from: result.systemoutjson.hello
    all:
      from: result.systemoutjson
output:
  display:
    hello: "{{.hello}}"
  all: "{{.all}}"
```

file `testsuite.yml`:

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

The user defined executors work with templating, you can check the templating result in `venom.log`. In this file, if you see an error as `error converting YAML to JSON: yaml: line 14: found unexpected end of stream`, you probably need to adjust indentation with the templating function `indent`. 

Example:

```yml
name: testsuite with a user executor multilines
testcases:
- name: test
  steps:
  - type: multilines
    script: |
      # test multilines
      echo "5"
    assertions:
    - result.alljson ShouldEqual 5
```

using this executor:

```yml
executor: multilines
input:
  script: "echo 'foo'"
steps:
- type: exec
  script: {{ .input.script | nindent 4 }}
  assertions:
  - result.code ShouldEqual 0
  vars:
    all:
      from: result.systemoutjson
output:
  all: '{{.all}}'
```


```bash
# lib/*.yml files will be loaded as executors.
$ venom run testsuite.yml 

# executors will be loaded from /etc/venom/lib, $HOME/venom.d/lib and lib/ directory relative to testsuite.yml file.
$ venom run --lib-dir=/etc/venom/lib:$HOME/venom.d/lib testsuite.yml 
```

## Variables

### Testsuite variables

You can define variables at the `testsuite` level.

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

More examples are available [here](https://github.com/ovh/venom/tree/master/variable_helpers.md)

## Use outputs from a test step as input of another test step

To be able to reuse a property from a teststep in a following testcase or step, you have to extract the variable, as the following example. 

After the first step execution, `venom` extracts a value using a regular expression `foo with a ([a-z]+) here` from the content of the `result.systemout` property returned by the `executor`.
Then this variable can be reused in another test, with the name `testA.myvariable` with `testA` corresponding to the name of the testcase. A default value could also be supplied if the variable can't be extracted from the output, which can commonly happen when parsing json output.

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
        default: "somevalue"

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

* {{.venom.datetime}}
* {{.venom.executable}}
* {{.venom.libdir}}
* {{.venom.outputdir}}
* {{.venom.testcase}}
* {{.venom.teststep.number}}
* {{.venom.testsuite.name}}
* {{.venom.testsuite.filename}}
* {{.venom.testsuite.filepath}}
* {{.venom.testsuite.shortName}}
* {{.venom.testsuite.workdir}}
* {{.venom.testsuite}}
* {{.venom.timestamp}}


## Assertions

### Keywords

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
* ShouldTimeEqual - [example](https://github.com/ovh/venom/tree/master/tests/assertions/ShouldTimeEqual.yml)

#### `Must` keywords

All the above assertions keywords also have a `Must` counterpart which can be used to create a required passing assertion and prevent test cases (and custom executors) to run remaining steps.

Example:
```yml
- steps:
  - type: exec
    script: exit 1
    assertions:
      - result.code MustEqual 0
  # Remaining steps in this context will not be executed
```

### Using logical operators

While assertions use `and` operator implicitly, it is possible to use other logical operators to perform complex assertions.

Supported operators are `and`, `or` and `xor`.

```yml
- name: Assertions operators
  steps:
  - script: echo 1
    assertions:
      - or:
        - result.systemoutjson ShouldEqual 1 
        - result.systemoutjson ShouldEqual 2
      # Nested operators
      - or:
        - result.systemoutjson ShouldBeGreaterThanOrEqualTo 1
        - result.systemoutjson ShouldBeLessThanOrEqualTo 1
        - or:
          - result.systemoutjson ShouldEqual 1
```

More examples are available in [`tests/assertions_operators.yml`](/tests/assertions_operators.yml).

# Write and run your first test suite 

To understand how Venom is working, let's create and run a first testsuite together.

The first assertions that we will do, in this testsuite, are to check whether the site we want to test (a public REST API, OVHcloud API for example):
- is accessible (respond with a 200 status code)
- responds in less than 5 seconds
- returns a valid response (which is in JSON format)

First, create your testsuite in a file called `testsuite.yml` for example.
Open it in your favorite editor or IDE and fill it with this content:

```yaml
name: APIIntegrationTest

vars:
  url: https://eu.api.ovh.com/

testcases:
- name: GET http testcase, with 5 seconds timeout
  steps:
  - type: http
    method: GET
    url: {{.url}}/1.0/
    timeout: 5
    assertions:
    - result.statuscode ShouldEqual 200
    - result.timeseconds ShouldBeLessThan 1
    - result.bodyjson ShouldContainKey apis
    - result.body ShouldContainSubstring /dedicated/server
    - result.body ShouldContainSubstring /ipLoadbalancing
```

Then, run your testsuite with the following command:

```bash
$ venom run 

 • APIIntegrationTest (testsuite.yml)
 	• GET-http-testcase-with-5-seconds-timeout SUCCESS
```

You wrote and executed your first testsuite with the HTTP executor! :)

# Export tests report

You can export your testsuite results as a report in several available formats: xUnit (XML), JSON, YAML, TAP.

You can specify the output directory with the `--output-dir` flag and the format with the `--format` flag (XML by default):

```bash
$ venom run --format=xml --output-dir="."

# html export
$ venom run --output-dir="." --html-report
```

Reports exported in XML can be visualized with a xUnit/jUnit Viewer, directly in your favorite CI/CD stack for example in order to see results run after run.

# Advanced usage

## Debug your testsuites

A *venom.log* file is generated for each `venom run` command.

There are two ways to debug a testsuite:
 - use `-v` flag on venom binary.
   - `$ venom run -v test.yml` will output details for each step
   - `$ venom run -vv test.yml` will generate *dump.json* files for each teststep.
   - `$ venom run -vvv test.yml` will generate *pprof* files for CPU profiling.
 - use `info` keyword in your teststep:
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

## Skip testcase and teststeps

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

A `skip` statement may also be placed at steps level to partially execute a testcase.
If all steps from a testcase are skipped, the testcase itself will also be treated as "skipped" rather than "passed"/"failed".

```yaml
name: "Skip testsuite"
vars:
  foo: bar

testcases:
- name: skip-one-of-these
  steps:
  - name: do-not-skip-this
    type: exec
    script: exit 0
    assertions:
    - result.code ShouldEqual 0
    skip:
    - foo ShouldNotBeEmpty
  - name: skip-this
    type: exec
    script: exit 1
    assertions:
    - result.code ShouldEqual 0
    skip:
    - foo ShouldBeEmpty

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
- name: range with hardcoded array
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

this is probably because you try to use a json value instead of a string. You should have more details in `venom.log` file.

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

# Use venom in CI/CD pipelines

Venom can be used in dev environment or on your CI server.
To display properly the venom output, you probably will have to export the environment variable `IS_TTY=true` before running venom.

# Hacking

[How to write your own executor?](https://github.com/ovh/venom/tree/master/executors#venom-executor)

## How to compile?
```bash
$ make build
```

## How to test?

### Unit tests:

```bash
make test
```

### Integration tests:

Prepare the stack:

```bash
make build OS=linux ARCH=amd64
cp dist/venom.linux-amd64 tests/venom
cd tests
make start-test-stack  # (wait a few seconds)
make build-test-binary-docker
```

Run integration tests:

```bash
make run-test
```

Cleanup:

```bash
make clean
make stop-test-stack
```

# Contributing

<a href="https://gitpod.io/#https://github.com/ovh/venom"><img src="https://img.shields.io/badge/Contribute%20with-Gitpod-908a85?logo=gitpod" alt="Contribute with Gitpod"/></a>

Please read the [contributing guide](./CONTRIBUTING.md) to learn about how you can contribute to Venom ;-).
There is no small contribution, don't hesitate!

Our awesome contributors:

<a href="https://github.com/ovh/venom/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=ovh/venom" />
</a>

# License

Copyright 2022 OVH SAS

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
