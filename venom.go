package venom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/bincover"
	"github.com/fatih/color"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/venom/reporting"
	"github.com/pkg/errors"
	"github.com/rockbears/yaml"
	"github.com/spf13/cast"
	"io"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"reflect"
	"sort"
	"strings"

	"github.com/confluentinc/bincover"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

var (
	// Version is set with -ldflags "-X github.com/ovh/venom/venom.Version=$(VERSION)"
	Version = "snapshot"
	IsTest  = ""
)

// OSExit is a wrapper for os.Exit
func OSExit(exitCode int) {
	if IsTest != "" {
		bincover.ExitCode = exitCode
	} else {
		os.Exit(exitCode)
	}
}

// ContextKey can be added in context to store contextual infos. Also used by logger.
type ContextKey string

// New instantiates a new venom on venom run cmd
func New() *Venom {
	v := &Venom{
		LogOutput:        os.Stdout,
		PrintFunc:        fmt.Printf,
		executorsBuiltin: map[string]Executor{},
		executorsPlugin:  map[string]Executor{},
		executorsUser:    map[string]Executor{},
		variables:        map[string]interface{}{},
		secrets:          map[string]interface{}{},
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

	Tests     Tests
	variables H
	secrets   H

	LibDir        string
	OutputFormat  string
	OutputDir     string
	StopOnFailure bool
	HtmlReport    bool
	Verbose       int
	OpenApiReport bool
}

var trace = color.New(color.Attribute(90)).SprintFunc()

func (v *Venom) Print(format string, a ...interface{}) {
	v.PrintFunc(format, a...) // nolint
}

func (v *Venom) Println(format string, a ...interface{}) {
	v.PrintFunc(format+"\n", a...) // nolint
}

func (v *Venom) PrintlnTrace(s string) {
	v.PrintlnIndentedTrace(s, "")
}

func (v *Venom) PrintlnIndentedTrace(s string, indent string) {
	v.Println("\t  %s%s %s", indent, trace("[trac]"), trace(s)) // nolint
}

func (v *Venom) AddVariables(variables map[string]interface{}) {
	for k, variable := range variables {
		v.variables[k] = variable
	}
}

func (v *Venom) AddSecrets(secrets map[string]interface{}) {
	for k, s := range secrets {
		v.secrets[k] = s
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

// RegisterExecutorUser registers an user executor
func (v *Venom) RegisterExecutorUser(name string, e Executor) error {
	if existing, ok := v.executorsUser[name]; ok {
		return fmt.Errorf("executor %q already exists (from file %q)", name, existing.(UserExecutor).Filename)
	}

	v.executorsUser[name] = e
	return nil
}

// GetExecutorRunner initializes a test according to its type
// if no type is provided, exec is default
func (v *Venom) GetExecutorRunner(ctx context.Context, ts TestStep, h H) (context.Context, ExecutorRunner, error) {
	name, _ := ts.StringValue("type")
	script, _ := ts.StringValue("script")
	command, _ := ts.StringSliceValue("command")
	if name == "" && (script != "" || len(command) != 0) {
		name = "exec"
	}
	retry, err := ts.IntValue("retry")
	if err != nil {
		return nil, nil, err
	}
	retryIf, err := ts.StringSliceValue("retry_if")
	if err != nil {
		return nil, nil, err
	}
	delay, err := ts.IntValue("delay")
	if err != nil {
		return nil, nil, err
	}
	timeout, err := ts.IntValue("timeout")
	if err != nil {
		return nil, nil, err
	}

	info, _ := ts.StringSliceValue("info")
	vars, err := DumpStringPreserveCase(h)
	if err != nil {
		return ctx, nil, err
	}

	allKeys := []string{}
	for k, v := range vars {
		ctx = context.WithValue(ctx, ContextKey("var."+k), v)
		allKeys = append(allKeys, k)
	}
	ctx = context.WithValue(ctx, ContextKey("vars"), allKeys)

	if name == "" {
		return ctx, newExecutorRunner(nil, name, "builtin", retry, retryIf, delay, timeout, info), nil
	}

	if ex, ok := v.executorsBuiltin[name]; ok {
		return ctx, newExecutorRunner(ex, name, "builtin", retry, retryIf, delay, timeout, info), nil
	}

	if ex, ok := v.executorsUser[name]; ok {
		return ctx, newExecutorRunner(ex, name, "user", retry, retryIf, delay, timeout, info), nil
	}

	if err := v.registerPlugin(ctx, name, vars); err != nil {
		Debug(ctx, "executor %q is not implemented as plugin - err:%v", name, err)
	}

	// then add the executor plugin to the map to not have to load it on each step
	if ex, ok := v.executorsPlugin[name]; ok {
		return ctx, newExecutorRunner(ex, name, "plugin", retry, retryIf, delay, timeout, info), nil
	}
	return ctx, nil, fmt.Errorf("user executor %q not found - loaded executors are: %v", name, reflect.ValueOf(v.executorsUser).MapKeys())
}

func (v *Venom) getUserExecutorFilesPath(ctx context.Context, vars map[string]string) []string {
	var libpaths []string
	if v.LibDir != "" {
		p := strings.Split(v.LibDir, string(os.PathListSeparator))
		libpaths = append(libpaths, p...)
	}
	libpaths = append(libpaths, path.Join(vars["venom.testsuite.workdir"], "lib"))

	//use a map to avoid duplicates
	filepaths := make(map[string]bool)

	for _, p := range libpaths {
		p = strings.TrimSpace(p)

		err := filepath.Walk(p, func(fp string, f os.FileInfo, err error) error {
			switch ext := filepath.Ext(fp); ext {
			case ".yml", ".yaml":
				filepaths[fp] = true
			}
			return nil
		})
		if err != nil {
			Warn(ctx, "Unable to list files in lib directory %q: %v", p, err)
		}
	}

	userExecutorFiles := make([]string, len(filepaths))
	i := 0
	for p := range filepaths {
		userExecutorFiles[i] = p
		i++
	}

	sort.Strings(userExecutorFiles)
	if len(userExecutorFiles) == 0 {
		Warn(ctx, "no user executor yml file selected")
	}
	return userExecutorFiles
}

func (v *Venom) registerUserExecutors(ctx context.Context) error {
	vars, err := DumpStringPreserveCase(v.variables)
	if err != nil {
		return errors.Wrapf(err, "unable to parse variables")
	}

	executorsPath := v.getUserExecutorFilesPath(ctx, vars)

	for _, f := range executorsPath {
		Info(ctx, "Reading %v", f)
		content, err := os.ReadFile(f)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %q", f)
		}

		ex := readPartialYML(content, "executor")
		if len(ex) == 0 {
			return errors.Errorf("missing key 'executor' in %q", f)
		}

		name := strings.Replace(ex, "executor:", "", 1)
		name = strings.TrimSpace(name)

		inputs := readPartialYML(content, "input")

		ux := UserExecutor{
			Filename:  f,
			Executor:  name,
			RawInputs: []byte(inputs),
			Raw:       content,
		}

		err = v.RegisterExecutorUser(ux.Executor, ux)
		if err != nil {
			return errors.Wrapf(err, "unable to register user executor %q from file %q", ux.Executor, f)
		}
		Info(ctx, "User executor %q registered", ux.Executor)
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

func JSONUnmarshal(btes []byte, i interface{}) error {
	d := json.NewDecoder(bytes.NewReader(btes))
	d.UseNumber()
	return d.Decode(i)
}

func (v *Venom) GenerateOpenApiReport() error {
	pattern := v.variables["openapi-report-pattern"]
	strPattern := fmt.Sprintf("%v", pattern)

	var files []reporting.FileEntry
	dirs, err := filepath.Glob(strPattern)
	if err != nil {
		fmt.Printf("Error finding directories with pattern %q: %v\n", strPattern, err)
		return nil
	}

	if len(dirs) == 0 {
		fmt.Printf("No directories match the pattern %q\n", strPattern)
		return nil
	}

	// Collect JSON (OpenAPI specs) and XML (JUnit results)
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				ext := filepath.Ext(d.Name())
				if ext == ".json" || ext == ".xml" {
					files = append(files, reporting.FileEntry{Path: path, Entry: d})
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking the path %q: %v\n", dir, err)
		}
	}

	var openAPIs []*reporting.OpenAPI

	for _, file := range files {
		if strings.HasSuffix(file.Entry.Name(), ".json") && !strings.Contains(file.Entry.Name(), "dump") {
			tmpOpenAPI, err := reporting.LoadOpenAPISpec(file.Path)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			openAPIs = append(openAPIs, tmpOpenAPI)
		}
	}

	if len(openAPIs) == 0 {
		return errors.Errorf("No OpenAPI Spec file found")
	}

	openAPIEndpoints := make(map[string]int)

	// Merge all endpoints from each spec into openAPIEndpoints
	for _, oapi := range openAPIs {
		endpoints := getAllEndpointsFromTyped(oapi)
		for _, ec := range endpoints {
			key := ec.Method + " " + ec.Path
			openAPIEndpoints[key] = 0
		}
	}

	if len(openAPIEndpoints) == 0 {
		return errors.Errorf("No endpoints found in the provided OpenAPI Specs")
	}

	// Combine all OpenAPI specs into a single typed spec, so coverage can be done on it
	combinedOpenAPI := &reporting.OpenAPI{
		Paths: make(map[string]*reporting.PathItem),
	}

	// Merge logic: If two specs have the same path, we merge method definitions
	for _, oapi := range openAPIs {
		for pathKey, pathItem := range oapi.Paths {
			if existing, ok := combinedOpenAPI.Paths[pathKey]; ok {
				mergePathItems(existing, pathItem)
			} else {
				combinedOpenAPI.Paths[pathKey] = pathItem
			}
		}
	}

	var allCoverages []reporting.EndpointCoverage
	bigTestSuites := reporting.TestSuites{}

	for _, file := range files {
		if strings.HasSuffix(file.Entry.Name(), ".xml") {
			testsuites, err := reporting.LoadJUnitXML(file.Path)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			bigTestSuites.Testsuites = append(bigTestSuites.Testsuites, testsuites.Testsuites...)

			allCoverages = reporting.CalculateCoverage(combinedOpenAPI, &bigTestSuites)

			for _, testsuite := range testsuites.Testsuites {
				httpMethod, endpoint := reporting.ExtractHttpEndpoint(testsuite.Name)
				if httpMethod != "" {
					key := httpMethod + " " + endpoint
					if count, ok := openAPIEndpoints[key]; ok {
						openAPIEndpoints[key] = count + 1
					}
				}
			}
		}
	}

	var filename = filepath.Join(v.OutputDir, computeOutputFilename("open_api_report.txt"))
	var data []byte
	htmlData := make(map[string]int)

	for endpoint, count := range openAPIEndpoints {
		htmlData[endpoint] = count
		line := fmt.Sprintf("%s: %d\n", endpoint, count)
		data = append(data, []byte(line)...)
	}

	if v.HtmlReport && len(htmlData) > 0 {

		data, err := reporting.OpenApiOutputHtml(allCoverages)
		if err != nil {
			return errors.Wrapf(err, "Error: cannot format output html")
		}
		var filenameHTML = filepath.Join(v.OutputDir, computeOutputFilename("open_api_report.html"))
		v.PrintFunc("Writing html file %s\n", filenameHTML)
		if err := os.WriteFile(filenameHTML, data, 0600); err != nil {
			return errors.Wrapf(err, "Error while creating file %s", filenameHTML)
		}

		v.PrintFunc("Open HTML report written to %s\n", filenameHTML)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return errors.Wrapf(err, "Error while creating file %s", filename)
	}
	v.PrintFunc("Writing open api report file %s\n", filename)
	return nil
}

func getAllEndpointsFromTyped(oapi *reporting.OpenAPI) []reporting.EndpointCoverage {
	var endpoints []reporting.EndpointCoverage
	for p, pathItem := range oapi.Paths {
		if pathItem.Get != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "GET", Path: p})
		}
		if pathItem.Post != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "POST", Path: p})
		}
		if pathItem.Put != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "PUT", Path: p})
		}
		if pathItem.Patch != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "PATCH", Path: p})
		}
		if pathItem.Delete != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "DELETE", Path: p})
		}
		if pathItem.Head != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "HEAD", Path: p})
		}
		if pathItem.Options != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "OPTIONS", Path: p})
		}
		if pathItem.Trace != nil {
			endpoints = append(endpoints, reporting.EndpointCoverage{Method: "TRACE", Path: p})
		}
	}
	return endpoints
}

func mergePathItems(dst, src *reporting.PathItem) *reporting.PathItem {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	if src.Get != nil {
		dst.Get = src.Get
	}
	if src.Post != nil {
		dst.Post = src.Post
	}
	if src.Put != nil {
		dst.Put = src.Put
	}
	if src.Patch != nil {
		dst.Patch = src.Patch
	}
	if src.Delete != nil {
		dst.Delete = src.Delete
	}
	if src.Head != nil {
		dst.Head = src.Head
	}
	if src.Options != nil {
		dst.Options = src.Options
	}
	if src.Trace != nil {
		dst.Trace = src.Trace
	}

	return dst
}
