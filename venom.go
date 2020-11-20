package venom

import (
	"context"
	"fmt"
	"io"
	"os"
	"plugin"

	"github.com/fsamin/go-dump"
	"github.com/spf13/cast"
)

var (
	//Version is set with -ldflags "-X github.com/ovh/venom/venom.Version=$(VERSION)"
	Version = "snapshot"
)

func New() *Venom {
	v := &Venom{
		LogOutput:       os.Stdout,
		PrintFunc:       fmt.Printf,
		executors:       map[string]Executor{},
		variables:       map[string]interface{}{},
		IgnoreVariables: []string{},
		OutputFormat:    "xml",
	}
	return v
}

type Venom struct {
	LogOutput io.Writer

	PrintFunc func(format string, a ...interface{}) (n int, err error)
	executors map[string]Executor

	testsuites      []TestSuite
	variables       H
	IgnoreVariables []string

	OutputFormat  string
	OutputDir     string
	StopOnFailure bool
	Verbose       int
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

// GetExecutorRunner initializes a test by name
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

	allKeys := []string{}
	for k, v := range vars {
		ctx = context.WithValue(ctx, ContextKey("var."+k), v)
		allKeys = append(allKeys, k)
	}
	ctx = context.WithValue(ctx, ContextKey("vars"), allKeys)

	if ex, ok := v.executors[name]; ok {
		return ctx, newExecutorRunner(ex, name, retry, delay, timeout, info), nil
	}

	// try to load executor as a plugin
	if err := v.registerPlugin(ctx, name); err != nil {
		Debug(ctx, "executor %q is not implemented as plugin - err:%v", name, err)
	}

	// then add the executor plugin to the map to not have to load it on each step
	if ex, ok := v.executors[name]; ok {
		return ctx, newExecutorRunner(ex, name, retry, delay, timeout, info), nil
	}
	return ctx, nil, fmt.Errorf("executor %q is not implemented", name)
}

func (v *Venom) registerPlugin(ctx context.Context, name string) error {
	p, err := plugin.Open("lib/" + name + ".so")
	if err != nil {
		return err
	}

	symbolExecutor, err := p.Lookup("Plugin")
	if err != nil {
		return err
	}

	executor := symbolExecutor.(Executor)
	v.RegisterExecutor(name, executor)

	return nil
}

func VarFromCtx(ctx context.Context, varname string) interface{} {
	i := ctx.Value(ContextKey("var." + varname))
	return i
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

func AllVarsFromCtx(ctx context.Context) H {
	i := ctx.Value(ContextKey("vars"))
	allKeys := cast.ToStringSlice(i)
	res := H{}
	for _, k := range allKeys {
		res.Add(k, VarFromCtx(ctx, k))
	}
	return res
}
