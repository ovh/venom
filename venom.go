package venom

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"plugin"

	"github.com/fatih/color"
	"github.com/fsamin/go-dump"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

var (
	//Version is set with -ldflags "-X github.com/ovh/venom/venom.Version=$(VERSION)"
	Version = "snapshot"
)

func New() *Venom {
	v := &Venom{
		LogOutput:        os.Stdout,
		PrintFunc:        fmt.Printf,
		executorsBuiltin: map[string]Executor{},
		executorsPlugin:  map[string]Executor{},
		executorsUser:    map[string]Executor{},
		variables:        map[string]interface{}{},
		OutputFormat:     "xml",
	}
	return v
}

type Venom struct {
	LogOutput io.Writer

	PrintFunc        func(format string, a ...interface{}) (n int, err error)
	executorsBuiltin map[string]Executor
	executorsPlugin  map[string]Executor
	executorsUser    map[string]Executor

	testsuites []TestSuite
	variables  H

	OutputFormat  string
	OutputDir     string
	StopOnFailure bool
	Verbose       int
}

var trace = color.New(color.Attribute(90)).SprintFunc()

func (v *Venom) Print(format string, a ...interface{}) {
	v.PrintFunc(format, a...) // nolint
}

func (v *Venom) Println(format string, a ...interface{}) {
	v.PrintFunc(format+"\n", a...) // nolint
}

func (v *Venom) PrintlnTrace(s string) {
	v.Println("\t  %s %s", trace("[trac]"), trace(s)) // nolint
}

func (v *Venom) AddVariables(variables map[string]interface{}) {
	for k, variable := range variables {
		v.variables[k] = variable
	}
}

// RegisterExecutorBuiltin register builtin executors
func (v *Venom) RegisterExecutorBuiltin(name string, e Executor) {
	v.executorsBuiltin[name] = e
}

// RegisterExecutorPlugin register plugin executors
func (v *Venom) RegisterExecutorPlugin(name string, e Executor) {
	v.executorsPlugin[name] = e
}

// RegisterExecutorUser register User sxecutors
func (v *Venom) RegisterExecutorUser(name string, e Executor) {
	v.executorsUser[name] = e
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

	if ex, ok := v.executorsBuiltin[name]; ok {
		return ctx, newExecutorRunner(ex, name, "builtin", retry, delay, timeout, info), nil
	}

	if err := v.registerUserExecutors(ctx, name, vars); err != nil {
		Debug(ctx, "executor %q is not implemented as user executor - err:%v", name, err)
	}

	if ex, ok := v.executorsUser[name]; ok {
		return ctx, newExecutorRunner(ex, name, "user", retry, delay, timeout, info), nil
	}

	if err := v.registerPlugin(ctx, name, vars); err != nil {
		Debug(ctx, "executor %q is not implemented as plugin - err:%v", name, err)
	}

	// then add the executor plugin to the map to not have to load it on each step
	if ex, ok := v.executorsUser[name]; ok {
		return ctx, newExecutorRunner(ex, name, "plugin", retry, delay, timeout, info), nil
	}
	return ctx, nil, fmt.Errorf("executor %q is not implemented", name)
}

func (v *Venom) registerUserExecutors(ctx context.Context, name string, vars map[string]string) error {
	workdir := vars["venom.testsuite.workdir"]
	executorsPath, err := getFilesPath([]string{path.Join(workdir, "lib")})
	if err != nil {
		return err
	}

	for _, f := range executorsPath {
		log.Info("Reading ", f)
		btes, err := ioutil.ReadFile(f)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %q", f)
		}

		ux := UserExecutor{}
		if err := yaml.Unmarshal(btes, &ux); err != nil {
			return errors.Wrapf(err, "unable to parse file %q", f)
		}

		for k, vr := range vars {
			ux.Input.Add(k, vr)
		}

		v.RegisterExecutorUser(name, ux)
	}
	return nil
}

func (v *Venom) registerPlugin(ctx context.Context, name string, vars map[string]string) error {
	workdir := vars["venom.testsuite.workdir"]
	// try to load from testsuite path
	p, err := plugin.Open(path.Join(workdir, "lib", name+".so"))
	if err != nil {
		// try to load from venom binary path
		p, err = plugin.Open(path.Join("lib", name+".so"))
		if err != nil {
			return fmt.Errorf("unable to load plugin %q.so", name)
		}
	}

	symbolExecutor, err := p.Lookup("Plugin")
	if err != nil {
		return err
	}

	executor := symbolExecutor.(Executor)
	v.RegisterExecutorPlugin(name, executor)

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
