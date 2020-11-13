package venom

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cast"
)

var (
	//Version is set with -ldflags "-X github.com/ovh/venom/venom.Version=$(VERSION)"
	Version = "snapshot"
)

func New() *Venom {
	v := &Venom{
		LogLevel:        "info",
		LogOutput:       os.Stdout,
		PrintFunc:       fmt.Printf,
		executors:       map[string]Executor{},
		contexts:        map[string]TestCaseContext{},
		variables:       map[string]interface{}{},
		EnableProfiling: false,
		IgnoreVariables: []string{},
		OutputFormat:    "xml",
	}
	return v
}

type Venom struct {
	LogLevel  string
	LogOutput io.Writer

	PrintFunc func(format string, a ...interface{}) (n int, err error)
	executors map[string]Executor
	contexts  map[string]TestCaseContext

	testsuites      []TestSuite
	variables       H
	IgnoreVariables []string
	Parallel        int

	EnableProfiling bool
	OutputFormat    string
	OutputDir       string
	StopOnFailure   bool
}

func (v *Venom) AddVariables(variables map[string]interface{}) {
	for k, variable := range variables {
		v.variables[k] = variable
	}
}

// RegisterExecutor register Test Executors
func (v *Venom) RegisterExecutor(name string, e Executor) {
	v.executors[name] = e
}

// WrapExecutor initializes a test by name
// no type -> exec is default
func (v *Venom) GetExecutorRunner(ctx context.Context, t TestStep, vars H) (context.Context, ExecutorRunner, error) {
	name, _ := t.StringValue("type")
	if name == "" {
		name = "exec"
	}
	retry, err := t.IntValue("retry")
	if err != nil {
		return nil, nil, err
	}
	delay, err := t.IntValue("delay")
	if err != nil {
		return nil, nil, err
	}
	timeout, err := t.IntValue("timeout")
	if err != nil {
		return nil, nil, err
	}

	for k, v := range vars {
		ctx = context.WithValue(ctx, ContextKey("var."+k), v)
	}

	if ex, ok := v.executors[name]; ok {
		return ctx, newExecutorRunner(ex, name, retry, delay, timeout), nil
	}

	return ctx, nil, fmt.Errorf("executor %q is not implemented", name)
}

// RegisterTestCaseContext new register TestCaseContext
func (v *Venom) RegisterTestCaseContext(name string, tcc TestCaseContext) {
	v.contexts[name] = tcc
}

// ContextWrap initializes a context for a testcase
// no type -> parent context
func (v *Venom) ContextWrap(tc *TestCase) (TestCaseContext, error) {
	if tc.Context == nil {
		return v.contexts["default"], nil
	}
	var typeName string
	if itype, ok := tc.Context["type"]; ok {
		typeName = fmt.Sprintf("%s", itype)
	}

	if typeName == "" {
		return v.contexts["default"], nil
	}
	v.contexts[typeName].SetTestCase(*tc)
	return v.contexts[typeName], nil
}

func StringVarFromCtx(ctx context.Context, varname string) string {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToString(i)
}

func IntVarFromCtx(ctx context.Context, varname string) int {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToInt(i)
}

func BoolVarFromCtx(ctx context.Context, varname string) bool {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToBool(i)
}

func VarFromCtx(ctx context.Context, varname string) interface{} {
	i := ctx.Value(ContextKey("var." + varname))
	return i
}
