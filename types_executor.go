package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gosimple/slug"
	"github.com/ovh/venom/interpolate"
	"github.com/pkg/errors"
	"github.com/rockbears/yaml"
)

// Executor execute a testStep.
type Executor interface {
	// Run run a Test Step
	Run(context.Context, TestStep) (interface{}, error)
}

type ExecutorRunner interface {
	Executor
	executorWithDefaultAssertions
	executorWithZeroValueResult
	ExecutorWithSetup
	Name() string
	Retry() int
	RetryIf() []string
	Delay() int
	Timeout() int
	Info() []string
	Type() string
	GetExecutor() Executor
}

var _ Executor = new(executor)

// ExecutorWrap contains an executor implementation and some attributes
type executor struct {
	Executor
	name    string
	retry   int      // nb retry a test case if it is in failure.
	retryIf []string // retry conditions to check before performing any retries
	delay   int      // delay between two retries
	timeout int      // timeout on executor
	info    []string // info to display after the run and before the assertion
	stype   string   // builtin, plugin, user
}

func (e executor) Name() string {
	return e.name
}

func (e executor) Type() string {
	return e.stype
}

func (e executor) Retry() int {
	return e.retry
}

func (e executor) RetryIf() []string {
	return e.retryIf
}

func (e executor) Delay() int {
	return e.delay
}

func (e executor) Timeout() int {
	return e.timeout
}

func (e executor) Info() []string {
	return e.info
}

func (e executor) GetExecutor() Executor {
	return e.Executor
}

func (e executor) GetDefaultAssertions() *StepAssertions {
	if e.Executor == nil {
		return nil
	}
	x, ok := e.Executor.(executorWithDefaultAssertions)
	if ok {
		return x.GetDefaultAssertions()
	}
	return nil
}

func (e executor) ZeroValueResult() interface{} {
	if e.Executor == nil {
		return nil
	}
	x, ok := e.Executor.(executorWithZeroValueResult)
	if ok {
		return x.ZeroValueResult()
	}
	return nil
}

func (e executor) Setup(ctx context.Context, vars H) (context.Context, error) {
	if e.Executor == nil {
		return ctx, nil
	}
	x, ok := e.Executor.(ExecutorWithSetup)
	if ok {
		return x.Setup(ctx, vars)
	}
	return ctx, nil
}

func (e executor) TearDown(ctx context.Context) error {
	if e.Executor == nil {
		return nil
	}
	x, ok := e.Executor.(ExecutorWithSetup)
	if ok {
		return x.TearDown(ctx)
	}
	return nil
}

func (e executor) Run(ctx context.Context, step TestStep) (interface{}, error) {
	if e.Executor == nil {
		return nil, nil
	}
	return e.Executor.Run(ctx, step)
}

func newExecutorRunner(e Executor, name, stype string, retry int, retryIf []string, delay, timeout int, info []string) ExecutorRunner {
	return &executor{
		Executor: e,
		name:     name,
		retry:    retry,
		retryIf:  retryIf,
		delay:    delay,
		timeout:  timeout,
		info:     info,
		stype:    stype,
	}
}

// executorWithDefaultAssertions execute a testStep.
type executorWithDefaultAssertions interface {
	// GetDefaultAssertion returns default assertions
	GetDefaultAssertions() *StepAssertions
}

type executorWithZeroValueResult interface {
	ZeroValueResult() interface{}
}

type ExecutorWithSetup interface {
	Setup(ctx context.Context, vars H) (context.Context, error)
	TearDown(ctx context.Context) error
}

func GetExecutorResult(r interface{}) map[string]interface{} {
	d, err := Dump(r)
	if err != nil {
		panic(err)
	}
	return d
}

type UserExecutor struct {
	Executor  string
	Input     H                 `json:"input" yaml:"input"`
	TestSteps []json.RawMessage `json:"steps" yaml:"steps"`
	Raw       []byte            `json:"-" yaml:"-"` // the raw file content of the executor
	RawInputs []byte            `json:"-" yaml:"-"`
	Filename  string            `json:"-" yaml:"-"`
	Output    json.RawMessage   `json:"output" yaml:"output"`
}

// Run is not implemented on user executor
func (ux UserExecutor) Run(ctx context.Context, step TestStep) (interface{}, error) {
	return nil, errors.New("Run not implemented for user interface, use RunUserExecutor instead")
}

func (ux UserExecutor) ZeroValueResult() interface{} {
	type Output struct {
		Result interface{} `json:"result"`
	}
	output := &Output{
		Result: ux.Output,
	}
	outputS, err := json.Marshal(output)
	if err != nil {
		return ""
	}

	result := make(map[string]interface{})
	err = JSONUnmarshal(outputS, &result)
	if err != nil {
		return ""
	}
	return result
}

// RunUserExecutor runs a user executor with the given context, runner, test case, test step result, and step.
func (v *Venom) RunUserExecutor(ctx context.Context, runner ExecutorRunner, tcIn *TestCase, tsIn *TestStepResult, step TestStep) (interface{}, error) {
	vrs := tcIn.TestSuiteVars.Clone()
	ux := runner.GetExecutor().(UserExecutor)
	var tsVars map[string]string
	newUX := UserExecutor{}
	var err error

	// process inputs
	if len(ux.RawInputs) != 0 {
		tsVars, err = DumpString(vrs)
		if err != nil {
			return nil, errors.Wrapf(err, "error processing executor inputs: unable to dump testsuite vars")
		}

		interpolatedInput, err := interpolate.Do(string(ux.RawInputs), tsVars)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to interpolate executor inputs %q", ux.Executor)
		}

		err = yaml.Unmarshal([]byte(interpolatedInput), &newUX)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal inputs for executor %q - raw interpolated:\n%v", ux.Executor, string(interpolatedInput))
		}

		for k, va := range newUX.Input {
			if strings.HasPrefix(k, "input.") {
				// do not reinject input.vars from parent user executor if exists
				continue
			} else if !strings.HasPrefix(k, "venom") {
				if vl, ok := step[k]; ok && vl != nil && vl != "" { // value from step and not nil/empty
					vrs.AddWithPrefix("input", k, vl)
				} else { // default value from executor
					vrs.AddWithPrefix("input", k, va)
				}
			} else {
				vrs.Add(k, va)
			}
		}
		tsVars, err = DumpString(vrs)
		if err != nil {
			return nil, errors.Wrapf(err, "error processing executor inputs: unable to dump testsuite vars")
		}
	}

	interpolatedFull, err := interpolate.Do(string(ux.Raw), tsVars)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to interpolate executor %q", ux.Executor)
	}
	// quote any remaining template expressions to ensure proper YAML parsing
	sanitized := quoteTemplateExpressions([]byte(interpolatedFull))

	err = yaml.Unmarshal([]byte(sanitized), &newUX)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal executor %q - raw interpolated :\n%v", ux.Executor, string(sanitized))
	}
	ux.Output = newUX.Output

	tc := &TestCase{
		TestCaseInput: TestCaseInput{
			Name:         ux.Executor,
			Vars:         vrs,
			RawTestSteps: newUX.TestSteps,
		},
		number:          tcIn.number,
		TestSuiteVars:   tcIn.TestSuiteVars,
		IsExecutor:      true,
		TestStepResults: make([]TestStepResult, 0),
	}

	tc.originalName = tc.Name
	tc.Name = slug.Make(tc.Name)
	tc.Vars.Add("venom.testcase", tc.Name)
	tc.Vars.Add("venom.executor.filename", ux.Filename)
	tc.Vars.Add("venom.executor.name", ux.Executor)
	tc.computedVars = H{}

	Debug(ctx, "running user executor %v", tc.Name)
	Debug(ctx, "with vars: %v", vrs)

	v.runTestSteps(ctx, tc, tsIn)

	computedVars, err := DumpString(tc.computedVars)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to dump testcase computedVars")
	}

	type Output struct {
		Result json.RawMessage `json:"result"`
	}
	output := Output{
		Result: ux.Output,
	}
	outputString, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	// the value of each var can contains a double-quote -> "
	// if the value is not escaped, it will be used as is, and the json sent to unmarshall will be incorrect.
	// This also avoids injections into the json structure of a user executor
	// Use strconv.Quote for proper escaping to avoid double-escaping issues
	for i := range computedVars {
		computedVars[i] = escapeQuotes(computedVars[i])
	}

	outputS, err := interpolate.Do(string(outputString), computedVars)
	if err != nil {
		return nil, err
	}

	var outputResult interface{}
	if err := yaml.Unmarshal([]byte(outputS), &outputResult); err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal")
	}

	// here, we have the user executor results.
	// and for each key in output, we try to add the json version
	// this will allow user to use json version of output (map, etc...)
	// because, it's not possible to to that:
	// output:
	//   therawout: {{.result.systemout}}
	//
	// test is in file user_executor.yml

	result, err := Dump(outputResult)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to compute result")
	}

	if len(tsIn.Errors) > 0 {
		Error(ctx, "user executor %q failed - raw interpolated:\n%v\n", ux.Executor, string(sanitized))
		return outputResult, fmt.Errorf("executor %q failed", ux.Executor)
	}

	for k, v := range result {
		switch z := v.(type) {
		case string:
			var outJSON interface{}
			if err := JSONUnmarshal([]byte(z), &outJSON); err == nil {
				result[k+"json"] = outJSON
				// Now we have to dump this object, but the key will change if this is a array or not
				if reflect.ValueOf(outJSON).Kind() == reflect.Slice {
					prefix := k + "json"
					splitPrefix := strings.Split(prefix, ".")
					prefix += "." + splitPrefix[len(splitPrefix)-1]
					outJSONDump, err := Dump(outJSON)
					if err != nil {
						return nil, errors.Wrapf(err, "unable to compute result")
					}
					for ko, vo := range outJSONDump {
						result[prefix+ko] = vo
					}
				} else {
					outJSONDump, err := DumpWithPrefix(outJSON, k+"json")
					if err != nil {
						return nil, errors.Wrapf(err, "unable to compute result")
					}
					for ko, vo := range outJSONDump {
						result[ko] = vo
					}
				}
			}
		}
	}
	return result, nil
}

// quoteTemplateExpressions adds double quotes around template expressions in YAML content.
// It specifically targets expressions that follow a colon and whitespace like 'key: {{.variable}}'
// and are not already enclosed in quotes. This ensures proper YAML parsing of template variables.
func quoteTemplateExpressions(content []byte) []byte {
	// First capture group matches everything up to the colon, checking the last non-whitespace
	// character isn't a quote (to skip JSON keys)
	re := regexp.MustCompile(`(?m)(^.*[^"\s][\s]*)(:\s+)({{.*?}})(.*?)(?:\s*)$`)

	// Put quotes around the template expression and what follows it
	return re.ReplaceAll(content, []byte(`$1$2"$3$4"`))
}
