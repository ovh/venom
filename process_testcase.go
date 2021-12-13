package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/pkg/errors"
)

var varRegEx = regexp.MustCompile("{{.*}}")

//Parse the testcase to find unreplaced and extracted variables
func (v *Venom) parseTestCase(ts *TestSuite, tc *TestCase) ([]string, []string, error) {
	dvars, err := DumpStringPreserveCase(tc.Vars)
	if err != nil {
		return nil, nil, err
	}

	vars := []string{}
	extractedVars := []string{}

	// the value of each var can contains a double-quote -> "
	// if the value is not escaped, it will be used as is, and the json sent to unmarshall will be incorrect.
	// This also avoids injections into the json structure of a step
	for i := range dvars {
		dvars[i] = strings.ReplaceAll(dvars[i], "\"", "\\\"")
	}
	for _, rawStep := range tc.RawTestSteps {
		content, err := interpolate.Do(string(rawStep), dvars)
		if err != nil {
			return nil, nil, err
		}

		var step TestStep
		if err := yaml.Unmarshal([]byte(content), &step); err != nil {
			return nil, nil, errors.Wrapf(err, "unable to unmarshal teststep")
		}

		_, exec, err := v.GetExecutorRunner(context.Background(), step, tc.Vars)
		if err != nil {
			return nil, nil, err
		}

		defaultResult := exec.ZeroValueResult()
		if defaultResult != nil {
			dumpE, err := DumpString(defaultResult)
			if err != nil {
				return nil, nil, err
			}
			for k := range dumpE {
				var found bool
				for i := 0; i < len(vars); i++ {
					if vars[i] == k {
						found = true
						break
					}
				}
				if !found {
					extractedVars = append(extractedVars, k)
				}
				extractedVars = append(extractedVars, tc.Name+"."+k)
				if strings.HasSuffix(k, "__type__") && dumpE[k] == "Map" {
					// go-dump doesnt dump the map name, here is a workaround
					k = strings.TrimSuffix(k, "__type__")
					extractedVars = append(extractedVars, tc.Name+"."+k)
				}
			}
		}

		dumpE, err := DumpStringPreserveCase(step)
		if err != nil {
			return nil, nil, err
		}

		for k, v := range dumpE {
			if strings.HasPrefix(k, "vars.") {
				s := tc.Name + "." + strings.Split(k[5:], ".")[0]
				extractedVars = append(extractedVars, s)
				continue
			}
			if strings.HasPrefix(k, "extracts.") {
				s := tc.Name + "." + strings.Split(k[9:], ".")[0]
				extractedVars = append(extractedVars, s)
				continue
			}
			if strings.HasPrefix(k, "info") {
				continue
			}
			if varRegEx.MatchString(v) {
				var found bool
				for i := 0; i < len(vars); i++ {
					if vars[i] == k {
						found = true
						break
					}
				}

				s := varRegEx.FindString(v)
				for i := 0; i < len(extractedVars); i++ {
					prefix := "{{." + extractedVars[i]
					if strings.HasPrefix(s, prefix) {
						found = true
						break
					}
				}
				if !found {
					s = strings.ReplaceAll(s, "{{.", "")
					s = strings.ReplaceAll(s, "}}", "")
					vars = append(vars, s)
				}
			}
		}
	}
	return vars, extractedVars, nil
}

func (v *Venom) runTestCase(ctx context.Context, ts *TestSuite, tc *TestCase) {
	ctx = context.WithValue(ctx, ContextKey("testcase"), tc.Name)
	tc.TestSuiteVars = ts.Vars.Clone()
	tc.Vars = ts.Vars.Clone()
	tc.Vars.Add("venom.testcase", tc.Name)
	tc.Vars.AddAll(ts.ComputedVars)
	tc.computedVars = H{}

	Info(ctx, "Starting testcase")

	for k, v := range tc.Vars {
		Debug(ctx, "Running testcase with variable %s: %+v", k, v)
	}

	defer Info(ctx, "Ending testcase")
	v.runTestSteps(ctx, tc)
}

func (v *Venom) runTestSteps(ctx context.Context, tc *TestCase) {
	for _, skipAssertion := range tc.Skip {
		Debug(ctx, "evaluating %s", skipAssertion)
		assert, err := parseAssertions(ctx, skipAssertion, tc.Vars)
		if err != nil {
			Error(ctx, "unable to parse skip assertion: %v", err)
			tc.AppendError(err)
			return
		}
		if err := assert.Func(assert.Actual, assert.Args...); err != nil {
			s := fmt.Sprintf("skipping testcase %q: %v", tc.originalName, err)
			tc.Skipped = append(tc.Skipped, Skipped{Value: s})
			Warn(ctx, s)
		}
	}

	if len(tc.Skipped) > 0 {
		return
	}

	var knowExecutors = map[string]struct{}{}

	for stepNumber, rawStep := range tc.RawTestSteps {
		stepVars := tc.Vars.Clone()
		stepVars.AddAllWithPrefix(tc.Name, tc.computedVars)
		stepVars.Add("venom.teststep.number", stepNumber)

		ranged, err := parseRanged(ctx, rawStep, stepVars)
		if err != nil {
			Error(ctx, "unable to parse \"range\" attribute: %v", err)
			tc.AppendError(err)
			return
		}

		for rangedIndex, rangedData := range ranged.Items {
			if ranged.Enabled {
				Debug(ctx, "processing step %d", rangedIndex)
				stepVars.Add("index", rangedIndex)
				stepVars.Add("key", rangedData.Key)
				stepVars.Add("value", rangedData.Value)
			}

			vars, err := DumpStringPreserveCase(stepVars)
			if err != nil {
				Error(ctx, "unable to dump testcase vars: %v", err)
				tc.AppendError(err)
				return
			}

			for k, v := range vars {
				content, err := interpolate.Do(v, vars)
				if err != nil {
					tc.AppendError(err)
					Error(ctx, "unable to interpolate variable %q: %v", v, err)
					return
				}
				vars[k] = content
			}

			// the value of each var can contains a double-quote -> "
			// if the value is not escaped, it will be used as is, and the json sent to unmarshall will be incorrect.
			// This also avoids injections into the json structure of a step
			for i := range vars {
				vars[i] = strings.ReplaceAll(vars[i], "\"", "\\\"")
			}

			var content string
			for i := 0; i < 10; i++ {
				content, err = interpolate.Do(string(rawStep), vars)
				if err != nil {
					tc.AppendError(err)
					Error(ctx, "unable to interpolate step: %v", err)
					return
				}
				if !strings.Contains(content, "{{") {
					break
				}
			}

			Info(ctx, "Step #%d content is: %q", stepNumber, content)
			var step TestStep
			if err := yaml.Unmarshal([]byte(content), &step); err != nil {
				tc.AppendError(err)
				Error(ctx, "unable to unmarshal step: %v", err)
				return
			}

			tc.testSteps = append(tc.testSteps, step)
			var e ExecutorRunner
			ctx, e, err = v.GetExecutorRunner(ctx, step, tc.Vars)
			if err != nil {
				tc.AppendError(err)
				Error(ctx, "unable to get executor: %v", err)
				break
			}

			_, known := knowExecutors[e.Name()]
			if !known {
				knowExecutors[e.Name()] = struct{}{}
				ctx, err = e.Setup(ctx, tc.Vars)
				if err != nil {
					tc.AppendError(err)
					Error(ctx, "unable to setup executor: %v", err)
					break
				}
				defer func(ctx context.Context) {
					if err := e.TearDown(ctx); err != nil {
						tc.AppendError(err)
						Error(ctx, "unable to teardown executor: %v", err)
					}
				}(ctx)
			}

			v.RunTestStep(ctx, e, tc, stepNumber, step)

			tc.testSteps = append(tc.testSteps, step)

			var hasFailed bool
			var isRequired bool
			if len(tc.Failures) > 0 {
				for _, f := range tc.Failures {
					Warning(ctx, "%v", f)
					isRequired = isRequired || f.AssertionRequired
				}
				hasFailed = true
			}

			if len(tc.Errors) > 0 {
				Error(ctx, "Errors: ")
				for _, e := range tc.Errors {
					Error(ctx, "%v", e)
				}
				hasFailed = true
			}

			if hasFailed {
				if isRequired {
					failure := newFailure(*tc, stepNumber, "", fmt.Errorf("At least one required assertion failed, skipping remaining steps"))
					tc.Failures = append(tc.Failures, *failure)
					return
				}
				break
			}

			allVars := tc.Vars.Clone()
			allVars.AddAll(tc.computedVars.Clone())

			assign, _, err := processVariableAssigments(ctx, tc.Name, allVars, rawStep)
			if err != nil {
				tc.AppendError(err)
				Error(ctx, "unable to process variable assignments: %v", err)
				break
			}

			tc.computedVars.AddAll(assign)
			tc.Vars.AddAll(tc.computedVars)
		}
	}
}

//Parse and format range data to allow iterations over user data
func parseRanged(ctx context.Context, rawStep []byte, stepVars H) (Range, error) {

	//Load "range" attribute and perform actions depending on its typing
	var ranged Range
	if err := json.Unmarshal(rawStep, &ranged); err != nil {
		return ranged, fmt.Errorf("unable to parse range expression: %v", err)
	}

	switch ranged.RawContent.(type) {

	//Nil means this is not a ranged data, append an empty item to force at least one iteration and exit
	case nil:
		ranged.Items = append(ranged.Items, RangeData{})
		return ranged, nil

	//String needs to be parsed and possibly templated
	case string:
		Debug(ctx, "attempting to parse range expression")
		rawString := ranged.RawContent.(string)
		if len(rawString) == 0 {
			return ranged, fmt.Errorf("range expression has been specified without any data")
		}

		// Try parsing already templated data
		err := json.Unmarshal([]byte("{\"range\":"+rawString+"}"), &ranged)
		// ... or fallback
		if err != nil {
			//Try templating and escaping data
			Debug(ctx, "attempting to template range expression and parse it again")
			vars, err := DumpStringPreserveCase(stepVars)
			if err != nil {
				Warn(ctx, "failed to parse range expression when loading step variables: %v", err)
				break
			}
			for i := range vars {
				vars[i] = strings.ReplaceAll(vars[i], "\"", "\\\"")
			}
			content, err := interpolate.Do(string(rawStep), vars)
			if err != nil {
				Warn(ctx, "failed to parse range expression when templating variables: %v", err)
				break
			}

			//Try parsing data
			err = json.Unmarshal([]byte(content), &ranged)
			if err != nil {
				Warn(ctx, "failed to parse range expression when parsing data into raw string: %v", err)
				break
			}
			switch ranged.RawContent.(type) {
			case string:
				rawString = ranged.RawContent.(string)
				err := json.Unmarshal([]byte("{\"range\":"+rawString+"}"), &ranged)
				if err != nil {
					Warn(ctx, "failed to parse range expression when parsing raw string into data: %v", err)
					return ranged, fmt.Errorf("unable to parse range expression: unable to transform string data into a supported range expression type")
				}
			}
		}
	}

	//Format data
	switch t := ranged.RawContent.(type) {

	//Array-like data
	case []interface{}:
		Debug(ctx, "\"range\" data is array-like")
		for index, value := range ranged.RawContent.([]interface{}) {
			key := strconv.Itoa(index)
			ranged.Items = append(ranged.Items, RangeData{key, value})
		}

	//Number data
	case float64:
		Debug(ctx, "\"range\" data is number-like")
		upperBound := int(ranged.RawContent.(float64))
		for i := 0; i < upperBound; i++ {
			key := strconv.Itoa(i)
			ranged.Items = append(ranged.Items, RangeData{key, i})
		}

	//Map-like data
	case map[string]interface{}:
		Debug(ctx, "\"range\" data is map-like")
		for key, value := range ranged.RawContent.(map[string]interface{}) {
			ranged.Items = append(ranged.Items, RangeData{key, value})
		}

	//Unsupported data format
	default:
		return ranged, fmt.Errorf("\"range\" was provided an unsupported type %T", t)
	}

	ranged.Enabled = true
	ranged.RawContent = nil
	return ranged, nil
}

func processVariableAssigments(ctx context.Context, tcName string, tcVars H, rawStep json.RawMessage) (H, bool, error) {
	var stepAssignment AssignStep
	var result = make(H)
	if err := yaml.Unmarshal(rawStep, &stepAssignment); err != nil {
		Error(ctx, "unable to parse assignements (%s): %v", string(rawStep), err)
		return nil, false, err
	}

	if len(stepAssignment.Assignments) == 0 {
		return nil, false, nil
	}

	var tcVarsKeys []string
	for k := range tcVars {
		tcVarsKeys = append(tcVarsKeys, k)
	}

	for varname, assigment := range stepAssignment.Assignments {
		Debug(ctx, "Processing %s assignment", varname)
		varValue, has := tcVars[assigment.From]
		if !has {
			varValue, has = tcVars[tcName+"."+assigment.From]
			if !has {
				err := fmt.Errorf("%s reference not found in %s", assigment.From, strings.Join(tcVarsKeys, "\n"))
				Info(ctx, "%v", err)
				return nil, true, err
			}
		}
		if assigment.Regex == "" {
			Info(ctx, "Assign '%s' value '%s'", varname, varValue)
			result.Add(varname, varValue)
		} else {
			regex, err := regexp.Compile(assigment.Regex)
			if err != nil {
				Warn(ctx, "unable to compile regexp %q", assigment.Regex)
				return nil, true, err
			}
			varValueS, ok := varValue.(string)
			if !ok {
				Warn(ctx, "%q is not a string value", varname)
				result.Add(varname, "")
				continue
			}
			submatches := regex.FindStringSubmatch(varValueS)
			if len(submatches) == 0 {
				Warn(ctx, "%s: %q doesn't match anything in %q", varname, regex, varValue)
				result.Add(varname, "")
				continue
			}
			Info(ctx, "Assign %q from regexp %q, values %q", varname, regex, submatches)
			result.Add(varname, submatches[len(submatches)-1])
		}
	}
	return result, true, nil
}
