package venom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/cucumber/gherkin-go/v19"
	"github.com/cucumber/messages-go/v16"
	"github.com/fatih/structs"
	"github.com/ovh/venom/assertions"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// ParseGherkin parses tests suite to check context and variables
func (v *GherkinVenom) ParseGherkin(ctx context.Context, path []string) error {
	filesPath, err := getFilesPath(path, ".feature")
	if err != nil {
		return err
	}

	if err := v.readGherkinFiles(ctx, filesPath); err != nil {
		return err
	}

	for _, gFeature := range v.Features {
		if err := v.registerAllUserExecutorsFromDir(ctx); err != nil {
			Warn(ctx, "%v", err)
		}
		testsuite, err := v.HandleGherkinFeature(gFeature)
		if err != nil {
			return fmt.Errorf("unable to parse feature %q (%q): %v", gFeature.Text, gFeature.Filename, err)
		}
		testsuite.Filename = gFeature.Filename
		v.Testsuites = append(v.Testsuites, *testsuite)
	}

	return nil
}

func (v *GherkinVenom) readGherkinFiles(ctx context.Context, filesPath []string) (err error) {
	var idx int64
	for _, f := range filesPath {
		Info(ctx, "Reading %s", f)
		rawGherkinDoc, err := parseGherkin(f, autoIncrement(&idx))
		if err != nil {
			return err
		}
		feature := v.parseGherkinFeature(ctx, rawGherkinDoc.Feature)
		feature.Filename = f
		v.Features = append(v.Features, feature)
	}
	return nil
}

func (v *GherkinVenom) parseGherkinFeature(ctx context.Context, feature *messages.Feature) GherkinFeature {
	gf := GherkinFeature{
		Text: feature.Name,
	}
	for _, child := range feature.Children {
		gs := v.parseGherkinFeatureScenario(ctx, child.Scenario)
		gf.Scenarios = append(gf.Scenarios, gs)
	}
	return gf
}

func (v *GherkinVenom) parseGherkinFeatureScenario(ctx context.Context, scenario *messages.Scenario) GherkinScenario {
	gscenario := GherkinScenario{
		Text: strings.TrimSpace(scenario.Name),
	}
	for _, step := range scenario.Steps {
		gs := v.parseGherkinFeatureScenarioStep(ctx, step)
		gscenario.Steps = append(gscenario.Steps, gs)
	}
	return gscenario
}

func (v *GherkinVenom) parseGherkinFeatureScenarioStep(ctx context.Context, step *messages.Step) GherkinStep {
	return GherkinStep{
		Keywork: strings.TrimSpace(step.Keyword),
		Text:    strings.TrimSpace(step.Text),
	}
}

func autoIncrement(sequence *int64) func() string {
	return func() string {
		i := atomic.AddInt64(sequence, 1)
		return strconv.FormatInt(i, 10)
	}
}

func parseGherkin(path string, autoIncrementFunc func() string) (*rawGherkinFeature, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %q: %v", path, err)
	}

	defer reader.Close()
	var buf bytes.Buffer
	gherkinDocument, err := gherkin.ParseGherkinDocument(io.TeeReader(reader, &buf), autoIncrementFunc)
	if err != nil {
		return nil, err
	}

	gherkinDocument.Uri = path
	pickles := gherkin.Pickles(*gherkinDocument, path, autoIncrementFunc)

	f := rawGherkinFeature{GherkinDocument: gherkinDocument, Pickles: pickles, Content: buf.Bytes()}
	return &f, nil
}

func (v *GherkinVenom) HandleGherkinFeature(gFeature GherkinFeature) (*TestSuite, error) {
	var testsuite = new(TestSuite)
	testsuite.Name = gFeature.Text
	for _, gScenario := range gFeature.Scenarios {
		s, err := v.HandleGherkinScenario(gScenario)
		if err != nil {
			return nil, err
		}
		testsuite.TestCases = append(testsuite.TestCases, *s)
	}
	return testsuite, nil
}

func (v *GherkinVenom) HandleGherkinScenario(gScenario GherkinScenario) (*TestCase, error) {
	var testcase = new(TestCase)
	testcase.Name = gScenario.Text
	var currentStep *TestStep
	var currentAssertions []Assertion
	for _, gStep := range gScenario.Steps {
		// Check if the step is a real step or an assertion on the previous step
		if currentStep == nil {
			step, err := v.FindSuitableExecutor(gStep)
			if err != nil {
				return nil, err
			}
			currentStep = &step
			continue
		}
		step, err := v.FindSuitableExecutor(gStep)
		if err != nil && err != ErrNoExecutor {
			return nil, err
		} else if err == nil {
			if len(currentAssertions) > 0 {
				(*currentStep)["assertions"] = currentAssertions
			}
			testcase.testSteps = append(testcase.testSteps, *currentStep)
			btes, _ := json.Marshal(currentStep)
			testcase.RawTestSteps = append(testcase.RawTestSteps, json.RawMessage(btes))
			currentStep = &step
			currentAssertions = nil
			continue
		} else {
			a, err := v.TransformGherkinStepToAssertion(gStep)
			if err != nil {
				return nil, err
			}
			currentAssertions = append(currentAssertions, a)
		}
	}
	if currentStep != nil {
		if len(currentAssertions) > 0 {
			(*currentStep)["assertions"] = currentAssertions
		}
		testcase.testSteps = append(testcase.testSteps, *currentStep)
		btes, _ := json.Marshal(currentStep)
		testcase.RawTestSteps = append(testcase.RawTestSteps, json.RawMessage(btes))
	}

	return testcase, nil
}

var ErrNoExecutor = fmt.Errorf("unable to find suitable executor")

func (v GherkinVenom) FindSuitableExecutor(gStep GherkinStep) (TestStep, error) {
	switch gStep.Keywork {
	case "Given", "When", "Then", "And", "But", "*":
	default:
		return nil, fmt.Errorf("unsupported keyword %q", gStep.Keywork)
	}

	// parse buitin executors
	for name, executor := range v.executorsBuiltin {
		gherkingExecutor, is := executor.(ExecutorWithGherkinSupport)
		if is {
			for _, regxp := range gherkingExecutor.GherkinRegExpr() {
				if regxp == nil {
					continue
				}
				if regxp.MatchString(gStep.Text) {
					return v.TransformGherkinStepToExecutor(gStep, name, gherkingExecutor, regxp)
				}
			}
		}
	}

	for name, executor := range v.executorsUser {
		gherkingExecutor, is := executor.(ExecutorWithGherkinSupport)
		if is {
			for _, regxp := range gherkingExecutor.GherkinRegExpr() {
				if regxp == nil {
					continue
				}
				if regxp.MatchString(gStep.Text) {
					return v.TransformGherkinStepToExecutor(gStep, name, gherkingExecutor, regxp)
				}
			}
		}
	}

	return nil, ErrNoExecutor
}

func (v GherkinVenom) TransformGherkinStepToExecutor(gStep GherkinStep, name string, gherkinExecutor ExecutorWithGherkinSupport, regxp *regexp.Regexp) (TestStep, error) {
	var res = TestStep{}

	matches := regxp.FindStringSubmatch(gStep.Text)
	matchNames := regxp.SubexpNames()

	paramsMap := make(map[string]string)
	for i, name := range matchNames {
		if i > 0 && i <= len(matches) {
			if matches[i] == "" {
				continue
			}
			paramsMap[name] = matches[i]
		}
	}

	var executorMap map[string]interface{}

	gherkinUserExecutor, is := gherkinExecutor.(UserExecutor)
	if is {
		executorMap = v.TransformUserExecutorToMap(gherkinUserExecutor)
	} else {
		executorMap = v.TransformExecutorToMap(gherkinExecutor)
	}

	for key, val := range executorMap {
		param, has := paramsMap[key]
		if !has {
			continue
		}
		valParam, err := CastStringParamTo(param, &val)
		if err != nil {
			return nil, err
		}

		res[key] = valParam
	}

	if x, has := paramsMap["retry"]; has {
		res["retry"] = cast.ToInt(x)
	}

	if x, has := paramsMap["delay"]; has {
		res["delay"] = cast.ToInt(x)
	}

	if x, has := paramsMap["timeout"]; has {
		res["timeout"] = cast.ToInt(x)
	}

	res["type"] = name

	return res, nil
}

func CastStringParamTo(param string, target *interface{}) (interface{}, error) {
	if param == "" {
		return nil, nil
	}

	switch (*target).(type) {
	case bool:
		return cast.ToBoolE(param)
	case *bool:
		x, err := cast.ToBoolE(param)
		return &x, err
	case float32:
		return cast.ToFloat32E(param)
	case *float32:
		x, err := cast.ToBoolE(param)
		return &x, err
	case float64:
		return cast.ToFloat64E(param)
	case *float64:
		x, err := cast.ToFloat64E(param)
		return &x, err
	case int:
		return cast.ToIntE(param)
	case *int:
		x, err := cast.ToIntE(param)
		return &x, err
	case int16:
		return cast.ToInt16E(param)
	case *int16:
		x, err := cast.ToInt16E(param)
		return &x, err
	case int32:
		return cast.ToInt32E(param)
	case *int32:
		x, err := cast.ToInt32E(param)
		return &x, err
	case int64:
		return cast.ToInt64E(param)
	case *int64:
		x, err := cast.ToInt64E(param)
		return &x, err
	case int8:
		return cast.ToInt8E(param)
	case *int8:
		x, err := cast.ToInt8E(param)
		return &x, err
	case string:
		return strings.TrimSpace(param), nil
	case *string:
		s := strings.TrimSpace(param)
		return &s, nil
	case uint:
		return cast.ToUintE(param)
	case *uint:
		x, err := cast.ToUintE(param)
		return &x, err
	case uint16:
		return cast.ToUint16E(param)
	case *uint16:
		x, err := cast.ToUint16E(param)
		return &x, err
	case uint32:
		return cast.ToUint32E(param)
	case *uint32:
		x, err := cast.ToUint32E(param)
		return &x, err
	case uint64:
		return cast.ToUint64E(param)
	case *uint64:
		x, err := cast.ToUint64E(param)
		return &x, err
	case uint8:
		return cast.ToUint8E(param)
	case *uint8:
		x, err := cast.ToUint8E(param)
		return &x, err
	}

	if reflect.TypeOf(*target).Kind() == reflect.Array {
		var slice []interface{}
		if err := json.Unmarshal([]byte(param), &slice); err != nil {
			return nil, fmt.Errorf("unable to JSON unmarshal %q: %v", param, err)
		}
		return slice, nil
	}

	if reflect.TypeOf(*target).Kind() == reflect.Map {
		var m interface{}
		if err := json.Unmarshal([]byte(param), &m); err != nil {
			return nil, fmt.Errorf("unable to JSON unmarshal %q: %v", param, err)
		}
		return m, nil
	}

	return nil, fmt.Errorf("unable to cast %q to %T", param, *target)
}

func (v GherkinVenom) TransformUserExecutorToMap(userExecutor UserExecutor) map[string]interface{} {
	return userExecutor.Input
}

func (v GherkinVenom) TransformExecutorToMap(gherkinExecutor ExecutorWithGherkinSupport) map[string]interface{} {
	var concreteExecutor, is = gherkinExecutor.(Executor)
	if !is {
		panic("wrong gherkin executor implementation")
	}
	var iExecutor Executor = concreteExecutor
	var val reflect.Value
	if reflect.TypeOf(iExecutor).Kind() == reflect.Ptr {
		val = reflect.ValueOf(iExecutor).Elem()
	} else {
		val = reflect.ValueOf(iExecutor)
	}
	newConcreteExecutor := reflect.New(val.Type()).Interface()
	iNewConcreteExecutor := newConcreteExecutor.(Executor)

	s := structs.New(iNewConcreteExecutor)
	m := s.Map()
	m1 := make(map[string]interface{}, len(m))

	for name, _ := range m {
		field := s.Field(name)
		tag := field.Tag("json")
		m1[tag] = m[name]
	}

	return m1
}

func (v GherkinVenom) TransformGherkinStepToAssertion(gStep GherkinStep) (Assertion, error) {
	switch gStep.Keywork {
	case "Then", "And", "But", "*":
	default:
		return nil, fmt.Errorf("unsupported keyword %q", gStep.Keywork)
	}

	var a Assertion
	assert := splitAssertion(strings.TrimSpace(gStep.Text))
	if len(assert) < 2 {
		return nil, errors.New("assertion syntax error")
	}

	actualRef := assert[0]
	assertFunc := assert[1]
	var optionnalExpectations []string
	if len(assert) > 2 {
		optionnalExpectations = assert[2:]
	}

	var fName = assert[1]
	if strings.HasPrefix(fName, "Must") {
		fName = strings.Replace(fName, "Must", "Should", 1)
	}
	_, ok := assertions.Get(fName)
	if !ok {
		return nil, fmt.Errorf("unknown assertion operator %q", assert[1])
	}

	as := actualRef + " " + assertFunc + " "
	as += strings.Join(optionnalExpectations, " ")
	a = strings.TrimSpace(as)
	return a, nil
}
