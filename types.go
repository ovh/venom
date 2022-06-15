package venom

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/cast"
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

// Tests contains all informations about tests in a pipeline build
type Tests struct {
	XMLName      xml.Name    `xml:"testsuites" json:"-" yaml:"-"`
	Total        int         `xml:"-" json:"total"`
	TotalOK      int         `xml:"-" json:"ok"`      // contains the number of testcases OK
	TotalKO      int         `xml:"-" json:"ko"`      // contains the number of testcases KO
	TotalSkipped int         `xml:"-" json:"skipped"` // contains the number of testcases skipped
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
	Package      string     `xml:"package,attr,omitempty" json:"package" yaml:"-"`
	Properties   []Property `xml:"-" json:"properties" yaml:"-"`
	Skipped      int        `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Total        int        `xml:"tests,attr" json:"total" yaml:"total,omitempty"`
	TestCases    []TestCase `xml:"testcase" json:"testcases" yaml:"testcases"`
	Version      string     `xml:"version,omitempty" json:"version" yaml:"version,omitempty"`
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
	Time            float64           `xml:"time,attr,omitempty" json:"time" yaml:"time,omitempty"`
	RawTestSteps    []json.RawMessage `xml:"-" json:"steps" yaml:"steps"`
	testSteps       []TestStep
	TestSuiteVars   H `xml:"-" json:"-" yaml:"-"`
	Vars            H `xml:"-" json:"-" yaml:"vars"`
	computedVars    H
	computedInfo    []string
	computedVerbose []string
	Skip            []string `xml:"-" json:"skip" yaml:"skip"`
	IsExecutor      bool     `xml:"-" json:"-" yaml:"-"`
	IsEvaluated     bool     `xml:"-" json:"-" yaml:"-"`
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
	AssertionRequired  bool   `xml:"-" json:"-" yaml:"-"`
	Error              error  `xml:"-" json:"-" yaml:"-"`

	Value   string `xml:",cdata" json:"value" yaml:"value,omitempty"`
	Type    string `xml:"type,attr,omitempty" json:"type" yaml:"type,omitempty"`
	Message string `xml:"message,attr,omitempty" json:"message" yaml:"message,omitempty"`
}

func newFailure(tc TestCase, stepNumber int, assertion string, err error) *Failure {
	var lineNumber = findLineNumber(tc.Classname, tc.originalName, stepNumber, assertion, -1)
	var value string
	if assertion != "" {
		value = fmt.Sprintf(`Testcase %q, step #%d: Assertion %q failed. %s (%v:%d)`,
			tc.originalName,
			stepNumber,
			RemoveNotPrintableChar(assertion),
			RemoveNotPrintableChar(err.Error()),
			tc.Classname,
			lineNumber,
		)
	} else {
		value = fmt.Sprintf(`Testcase %q, step #%d: %s (%v:%d)`,
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
		return color.YellowString(f.Value)
	}
	if f.Error != nil {
		return color.YellowString(f.Error.Error())
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
	From    string      `json:"from" yaml:"from"`
	Regex   string      `json:"regex" yaml:"regex"`
	Default interface{} `json:"default" yaml:"default"`
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
