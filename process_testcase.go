package venom

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

func (v *Venom) initTestCaseContext(ts *TestSuite, tc *TestCase) (TestCaseContext, error) {
	var errContext error
	_, tc.Context, errContext = ts.Templater.ApplyOnMap(tc.Context)
	if errContext != nil {
		return nil, errContext
	}
	tcc, errContext := v.ContextWrap(tc, ts)
	if errContext != nil {
		return nil, errContext
	}
	if err := tcc.Init(); err != nil {
		return nil, err
	}
	return tcc, nil
}

var varRegEx, _ = regexp.Compile("{{.*}}")

//Parse the testcase to find unreplaced and extracted variables
func (v *Venom) parseTestCase(ts *TestSuite, tc *TestCase) ([]string, []string, error) {
	tcc, err := v.initTestCaseContext(ts, tc)
	if err != nil {
		return nil, nil, err
	}
	defer tcc.Close()

	vars := []string{}
	extractedVars := []string{}

	for stepNumber, stepIn := range tc.TestSteps {
		step, erra := ts.Templater.ApplyOnStep(stepNumber, stepIn)
		if erra != nil {
			return nil, nil, erra
		}

		exec, err := v.WrapExecutor(step, tcc)
		if err != nil {
			return nil, nil, err
		}

		withZero, ok := exec.executor.(executorWithZeroValueResult)
		if ok {
			defaultResult := withZero.ZeroValueResult()
			dumpE, err := dump.ToStringMap(defaultResult, dump.WithDefaultLowerCaseFormatter())
			if err != nil {
				return nil, nil, err
			}

			for k := range dumpE {
				extractedVars = append(extractedVars, tc.Name+"."+k)
				if strings.HasSuffix(k, "__type__") && dumpE[k] == "Map" {
					// go-dump doesnt dump the map name, here is a workaround
					k = strings.TrimSuffix(k, "__type__")
					extractedVars = append(extractedVars, tc.Name+"."+k)
				}
			}
		}

		dumpE, err := dump.ToStringMap(step, dump.WithDefaultLowerCaseFormatter())
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
				for _, extractVar := range extractPattern.FindAllString(v, -1) {
					varname := extractVar[2:strings.Index(extractVar, "=")]
					var found bool
					for i := 0; i < len(extractedVars); i++ {
						if extractedVars[i] == varname {
							found = true
							break
						}
					}
					if !found {
						extractedVars = append(extractedVars, tc.Name+"."+varname)
					}
				}
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

				if strings.HasPrefix(s, "{{expandEnv ") {
					continue
				}

				for i := 0; i < len(extractedVars); i++ {
					prefix := "{{." + extractedVars[i]
					if strings.HasPrefix(s, prefix) {
						found = true
						break
					}
				}
				if !found {
					s = strings.Replace(s, "{{.", "", -1)
					s = strings.Replace(s, "}}", "", -1)
					vars = append(vars, s)
				}
			}
		}

	}
	return vars, extractedVars, nil
}

func (v *Venom) runTestCase(ts *TestSuite, tc *TestCase, l Logger) {
	tcc, err := v.initTestCaseContext(ts, tc)
	if err != nil {
		tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
		return
	}
	defer tcc.Close()

	if _l, ok := l.(*logrus.Entry); ok {
		l = _l.WithField("x.testcase", tc.Name)
	}

	ts.Templater.Add("", map[string]string{"venom.testcase": tc.Name})
	for stepNumber, stepIn := range tc.TestSteps {
		step, erra := ts.Templater.ApplyOnStep(stepNumber, stepIn)
		if erra != nil {
			tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(erra.Error())})
			break
		}

		e, err := v.WrapExecutor(step, tcc)
		if err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
			break
		}

		v.RunTestStep(tcc, e, ts, tc, stepNumber, step, l)

		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			break
		}

		assign, _, err := ProcessVariableAssigments(tc.Name, ts.Templater.Values, stepIn, l)
		if err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
			break
		}

		ts.Templater.Add(tc.Name, assign)
	}
}

func ProcessVariableAssigments(tcName string, tcVars H, stepIn TestStep, l Logger) (H, bool, error) {
	var stepAssignment AssignStep
	var result = make(H)
	if err := mapstructure.Decode(stepIn, &stepAssignment); err != nil {
		return nil, false, nil
	}

	if len(stepAssignment.Assignments) == 0 {
		return nil, false, nil
	}

	var tcVarsKeys []string
	for k := range tcVars {
		tcVarsKeys = append(tcVarsKeys, k)
	}

	for varname, assigment := range stepAssignment.Assignments {
		l.Debugf("Processing %s assignment", varname)
		varValue, has := tcVars[assigment.From]
		if !has {
			varValue, has = tcVars[tcName+"."+assigment.From]
			if !has {
				err := fmt.Errorf("%s reference not found in %s", assigment.From, strings.Join(tcVarsKeys, "\n"))
				l.Errorf("%v", err)
				return nil, true, err
			}
		}
		if assigment.Regex == "" {
			l.Debugf("Assign '%s' value '%s'", varname, varValue)
			result.Add(varname, varValue)
		} else {
			regex, err := regexp.Compile(assigment.Regex)
			if err != nil {
				l.Errorf("unable to compile regexp %s", assigment.Regex)
				return nil, true, err
			}
			submatches := regex.FindStringSubmatch(varValue)
			if len(submatches) == 0 {
				l.Debugf("%s: '%v' doesn't match anything in '%s'", varname, regex, varValue)
				result.Add(varname, "")
				continue
			}
			l.Debugf("Assign '%s' from regexp '%v', values '%v'", varname, regex, submatches)
			result.Add(varname, submatches[len(submatches)-1])
		}
	}
	return result, true, nil
}
