package venom

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gosimple/slug"
	"github.com/ovh/venom/executors"
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
}

func (e executor) Name() string {
	return e.name
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

func newExecutorRunner(e Executor, name string, retry, delay, timeout int, info []string) ExecutorRunner {
	return &executor{
		Executor: e,
		name:     name,
		retry:    retry,
		delay:    delay,
		timeout:  timeout,
		info:     info,
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
	Executor     string             `json:"executor" yaml:"executor"`
	Input        H                  `json:"input" yaml:"input"`
	RawTestSteps []json.RawMessage  `json:"steps" yaml:"steps"`
	Output       UserExecutorOutput `json:"output" yaml:"output"`
	v            *Venom             `json:"-" yaml:"-"`
}

type UserExecutorOutput struct {
	Result string `json:"result" yaml:"result"`
}

func (ux UserExecutor) Run(ctx context.Context, step TestStep) (interface{}, error) {
	vrs := H{}
	for k, v := range ux.Input {
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
				vrs.Add("input."+k, v)
			}

		} else {
			vrs.Add(k, v)
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

	Debug(ctx, "running user executor %v\n", tc.Name)
	Debug(ctx, "with vars: %v", vrs)

	ux.v.runTestSteps(ctx, tc)
	return nil, nil
}
