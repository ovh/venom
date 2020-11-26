package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/ghodss/yaml"
	"github.com/gosimple/slug"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/venom/executors"
	"github.com/pkg/errors"
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
	x, ok := e.Executor.(executorWithDefaultAssertions)
	if ok {
		return x.GetDefaultAssertions()
	}
	return nil
}

func (e executor) ZeroValueResult() interface{} {
	x, ok := e.Executor.(executorWithZeroValueResult)
	if ok {
		return x.ZeroValueResult()
	}
	return nil
}

func (e executor) Setup(ctx context.Context, vars H) (context.Context, error) {
	x, ok := e.Executor.(ExecutorWithSetup)
	if ok {
		return x.Setup(ctx, vars)
	}
	return ctx, nil
}

func (e executor) TearDown(ctx context.Context) error {
	x, ok := e.Executor.(ExecutorWithSetup)
	if ok {
		return x.TearDown(ctx)
	}
	return nil
}

func newExecutorRunner(e Executor, name, stype string, retry, delay, timeout int, info []string) ExecutorRunner {
	return &executor{
		Executor: e,
		name:     name,
		retry:    retry,
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
	d, err := executors.Dump(r)
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
}

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
	err = json.Unmarshal(outputS, &result)
	if err != nil {
		return ""
	}
	return result
}

func (v *Venom) RunUserExecutor(ctx context.Context, ux UserExecutor, step TestStep) (interface{}, error) {
	vrs := H{}
	for k, va := range ux.Input {
		if !strings.HasPrefix(k, "venom") {
			vl, err := step.StringValue(k)
			if err != nil {
				return nil, err
			}
			if vl != "" {
				// value from step
				vrs.Add("input."+k, vl)
			} else {
				// default value from executor
				vrs.Add("input."+k, va)
			}
		} else {
			vrs.Add(k, va)
		}
	}

	tc := &TestCase{
		Name:         ux.Executor,
		RawTestSteps: ux.RawTestSteps,
		Vars:         vrs,
	}

	tc.originalName = tc.Name
	tc.Name = slug.Make(tc.Name)
	tc.Vars.Add("venom.testcase", tc.Name)
	tc.computedVars = H{}

	Debug(ctx, "running user executor %v", tc.Name)
	Debug(ctx, "with vars: %v", vrs)

	v.runTestSteps(ctx, tc)

	computedVars, err := dump.ToStringMap(tc.computedVars)
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
	outputS, err := interpolate.Do(string(outputString), computedVars)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := yaml.Unmarshal([]byte(outputS), &result); err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal output")
	}
	if len(tc.Errors) > 0 || len(tc.Failures) > 0 {
		return result, fmt.Errorf("failed")
	}
	return result, nil
}
