package venom

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/ovh/venom/executors"
	"github.com/spf13/cast"
)

type H map[string]interface{}

func (h H) Clone() H {
	var h2 = make(H, len(h))
	h2.AddAll(h)
	return h2
}

func (h *H) Add(k string, v interface{}) {
	if h == nil {
		var _h = H{}
		*h = _h
	}
	(*h)[k] = v
}

func (h *H) AddWithPrefix(p, k string, v interface{}) {
	(*h)[p+"."+k] = v
}

func (h *H) AddAll(h2 H) {
	for k, v := range h2 {
		h.Add(k, v)
	}
}

func (h *H) AddAllWithPrefix(p string, h2 H) {
	if h2 == nil {
		return
	}
	if h == nil {
		var _h = H{}
		*h = _h
	}
	for k, v := range h2 {
		h.AddWithPrefix(p, k, v)
	}
}

// Aliases contains list of aliases
type Aliases map[string]string

func GetExecutorResult(r interface{}) map[string]interface{} {
	d, err := executors.Dump(r)
	if err != nil {
		panic(err)
	}
	return d
}

// StepAssertions contains step assertions
type StepAssertions struct {
	Assertions []string `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

// Executor execute a testStep.
type Executor interface {
	// Run run a Test Step
	Run(context.Context, TestStep, string) (interface{}, error)
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

// Tests contains all informations about tests in a pipeline build
type Tests struct {
	XMLName      xml.Name    `xml:"testsuites" json:"-" yaml:"-"`
	Total        int         `xml:"-" json:"total"`
	TotalOK      int         `xml:"-" json:"ok"`
	TotalKO      int         `xml:"-" json:"ko"`
	TotalSkipped int         `xml:"-" json:"skipped"`
	TestSuites   []TestSuite `xml:"testsuite" json:"test_suites"`
}

// TestSuite is a single JUnit test suite which may contain many
// testcases.
type TestSuite struct {
	XMLName      xml.Name   `xml:"testsuite" json:"-" yaml:"-"`
	Disabled     int        `xml:"disabled,attr,omitempty" json:"disabled" yaml:""`
	Errors       int        `xml:"errors,attr,omitempty" json:"errors" yaml:"-"`
	Failures     int        `xml:"failures,attr,omitempty" json:"failures" yaml:"-"`
	Hostname     string     `xml:"hostname,attr,omitempty" json:"hostname" yaml:"-"`
	ID           string     `xml:"id,attr,omitempty" json:"id" yaml:"-"`
	Name         string     `xml:"name,attr" json:"name" yaml:"name"`
	Filename     string     `xml:"-" json:"-" yaml:"-"`
	ShortName    string     `xml:"-" json:"-" yaml:"-"`
	Package      string     `xml:"package,attr,omitempty" json:"package" yaml:"-"`
	Properties   []Property `xml:"-" json:"properties" yaml:"-"`
	Skipped      int        `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Total        int        `xml:"tests,attr" json:"total" yaml:"total,omitempty"`
	TestCases    []TestCase `xml:"testcase" hcl:"testcase" json:"testcases" yaml:"testcases"`
	Version      string     `xml:"version,omitempty" hcl:"version" json:"version" yaml:"version,omitempty"`
	Time         string     `xml:"time,attr,omitempty" json:"time" yaml:"-"`
	Timestamp    string     `xml:"timestamp,attr,omitempty" json:"timestamp" yaml:"-"`
	Vars         H          `xml:"-" json:"-" yaml:"vars"`
	ComputedVars H          `xml:"-" json:"-" yaml:"-"`
	WorkDir      string     `xml:"-" json:"-" yaml:"-"`
}

// Property represents a key/value pair used to define properties.
type Property struct {
	XMLName xml.Name `xml:"property" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr" json:"name" yaml:"-"`
	Value   string   `xml:"value,attr" json:"value" yaml:"-"`
}

// TestCase is a single test case with its result.
type TestCase struct {
	XMLName         xml.Name  `xml:"testcase" json:"-" yaml:"-"`
	Classname       string    `xml:"classname,attr,omitempty" json:"classname" yaml:"-"`
	Errors          []Failure `xml:"error,omitempty" json:"errors" yaml:"errors,omitempty"`
	Failures        []Failure `xml:"failure,omitempty" json:"failures" yaml:"failures,omitempty"`
	Name            string    `xml:"name,attr" json:"name" yaml:"name"`
	originalName    string
	Skipped         []Skipped         `xml:"skipped,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Status          string            `xml:"status,attr,omitempty" json:"status" yaml:"status,omitempty"`
	Systemout       InnerResult       `xml:"system-out,omitempty" json:"systemout" yaml:"systemout,omitempty"`
	Systemerr       InnerResult       `xml:"system-err,omitempty" json:"systemerr" yaml:"systemerr,omitempty"`
	Time            string            `xml:"time,attr,omitempty" json:"time" yaml:"time,omitempty"`
	RawTestSteps    []json.RawMessage `xml:"-" hcl:"step" json:"steps" yaml:"steps"`
	testSteps       []TestStep
	Vars            H `xml:"-" json:"-" yaml:"vars"`
	computedVars    H
	computedInfo    []string
	computedVerbose []string
	Skip            []string `xml:"-" json:"skip" yaml:"skip"`
}

// TestStep represents a testStep
type TestStep map[string]interface{}

func (t TestStep) IntValue(name string) (int, error) {
	out, err := cast.ToIntE(t[name])
	if err != nil {
		return -1, fmt.Errorf("attribute %q is not an integer", name)
	}
	return out, nil
}

func (t TestStep) StringValue(name string) (string, error) {
	out, err := cast.ToStringE(t[name])
	if err != nil {
		return "", fmt.Errorf("attribute %q is not an string", name)
	}
	return out, nil
}

func (t TestStep) StringSliceValue(name string) ([]string, error) {
	out, err := cast.ToStringE(t[name])
	if err != nil {
		out, err := cast.ToStringSliceE(t[name])
		if err != nil {
			return nil, fmt.Errorf("attribute %q is neither a string nor a string array", name)
		}
		return out, nil
	}
	return []string{out}, nil
}

// Skipped contains data related to a skipped test.
type Skipped struct {
	Value string `xml:",cdata" json:"value" yaml:"value,omitempty"`
}

func (tc *TestCase) AppendError(err error) {
	tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
}

// Failure contains data related to a failed test.
type Failure struct {
	TestcaseClassname  string `xml:"-" json:"-" yaml:"-"`
	TestcaseName       string `xml:"-" json:"-" yaml:"-"`
	TestcaseLineNumber int    `xml:"-" json:"-" yaml:"-"`
	StepNumber         int    `xml:"-" json:"-" yaml:"-"`
	Assertion          string `xml:"-" json:"-" yaml:"-"`
	Error              error  `xml:"-" json:"-" yaml:"-"`

	Value   string `xml:",cdata" json:"value" yaml:"value,omitempty"`
	Type    string `xml:"type,attr,omitempty" json:"type" yaml:"type,omitempty"`
	Message string `xml:"message,attr,omitempty" json:"message" yaml:"message,omitempty"`
}

func newFailure(tc TestCase, stepNumber int, assertion string, err error) *Failure {
	var lineNumber = findLineNumber(tc.Classname, tc.originalName, stepNumber, assertion)
	var value string
	if assertion != "" {
		value = color.YellowString(`Testcase %q, step #%d: Assertion %q failed. %s (%v:%d)`,
			tc.originalName,
			stepNumber,
			RemoveNotPrintableChar(assertion),
			RemoveNotPrintableChar(err.Error()),
			tc.Classname,
			lineNumber,
		)
	} else {
		value = color.YellowString(`Testcase %q, step #%d: %s (%v:%d)`,
			tc.originalName,
			stepNumber,
			RemoveNotPrintableChar(err.Error()),
			tc.Classname,
			lineNumber,
		)
	}

	var failure = Failure{
		TestcaseClassname:  tc.Classname,
		TestcaseName:       tc.Name,
		TestcaseLineNumber: lineNumber,
		StepNumber:         stepNumber,
		Assertion:          assertion,
		Error:              err,
		Value:              value,
	}

	return &failure
}

func (f Failure) String() string {
	if f.Value != "" {
		return f.Value
	}
	if f.Error != nil {
		return f.Error.Error()
	}
	return f.Message
}

// InnerResult is used by TestCase
type InnerResult struct {
	Value string `xml:",cdata" json:"value" yaml:"value"`
}

type AssignStep struct {
	Assignments map[string]Assignment `json:"vars" yaml:"vars" mapstructure:"vars"`
}

type Assignment struct {
	From  string `json:"from" yaml:"from"`
	Regex string `json:"regex" yaml:"regex"`
}

// RemoveNotPrintableChar removes not printable chararacter from a string
func RemoveNotPrintableChar(in string) string {
	m := func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) || unicode.IsPunct(r) {
			return r
		}
		return ' '
	}
	return strings.Map(m, in)
}
