package venom

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/sdk/interpolate"

	yaml "gopkg.in/yaml.v2"
)

const (
	// DetailsLow prints only summary results
	DetailsLow = "low"
	// DetailsMedium summary with lines in failure
	DetailsMedium = "medium"
	// DetailsHigh all
	DetailsHigh = "high"
)

type H map[string]string

func (h H) Clone() H {
	var h2 = make(H, len(h))
	h2.AddAll(h)
	return h2
}

func (h *H) Add(k, v string) {
	(*h)[k] = v
}

func (h *H) AddWithPrefix(p, k, v string) {
	(*h)[p+"."+k] = v
}

func (h *H) AddAll(h2 H) {
	for k, v := range h2 {
		h.Add(k, v)
	}
}

func (h *H) AddAllWithPrefix(p string, h2 H) {
	for k, v := range h2 {
		h.AddWithPrefix(p, k, v)
	}
}

// Aliases contains list of aliases
type Aliases map[string]string

// ExecutorResult represents an executor result on a test step
type ExecutorResult map[string]interface{}

func (e ExecutorResult) H() H {
	out := make(H, len(e))
	for k, v := range e {
		out.Add(k, fmt.Sprintf("%v", v))
	}
	return out
}

// StepAssertions contains step assertions
type StepAssertions struct {
	Assertions []string `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

// StepExtracts contains "step extracts"
type StepExtracts struct {
	Extracts map[string]string `json:"extracts,omitempty" yaml:"extracts,omitempty"`
}

// Executor execute a testStep.
type Executor interface {
	// Run run a Test Step
	Run(TestContext, TestStep) (ExecutorResult, error)
}

// TestContext represents the context initialized over a test suite or a test case testcase
type TestContext interface {
	context.Context
	Get(string) interface{}
	RunCommand(cmd string, args ...interface{}) error
	WithTimeout(d time.Duration) context.CancelFunc
	WithCancel() context.CancelFunc
	SetWorkingDirectory(string)
	GetWorkingDirectory() string
	Bag() H
}

//func (t TestContext) WithTimeout(d time.Duration) context.CancelFunc {
//	var cancel context.CancelFunc
//	t.context, cancel = context.WithTimeout(d)
//	return cancel
//}

// ExecutorWithDefaultAssertions define default assertions on a Eexcutor
type ExecutorWithDefaultAssertions interface {
	Executor
	// GetDefaultAssertion returns default assertions
	GetDefaultAssertions() *StepAssertions
}

type executorWithZeroValueResult interface {
	ZeroValueResult() ExecutorResult
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
	XMLName    xml.Name     `xml:"testsuite" json:"-" yaml:"-"`
	Disabled   int          `xml:"disabled,attr,omitempty" json:"disabled" yaml:"-"`
	Errors     int          `xml:"errors,attr,omitempty" json:"errors" yaml:"-"`
	Failures   int          `xml:"failures,attr,omitempty" json:"failures" yaml:"-"`
	Hostname   string       `xml:"hostname,attr,omitempty" json:"hostname" yaml:"-"`
	ID         string       `xml:"id,attr,omitempty" json:"id" yaml:"-"`
	Name       string       `xml:"name,attr" json:"name" yaml:"name"`
	Filename   string       `xml:"-" json:"-" yaml:"-"`
	ShortName  string       `xml:"-" json:"-" yaml:"-"`
	Package    string       `xml:"package,attr,omitempty" json:"package" yaml:"-"`
	Properties []Property   `xml:"-" json:"properties" yaml:"-"`
	Skipped    int          `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Total      int          `xml:"tests,attr" json:"total" yaml:"total,omitempty"`
	TestCases  []TestCase   `xml:"testcase" hcl:"testcase" json:"tests" yaml:"testcases"`
	Version    string       `xml:"version,omitempty" hcl:"version" json:"version" yaml:"version,omitempty"`
	Time       string       `xml:"time,attr,omitempty" json:"time" yaml:"-"`
	Timestamp  string       `xml:"timestamp,attr,omitempty" json:"timestamp" yaml:"-"`
	Vars       H            `xml:"-" json:"-" yaml:"vars"`
	WorkDir    string       `xml:"-" json:"-" yaml:"-"`
	Context    *ContextData `xml:"-" json:"-" yaml:"context,omitempty"`
}

type ContextData struct {
	Type string `xml:"-" json:"-" yaml:"type,omitempty"`
	H
}

// Property represents a key/value pair used to define properties.
type Property struct {
	XMLName xml.Name `xml:"property" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr" json:"name" yaml:"-"`
	Value   string   `xml:"value,attr" json:"value" yaml:"-"`
}

// TestCase is a single test case with its result.
type TestCase struct {
	XMLName   xml.Name     `xml:"testcase" json:"-" yaml:"-"`
	Classname string       `xml:"classname,attr,omitempty" json:"classname" yaml:"-"`
	Errors    []Failure    `xml:"error,omitempty" json:"errors" yaml:"errors,omitempty"`
	Failures  []Failure    `xml:"failure,omitempty" json:"failures" yaml:"failures,omitempty"`
	Name      string       `xml:"name,attr" json:"name" yaml:"name"`
	ShortName string       `xml:"-" json:"-shortname" yaml:"shortname"`
	Skipped   []Skipped    `xml:"skipped,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Status    string       `xml:"status,attr,omitempty" json:"status" yaml:"status,omitempty"`
	Systemout InnerResult  `xml:"system-out,omitempty" json:"systemout" yaml:"systemout,omitempty"`
	Systemerr InnerResult  `xml:"system-err,omitempty" json:"systemerr" yaml:"systemerr,omitempty"`
	Time      string       `xml:"time,attr,omitempty" json:"time" yaml:"time,omitempty"`
	TestSteps []TestStep   `xml:"-" hcl:"step" json:"steps" yaml:"steps"`
	Context   *ContextData `xml:"-" json:"-" yaml:"context,omitempty"`
	Vars      H            `xml:"-" json:"-" yaml:"vars"`
}

func (tc TestCase) HasFailureOrError() bool {
	return len(tc.Failures) != 0 || len(tc.Errors) != 0
}

type AssignStep struct {
	Assignments map[string]Assignment `json:"vars" yaml:"vars" mapstructure:"vars"`
}

type Assignment struct {
	From  string `json:"from" yaml:"from"`
	Regex string `json:"regex" yaml:"regex"`
}

// TestStep represents a testStep
type TestStep map[string]interface{}

func (t *TestStep) GetRetry() int {
	return getAttrInt(*t, "retry")
}

func (t *TestStep) GetDelay() int {
	return getAttrInt(*t, "delay")

}

func (t *TestStep) GetTimeout() int {
	return getAttrInt(*t, "timeout")
}

func (t *TestStep) GetType() string {
	return getAttrString(*t, "type")
}

func getAttrInt(t map[string]interface{}, name string) int {
	var out int
	if i, ok := t[name]; ok {
		var ok bool
		out, ok = i.(int)
		if !ok {
			return 0
		}
	}
	if out < 0 {
		out = 0
	}
	return out
}

func getAttrString(t map[string]interface{}, name string) string {
	var out string
	if i, ok := t[name]; ok {
		var ok bool
		out, ok = i.(string)
		if !ok {
			return ""
		}
	}
	return out
}

func (t *TestStep) Interpolate(stepNumber int, vars H, l Logger) error {
	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := s
	// if the testTest use some variable, we run tmpl.apply on it
	if strings.Contains(string(s), "{{") {
		if stepNumber >= 0 {
			vars.Add("venom.teststep.number", strconv.Itoa(stepNumber))
		}
		l.Debugf("Interpolating teststep #%d '%+v'", stepNumber, t)
		btes, err := interpolate.Do(string(sb), vars)
		if err != nil {
			return err
		}
		sb = []byte(btes)
	}

	var newT TestStep
	if err := yaml.Unmarshal([]byte(sb), &newT); err != nil {
		return fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}
	*t = newT
	return nil
}

// Skipped contains data related to a skipped test.
type Skipped struct {
	Value string `xml:",cdata" json:"value" yaml:"value,omitempty"`
}

// Failure contains data related to a failed test.
type Failure struct {
	Value   string         `xml:",cdata" json:"value" yaml:"value,omitempty"`
	Result  ExecutorResult `xml:"-" json:"-" yaml:"-"`
	Type    string         `xml:"type,attr,omitempty" json:"type" yaml:"type,omitempty"`
	Message string         `xml:"message,attr,omitempty" json:"message" yaml:"message,omitempty"`
}

// InnerResult is used by TestCase
type InnerResult struct {
	Value string `xml:",cdata" json:"value" yaml:"value"`
}
