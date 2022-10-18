package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gosimple/slug"
	"github.com/ovh/cds/sdk/interpolate"
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
	Executor     string            `json:"executor" yaml:"executor"`
	Input        H                 `json:"input" yaml:"input"`
	RawTestSteps []json.RawMessage `json:"steps" yaml:"steps"`
	Output       json.RawMessage   `json:"output" yaml:"output"`
	Filename     string            `json:"-" yaml:"-"`
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

func (v *Venom) RunUserExecutor(ctx context.Context, runner ExecutorRunner, tcIn *TestCase, tsIn *TestStepResult, step TestStep) (interface{}, error) {
	vrs := tcIn.TestSuiteVars.Clone()
	uxIn := runner.GetExecutor().(UserExecutor)

	for k, va := range uxIn.Input {
		if strings.HasPrefix(k, "input.") {
			// do not reinject input.vars from parent user executor if exists
			continue
		} else if !strings.HasPrefix(k, "venom") {
			if vl, ok := step[k]; ok && vl != "" { // value from step
				vrs.AddWithPrefix("input", k, vl)
			} else { // default value from executor
				vrs.AddWithPrefix("input", k, va)
			}
		} else {
			vrs.Add(k, va)
		}
	}
	// reload the user executor with the interpolated vars
	_, exe, err := v.GetExecutorRunner(ctx, step, vrs)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to reload executor")
	}
	ux := exe.GetExecutor().(UserExecutor)

	tc := &TestCase{
		TestCaseInput: TestCaseInput{
			Name:         ux.Executor,
			RawTestSteps: ux.RawTestSteps,
			Vars:         vrs,
		},
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
	for i := range computedVars {
		computedVars[i] = strings.ReplaceAll(computedVars[i], "\"", "\\\"")
	}

	outputS, err := interpolate.Do(string(outputString), computedVars)
	if err != nil {
		return nil, err
	}

	// re-inject info into executorRunner
	b := runner.(*executor)
	b.info = append(b.info, tc.computedInfo...)

	var outputResult interface{}
	if err := yaml.Unmarshal([]byte(outputS), &outputResult); err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal")
	}

	if len(tsIn.Errors) > 0 {
		return outputResult, fmt.Errorf("failed")
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
