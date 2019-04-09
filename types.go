package venom

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk/interpolate"
	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
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

func (h H) Get(k string) string {
	return (h)[k]
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
	SetWorkingDirectory(string)
	GetWorkingDirectory() string
	Bag() HH
}

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
	XMLName    xml.Name   `xml:"testsuite" json:"-" yaml:"-"`
	Disabled   int        `xml:"disabled,attr,omitempty" json:"disabled" yaml:"-"`
	Errors     int        `xml:"errors,attr,omitempty" json:"errors" yaml:"-"`
	Failures   int        `xml:"failures,attr,omitempty" json:"failures" yaml:"-"`
	Hostname   string     `xml:"hostname,attr,omitempty" json:"hostname" yaml:"-"`
	ID         string     `xml:"id,attr,omitempty" json:"id" yaml:"-"`
	Name       string     `xml:"name,attr" json:"name" yaml:"name"`
	Filename   string     `xml:"-" json:"-" yaml:"-"`
	ShortName  string     `xml:"-" json:"-" yaml:"-"`
	Package    string     `xml:"package,attr,omitempty" json:"package" yaml:"-"`
	Properties []Property `xml:"-" json:"properties" yaml:"-"`
	Skipped    int        `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Total      int        `xml:"tests,attr" json:"total" yaml:"total,omitempty"`
	TestCases  []TestCase `xml:"testcase" hcl:"testcase" json:"tests" yaml:"testcases"`
	Version    string     `xml:"version,omitempty" hcl:"version" json:"version" yaml:"version,omitempty"`
	Time       string     `xml:"time,attr,omitempty" json:"time" yaml:"-"`
	Timestamp  string     `xml:"timestamp,attr,omitempty" json:"timestamp" yaml:"-"`
	Vars       H          `xml:"-" json:"-" yaml:"vars"`
	WorkDir    string     `xml:"-" json:"-" yaml:"-"`
	Context    HH         `xml:"-" json:"-" yaml:"context,omitempty"`
}

// Property represents a key/value pair used to define properties.
type Property struct {
	XMLName xml.Name `xml:"property" json:"-" yaml:"-"`
	Name    string   `xml:"name,attr" json:"name" yaml:"-"`
	Value   string   `xml:"value,attr" json:"value" yaml:"-"`
}

// TestCase is a single test case with its result.
type TestCase struct {
	XMLName   xml.Name    `xml:"testcase" json:"-" yaml:"-"`
	Classname string      `xml:"classname,attr,omitempty" json:"classname" yaml:"-"`
	Errors    []Failure   `xml:"error,omitempty" json:"errors" yaml:"errors,omitempty"`
	Failures  []Failure   `xml:"failure,omitempty" json:"failures" yaml:"failures,omitempty"`
	Name      string      `xml:"name,attr" json:"name" yaml:"name"`
	ShortName string      `xml:"-" json:"-shortname" yaml:"shortname"`
	Skipped   []Skipped   `xml:"skipped,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Status    string      `xml:"status,attr,omitempty" json:"status" yaml:"status,omitempty"`
	Systemout InnerResult `xml:"system-out,omitempty" json:"systemout" yaml:"systemout,omitempty"`
	Systemerr InnerResult `xml:"system-err,omitempty" json:"systemerr" yaml:"systemerr,omitempty"`
	Time      string      `xml:"time,attr,omitempty" json:"time" yaml:"time,omitempty"`
	TestSteps []TestStep  `xml:"-" hcl:"step" json:"steps" yaml:"steps"`
	Context   HH          `xml:"-" json:"-" yaml:"context,omitempty"`
	Vars      H           `xml:"-" json:"-" yaml:"vars"`
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

type HH map[string]interface{}

// Get returns string from default context.
func (h HH) Get(key string) string {
	val, has := h[key]
	if !has {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetString returns string from default context.
func (h HH) GetString(key string) (string, error) {
	if h[key] == nil {
		return "", NotFound(key)
	}
	result, ok := h[key].(string)
	if !ok {
		return "", InvalidArgument(key)
	}
	return result, nil
}

// GetFloat returns float64 from default context.
func (h HH) GetFloat(key string) (float64, error) {
	if h[key] == nil {
		return 0, NotFound(key)
	}
	result, ok := h[key].(float64)
	if !ok {
		return 0, InvalidArgument(key)
	}
	return result, nil
}

// GetInt returns int from default context.
func (h HH) GetInt(key string) (int, error) {
	res, err := h.GetFloat(key)
	if err != nil {
		return 0, err
	}

	return int(res), nil
}

// GetBool returns bool from default context.
func (h HH) GetBool(key string) (bool, error) {
	if h[key] == nil {
		return false, NotFound(key)
	}
	result, ok := h[key].(bool)
	if !ok {
		return false, InvalidArgument(key)
	}
	return result, nil
}

// GetStringSlice returns string slice from default context.
func (h HH) GetStringSlice(key string) ([]string, error) {
	if h[key] == nil {
		return nil, NotFound(key)
	}

	stringSlice, ok := h[key].([]string)
	if ok {
		return stringSlice, nil
	}

	slice, ok := h[key].([]interface{})
	if !ok {
		return nil, InvalidArgument(key)
	}

	res := make([]string, len(slice))

	for k, v := range slice {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("cannot cast to string")
		}

		res[k] = s
	}

	return res, nil
}

// GetComplex unmarshal argument in struct from default context.
func (h HH) GetComplex(key string, arg interface{}) error {
	if h[key] == nil {
		return NotFound(key)
	}

	val, err := yaml.Marshal(h[key])
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(val, arg)
	if err != nil {
		return err
	}
	return nil
}

func (h *HH) Add(k string, v interface{}) {
	(*h)[k] = v
}

// NotFound is error returned when trying to get missing argument
func NotFound(key string) error { return fmt.Errorf("missing context argument '%s'", key) }

// InvalidArgument is error returned when trying to cast argument with wrong type
func InvalidArgument(key string) error { return fmt.Errorf("invalid context argument type '%s'", key) }
