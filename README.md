# 🐍 Venom

Venom run executors (script, HTTP Request, etc. ) and assertions.
It can also output xUnit results files.

<img src="./venom.gif" alt="Venom Demonstration" width="80%">

## Command Line

Download latest binary release from https://github.com/ovh/venom/releases  
or just go get it ```go get -u github.com/ovh/venom/cli/venom```


```bash
$ venom run -h
Run Tests

Usage:
  venom run [flags]

Flags:
      --format string           --format:yaml, json, xml, tap (default "xml")
  -h, --help                    help for run
      --log string              Log Level : debug, info, warn or disable (default "warn")
      --no-check-variables      Don't check variables before run
      --output-dir string       Output Directory: create tests results file inside this directory
      --parallel int            --parallel=2 : launches 2 Test Suites in parallel (default 1)
      --profiling               Enable Mem / CPU Profile with pprof
      --stop-on-failure         Stop running Test Suite on first Test Case failure
      --strict                  Exit with an error code if one test fails
      --var strings             --var cds='cds -f config.json' --var cds2='cds -f config.json'
      --var-from-file strings   --var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionnary'
```

## TestSuites

A test suite is a collection of test cases that are intended to be used to test a software program to show that it has some specified set of behaviours. 
A test case is a specification of the inputs, execution conditions, testing procedure, and expected results that define a single test to be executed to achieve a particular software testing objective, such as to exercise a particular program path or to verify compliance with a specific requirement.

In `venom` the testcases are executed sequentialy within a testsuite. Each testcase is an ordered set of steps. Each step is based on an `executor` that enable some specific kind of behavior.

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

## Executors

* **dbfixtures**: https://github.com/ovh/venom/tree/master/executors/dbfixtures
* **exec**: https://github.com/ovh/venom/tree/master/executors/exec `exec` is the default type for a step
* **http**: https://github.com/ovh/venom/tree/master/executors/http
* **imap**: https://github.com/ovh/venom/tree/master/executors/imap
* **kafka** https://github.com/ovh/venom/tree/master/executors/kafka
* **ovhapi**: https://github.com/ovh/venom/tree/master/executors/ovhapi
* **readfile**: https://github.com/ovh/venom/tree/master/executors/readfile
* **redis**: https://github.com/ovh/venom/tree/master/executors/redis
* **smtp**: https://github.com/ovh/venom/tree/master/executors/smtp
* **ssh**: https://github.com/ovh/venom/tree/master/executors/ssh
* **web**: https://github.com/ovh/venom/tree/master/executors/web
* **grpc**: https://github.com/ovh/venom/tree/master/executors/grpc
* **rabbitmq**: https://github.com/ovh/venom/tree/master/executors/rabbitmq
* **sql**: https://github.com/ovh/venom/tree/master/executors/sql


## Variables

### Testsuite variables

You can define variable on the `testsuite` level.

```yaml
name: myTestSuite
vars:
  foo: foo
  biz:
    bar: bar

testcases:
- name: first-test-case
  steps:
  - type: exec
    script: echo '{{.foo}} {{.biz.bar}}'
    assertions:
    - result.code ShouldEqual 0
    - result.systemout ShouldEqual "foo bar"
...
```

Each user variable used in testsuite must be declared in this section. You can override the values at runtime in a number of ways:
- Individually, with the `--var` command line option.
- In variable definitions files, either specified on the command line `--var-from-file`.
- As environment variables.


#### Variable on Command Line

To specify individual variables on the command line, use the `--var` option when running the `venom run` commands:

```
venom run --var="foo=bar"
venom run --var='foo_list=["biz","buz"]'
venom run --var='foo={"biz":"bar","biz":"barr"}'
```

The -var option can be used any number of times in a single command.

#### Variable Definitions Files

To set lots of variables, it is more convenient to specify their values in a variable definitions file. This file is a yaml dictionnay and you have specify that file on the command line with `--var-from-file`

#### Environment Variables

As a fallback for the other ways of defining variables, `venom` searches the environment of its own process for environment variables named VENOM_VAR_ followed by the name of a declared variable.

```bash
$ export VENOM_VAR_foo=bar
$ venom run *.yml
```

### How to use outputs from a test step as input of another test step

To be able to reuse a property from a teststep in a following testcase or step, you have to extract the variable, as the following example. 

After the first step execution, `venom` extracts a value using a regular expression `foo with a ([a-z]+) here` from the content of the `result.systemout` property returned by the `executor`.
Then we are able to reuse this variable with the name `testA.myvariable` with `testA` corresponding to the name of the testcase.

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

### Builtin venom variables

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
* {{.venom.testcase}}
* {{.venom.teststep.number}}
* {{.venom.datetime}}
* {{.venom.timestamp}}

## Export test results as jUnit, json, yaml or tap reports

```bash
venom run --format=xml --output-dir="."
```

## Assertion

### Keywords

* ShouldEqual
* ShouldNotEqual
* ShouldAlmostEqual
* ShouldNotAlmostEqual
* ShouldResemble
* ShouldNotResemble
* ShouldPointTo
* ShouldNotPointTo
* ShouldBeNil
* ShouldNotBeNil
* ShouldBeTrue
* ShouldBeFalse
* ShouldBeZeroValue
* ShouldBeGreaterThan
* ShouldBeGreaterThanOrEqualTo
* ShouldBeLessThan
* ShouldBeLessThanOrEqualTo
* ShouldBeBetween
* ShouldNotBeBetween
* ShouldBeBetweenOrEqual
* ShouldNotBeBetweenOrEqual
* ShouldContain
* ShouldNotContain
* ShouldContainKey
* ShouldNotContainKey
* ShouldBeIn
* ShouldNotBeIn
* ShouldBeEmpty
* ShouldNotBeEmpty
* ShouldHaveLength
* ShouldStartWith
* ShouldNotStartWith
* ShouldEndWith
* ShouldNotEndWith
* ShouldBeBlank
* ShouldNotBeBlank
* ShouldContainSubstring
* ShouldNotContainSubstring
* ShouldEqualWithout
* ShouldEqualTrimSpace
* ShouldHappenBefore
* ShouldHappenOnOrBefore
* ShouldHappenAfter
* ShouldHappenOnOrAfter
* ShouldHappenBetween
* ShouldHappenOnOrBetween
* ShouldNotHappenOnOrBetween
* ShouldHappenWithin
* ShouldNotHappenWithin
* ShouldBeChronological
* ShouldNotExist

Most assertion keywords documentation can be found on https://pkg.go.dev/github.com/ovh/venom/assertions.

### Write your executor

An executor have to implement this interface

```go

// Executor execute a testStep.
type Executor interface {
	// Run run a Test Step
	Run(ctx context.Content, TestStep) (interface{}, error)
}
```

Example

```go


// Name of executor
const Name = "myexecutor"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor struct
type Executor struct {
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
}

// Result represents a step result
type Result struct {
	Code        int    `json:"code,omitempty" yaml:"code,omitempty"`
	Command     string `json:"command,omitempty" yaml:"command,omitempty"`
	Systemout   string   `json:"systemout,omitempty" yaml:"systemout,omitempty"` // put in testcase.Systemout by venom if present
	Systemerr   string   `json:"systemerr,omitempty" yaml:"systemerr,omitempty"` // put in testcase.Systemerr by venom if present
}

// GetDefaultAssertions return default assertions for this executor
// Optional
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

// Run execute TestStep
func (Executor)	Run(context.Context, TestStep, string) (interface{}, error) {
	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	// to something with e.Command here...
	//...

	systemout := "foo"
	ouputCode := 0

	// prepare result
	r := Result{
		Code:    ouputCode, // return Output Code
		Command: e.Command, // return Command executed
		Systemout: systemout, // return Output string
	}

	return r
}

```

Feel free to open a Pull Request with your executors.


# Hacking

You've developed a new cool feature? Fixed an annoying bug? We'd be happy
to hear from you! Make sure to read [CONTRIBUTING.md](./CONTRIBUTING.md) before.

# License

This work is under the BSD license, see the [LICENSE](LICENSE) file for details.
