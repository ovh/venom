package venom

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/cast"
)

type Status string

const (
	StatusRun  Status = "RUN"
	StatusFail Status = "FAIL"
	StatusSkip Status = "SKIP"
	StatusPass Status = "PASS"
)

type H map[string]interface{}

func (h H) Clone() H {
	var h2 = make(H, len(h))
	h2.AddAll(h)
	return h2
}

func (h *H) Add(k string, v interface{}) {
	if h == nil || *h == nil {
		*h = make(map[string]interface{})
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

// Assertion (usually a string, but could be a dictionary when using logical operators)
type Assertion interface{}

// StepAssertions contains step assertions
type StepAssertions struct {
	Assertions []Assertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

type TestsXML struct {
	XMLName    xml.Name       `xml:"testsuites" json:"-" yaml:"-"`
	TestSuites []TestSuiteXML `xml:"testsuite" json:"test_suites"`
}

type Tests struct {
	TestSuites       []TestSuite `json:"test_suites" yml:"tests_suites"`
	Status           Status      `json:"status" yml:"status"`
	NbTestsuitesFail int         `json:"nbTestsuitesFail"  yaml:"-"`
	NbTestsuitesPass int         `json:"nbTestsuitesPass"  yaml:"-"`
	NbTestsuitesSkip int         `json:"nbTestsuitesSkip"  yaml:"-"`
	Duration         float64     `json:"duration" yaml:"-"`
	Start            time.Time   `json:"start" yaml:"-"`
	End              time.Time   `json:"end" yaml:"-"`
}

// TestSuite is a single JUnit test suite which may contain many
// testcases.
type TestSuiteXML struct {
	XMLName   xml.Name      `xml:"testsuite" json:"-" yaml:"-"`
	Disabled  int           `xml:"disabled,attr,omitempty" json:"disabled" yaml:""`
	Errors    int           `xml:"errors,attr,omitempty" json:"errors" yaml:"-"`
	Failures  int           `xml:"failures,attr,omitempty" json:"failures" yaml:"-"`
	Hostname  string        `xml:"hostname,attr,omitempty" json:"hostname" yaml:"-"`
	ID        string        `xml:"id,attr,omitempty" json:"id" yaml:"-"`
	Name      string        `xml:"name,attr" json:"name" yaml:"name"`
	Package   string        `xml:"package,attr,omitempty" json:"package" yaml:"-"`
	Skipped   int           `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Total     int           `xml:"tests,attr" json:"total" yaml:"total,omitempty"`
	TestCases []TestCaseXML `xml:"testcase" json:"testcases" yaml:"testcases"`
	Version   string        `xml:"version,omitempty" json:"version" yaml:"version,omitempty"`
	Time      string        `xml:"time,attr,omitempty" json:"time" yaml:"-"`
	Timestamp string        `xml:"timestamp,attr,omitempty" json:"timestamp" yaml:"-"`
}

type TestSuiteInput struct {
	Name      string          `json:"name" yaml:"name"`
	TestCases []TestCaseInput `json:"testcases" yaml:"testcases"`
	Vars      H               `json:"vars" yaml:"vars"`
}

type TestSuite struct {
	Name      string     `json:"name" yaml:"name"`
	TestCases []TestCase `json:"testcases" yaml:"testcases"`
	Vars      H          `json:"vars" yaml:"vars"`

	// computed
	ShortName    string `json:"shortname" yaml:"-"`
	Filename     string `json:"filename" yaml:"-"`
	Filepath     string `json:"filepath" yaml:"-"`
	ComputedVars H      `json:"computed_vars" yaml:"-"`
	WorkDir      string `json:"workdir" yaml:"_"`
	Status       Status `json:"status" yaml:"status"`

	Duration float64   `json:"duration" yaml:"-"`
	Start    time.Time `json:"start" yaml:"-"`
	End      time.Time `json:"end" yaml:"-"`

	NbTestcasesFail int `json:"nbTestcasesFail"  yaml:"-"`
	NbTestcasesPass int `json:"nbTestcasesPass"  yaml:"-"`
	NbTestcasesSkip int `json:"nbTestcasesSkip"  yaml:"-"`
}

// TestCase is a single test case with its result.
type TestCaseXML struct {
	XMLName   xml.Name     `xml:"testcase" json:"-" yaml:"-"`
	Classname string       `xml:"classname,attr,omitempty" json:"classname" yaml:"-"`
	Errors    []FailureXML `xml:"error,omitempty" json:"errors" yaml:"errors,omitempty"`
	Failures  []FailureXML `xml:"failure,omitempty" json:"failures" yaml:"failures,omitempty"`
	Name      string       `xml:"name,attr" json:"name" yaml:"name"`
	Skipped   []Skipped    `xml:"skipped,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Systemout InnerResult  `xml:"system-out,omitempty" json:"systemout" yaml:"systemout,omitempty"`
	Systemerr InnerResult  `xml:"system-err,omitempty" json:"systemerr" yaml:"systemerr,omitempty"`
	Time      float64      `xml:"time,attr,omitempty" json:"time" yaml:"time,omitempty"`
}

type TestCaseInput struct {
	Name         string            `json:"name" yaml:"name"`
	Vars         H                 `json:"vars" yaml:"vars"`
	Skip         []string          `json:"skip" yaml:"skip"`
	RawTestSteps []json.RawMessage `json:"steps" yaml:"steps"`
}

type TestCase struct {
	TestCaseInput

	// Computed
	originalName string
	Skipped      []Skipped `json:"skipped" yaml:"-"`
	Status       Status    `json:"status" yaml:"-"`

	Duration float64   `json:"duration" yaml:"-"`
	Start    time.Time `json:"start" yaml:"-"`
	End      time.Time `json:"end" yaml:"-"`

	testSteps       []TestStep       `json:"-" yaml:"-"`
	TestStepResults []TestStepResult `json:"results" yaml:"-"`
	TestSuiteVars   H                `json:"-" yaml:"-"`

	computedVars    H        `json:"-" yaml:"-"`
	computedVerbose []string `json:"-" yaml:"-"`
	IsExecutor      bool     `json:"-" yaml:"-"`
	IsEvaluated     bool     `json:"-" yaml:"-"`
}

type TestStepResult struct {
	Name              string            `json:"name"`
	Errors            []Failure         `json:"errors"`
	Skipped           []Skipped         `json:"skipped" yaml:"skipped"`
	Status            Status            `json:"status" yaml:"status"`
	Raw               interface{}       `json:"raw" yaml:"raw"`
	Interpolated      interface{}       `json:"interpolated" yaml:"interpolated"`
	Number            int               `json:"number" yaml:"number"`
	RangedIndex       int               `json:"rangedIndex" yaml:"rangedIndex"`
	RangedEnable      bool              `json:"rangedEnable" yaml:"rangedEnable"`
	InputVars         map[string]string `json:"inputVars" yaml:"-"`
	ComputedVars      H                 `json:"computedVars" yaml:"-"`
	ComputedInfo      []string          `json:"computedInfos" yaml:"-"`
	AssertionsApplied AssertionsApplied `json:"assertionsApplied" yaml:"-"`
	Retries           int               `json:"retries" yaml:"retries"`

	Systemout string    `json:"systemout"`
	Systemerr string    `json:"systemerr"`
	Duration  float64   `json:"duration"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
}

func (ts *TestStepResult) appendError(err error) {
	ts.Errors = append(ts.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
	ts.Status = StatusFail
}

// Append an error to a test step and its associated test case
func (ts *TestStepResult) appendFailure(failure ...Failure) {
	ts.Errors = append(ts.Errors, failure...)
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
	//If string is empty, return an empty slice instead
	if len(out) == 0 {
		return []string{}, nil
	}
	return []string{out}, nil
}

// Range contains data related to iterable user values
type Range struct {
	Enabled    bool
	Items      []RangeData
	RawContent interface{} `json:"range"`
}

// RangeData contains a single iterable user value
type RangeData struct {
	Key   string
	Value interface{}
}

// Skipped contains data related to a skipped test.
type Skipped struct {
	Value string `xml:",cdata" json:"value" yaml:"value,omitempty"`
}

// Failure contains data related to a failed test.
type Failure struct {
	TestcaseClassname  string `xml:"-" json:"-" yaml:"-"`
	TestcaseName       string `xml:"-" json:"-" yaml:"-"`
	TestcaseLineNumber int    `xml:"-" json:"-" yaml:"-"`
	StepNumber         int    `xml:"-" json:"-" yaml:"-"`
	Assertion          string `xml:"-" json:"-" yaml:"-"`
	AssertionRequired  bool   `xml:"-" json:"-" yaml:"-"`
	Error              error  `xml:"-" json:"-" yaml:"-"`

	Value string `json:"value" yaml:"value,omitempty"`
}

type FailureXML struct {
	Value   string `xml:",cdata" json:"value" yaml:"value,omitempty"`
	Type    string `xml:"type,attr,omitempty" json:"type" yaml:"type,omitempty"`
	Message string `xml:"message,attr,omitempty" json:"message" yaml:"message,omitempty"`
}

func newFailure(ctx context.Context, tc TestCase, stepNumber int, rangedIndex int, assertion string, err error) *Failure {
	filename := StringVarFromCtx(ctx, "venom.testsuite.filename")
	var lineNumber = findLineNumber(filename, tc.originalName, stepNumber, assertion, -1)
	var value string
	if assertion != "" {
		value = fmt.Sprintf(`Testcase %q, step #%d-%d: Assertion %q failed. %s (%v:%d)`,
			tc.originalName,
			stepNumber,
			rangedIndex,
			RemoveNotPrintableChar(assertion),
			RemoveNotPrintableChar(err.Error()),
			filename,
			lineNumber,
		)
	} else {
		value = fmt.Sprintf(`Testcase %q, step #%d-%d: %s (%v:%d)`,
			tc.originalName,
			stepNumber,
			rangedIndex,
			RemoveNotPrintableChar(err.Error()),
			filename,
			lineNumber,
		)
	}

	var failure = Failure{
		TestcaseClassname:  filename,
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
		return color.YellowString(f.Value)
	}
	if f.Error != nil {
		return color.YellowString(f.Error.Error())
	}
	return ""
}

// InnerResult is used by TestCase
type InnerResult struct {
	Value string `xml:",cdata" json:"value" yaml:"value"`
}

type AssignStep struct {
	Assignments map[string]Assignment `json:"vars" yaml:"vars" mapstructure:"vars"`
}

type Assignment struct {
	From    string      `json:"from" yaml:"from"`
	Regex   string      `json:"regex" yaml:"regex"`
	Default interface{} `json:"default" yaml:"default"`
}

// RemoveNotPrintableChar removes not printable character from a string
func RemoveNotPrintableChar(in string) string {
	m := func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) || unicode.IsPunct(r) {
			return r
		}
		return ' '
	}
	return strings.Map(m, in)
}

var Red = color.New(color.FgRed).SprintFunc()
var Yellow = color.New(color.FgYellow).SprintFunc()
var Green = color.New(color.FgGreen).SprintFunc()
var Cyan = color.New(color.FgCyan).SprintFunc()
var Gray = color.New(color.Attribute(90)).SprintFunc()
