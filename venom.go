package venom

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/fsamin/go-dump"
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

	testsuites      []TestSuite
	variables       H
	IgnoreVariables []string

	EnableProfiling bool
	OutputFormat    string
	OutputDir       string
	StopOnFailure   bool
}

func (v *Venom) Print(format string, a ...interface{}) {
	v.PrintFunc(format, a...) // nolint
}

func (v *Venom) Println(format string, a ...interface{}) {
	v.PrintFunc(format+"\n", a...) // nolint
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
func (v *Venom) GetExecutorRunner(ctx context.Context, t TestStep, h H) (context.Context, ExecutorRunner, error) {
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

	info, _ := t.StringSliceValue("info")
	vars, err := dump.ToStringMap(h)
	if err != nil {
		return ctx, nil, err
	}

	for k, v := range vars {
		ctx = context.WithValue(ctx, ContextKey("var."+k), v)
	}

	if ex, ok := v.executors[name]; ok {
		return ctx, newExecutorRunner(ex, name, retry, delay, timeout, info), nil
	}

	return ctx, nil, fmt.Errorf("executor %q is not implemented", name)
}

func StringVarFromCtx(ctx context.Context, varname string) string {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToString(i)
}

func StringSliceVarFromCtx(ctx context.Context, varname string) []string {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToStringSlice(i)
}

func IntVarFromCtx(ctx context.Context, varname string) int {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToInt(i)
}

func BoolVarFromCtx(ctx context.Context, varname string) bool {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToBool(i)
}

func StringMapInterfaceVarFromCtx(ctx context.Context, varname string) map[string]interface{} {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToStringMap(i)
}

func StringMapStringVarFromCtx(ctx context.Context, varname string) map[string]string {
	i := ctx.Value(ContextKey("var." + varname))
	return cast.ToStringMapString(i)
}
