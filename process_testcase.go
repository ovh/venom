package venom

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/mitchellh/mapstructure"
)

//func (v *Venom) initTestCaseContext(ts *TestSuite, tc *TestCase) (TestCaseContext, error) {
//	var errContext error
//	_, tc.Context, errContext = ts.Templater.ApplyOnMap(tc.Context)
//	if errContext != nil {
//		return nil, errContext
//	}
//	tcc, errContext := v.ContextWrap(tc)
//	if errContext != nil {
//		return nil, errContext
//	}
//	if err := tcc.Init(); err != nil {
//		return nil, err
//	}
//	return tcc, nil
//}
//
//var varRegEx, _ = regexp.Compile("{{.*}}")
//
////Parse the testcase to find unreplaced and extracted variables
//func (v *Venom) parseTestCase(ts *TestSuite, tc *TestCase) ([]string, []string, error) {
//	tcc, err := v.initTestCaseContext(ts, tc)
//	if err != nil {
//		return nil, nil, err
//	}
//	defer tcc.Close()
//
//	vars := []string{}
//	extractedVars := []string{}
//
//	for stepNumber, stepIn := range tc.TestSteps {
//		step, erra := ts.Templater.ApplyOnStep(stepNumber, stepIn)
//		if erra != nil {
//			return nil, nil, erra
//		}
//
//		exec, err := v.WrapExecutor(step, tcc)
//		if err != nil {
//			return nil, nil, err
//		}
//
//		withZero, ok := exec.executor.(executorWithZeroValueResult)
//		if ok {
//			defaultResult := withZero.ZeroValueResult()
//			dumpE, err := dump.ToStringMap(defaultResult, dump.WithDefaultLowerCaseFormatter())
//			if err != nil {
//				return nil, nil, err
//			}
//
//			for k := range dumpE {
//				extractedVars = append(extractedVars, tc.Name+"."+k)
//			}
//		}
//
//		dumpE, err := dump.ToStringMap(step, dump.WithDefaultLowerCaseFormatter())
//		if err != nil {
//			return nil, nil, err
//		}
//
//		for k, v := range dumpE {
//			if strings.HasPrefix(k, "extracts.") {
//				for _, extractVar := range extractPattern.FindAllString(v, -1) {
//					varname := extractVar[2:strings.Index(extractVar, "=")]
//					var found bool
//					for i := 0; i < len(extractedVars); i++ {
//						if extractedVars[i] == varname {
//							found = true
//							break
//						}
//					}
//					if !found {
//						extractedVars = append(extractedVars, tc.Name+"."+varname)
//					}
//				}
//				continue
//			}
//
//			if varRegEx.MatchString(v) {
//				var found bool
//				for i := 0; i < len(vars); i++ {
//					if vars[i] == k {
//						found = true
//						break
//					}
//				}
//
//				for i := 0; i < len(extractedVars); i++ {
//					s := varRegEx.FindString(v)
//					prefix := "{{." + extractedVars[i]
//					if strings.HasPrefix(s, prefix) {
//						found = true
//						break
//					}
//				}
//				if !found {
//					s := varRegEx.FindString(v)
//					s = strings.Replace(s, "{{.", "", -1)
//					s = strings.Replace(s, "}}", "", -1)
//					vars = append(vars, s)
//				}
//			}
//		}
//
//	}
//	return vars, extractedVars, nil
//}
//

func (v *Venom) runTestCases(ctx TestContext, ts *TestSuite, l Logger) {
	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		tc.ShortName = slug(tc.Name)
		log := LoggerWithField(l, "testcase", tc.Name)
		log.Infof("Starting testcase %d: %s [%s]", i+1, tc.ShortName, tc.Name)

		if len(tc.Skipped) == 0 {
			if err := v.runTestCase(ctx, ts, tc, log); err != nil {
				tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
			}
		}

		// Push variables from the testcase in the testsuite
		ts.Vars.AddAll(tc.Vars)

		if len(tc.Failures) > 0 {
			ts.Failures += len(tc.Failures)
		}
		if len(tc.Errors) > 0 {
			ts.Errors += len(tc.Errors)
		}
		if len(tc.Skipped) > 0 {
			ts.Skipped += len(tc.Skipped)
		}

		if v.StopOnFailure && (len(tc.Failures) > 0 || len(tc.Errors) > 0) {
			// break TestSuite
			return
		}
	}
}

func (v *Venom) runTestCase(ctx TestContext, ts *TestSuite, tc *TestCase, l Logger) error {
	displayCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var p = new(Progress)
	go p.Display(displayCtx, v.Display)
	p.testsuite = ts.Name
	p.testcase = tc.Name
	p.teststepTotal = len(tc.TestSteps)
	p.runnnig = true

	start := time.Now()
	defer func() {
		l.Infof("End testcase (%.3f seconds)", time.Since(start).Seconds())
	}()

	if tc.Context != nil {
		modCtx, err := v.getContextModule(tc.Context.Get("type"))
		if err != nil {
			return fmt.Errorf("unable to get context module: %v", err)
		}
		ctx, err = modCtx.New(ctx, ctx.Bag())
		if err != nil {
			return fmt.Errorf("unable to get context: %v", err)
		}
	}

	tc.Vars = ts.Vars.Clone()
	tc.Vars.Add("venom.testcase", tc.ShortName)
	tc.Vars.Add("venom.datetime", time.Now().Format(time.RFC3339))
	tc.Vars.Add("venom.timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	for stepNumber, stepIn := range tc.TestSteps {
		t0 := time.Now()

		p.teststepNumber = stepNumber + 1

		l.Debugf("Processing step #%d", stepNumber)
		if err := stepIn.Interpolate(stepNumber, tc.Vars, l); err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
			break
		}
		l := LoggerWithField(l, "step", fmt.Sprintf("#%-2d", stepNumber))

		assign, isAssign, err := ProcessVariableAssigments(tc.ShortName, tc.Vars, stepIn, l)

		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: RemoveNotPrintableChar(err.Error())})
		}
		tc.Vars.AddAllWithPrefix(tc.ShortName, assign)

		if isAssign {
			if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
				break
			}
			continue
		}

		res, assertRes, err := v.RunTestStep(ctx, tc.Name, stepNumber, stepIn, l)
		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: RemoveNotPrintableChar(err.Error())})
		}

		tc.Vars.AddAllWithPrefix(tc.ShortName, res.H())

		tc.Errors = append(tc.Errors, assertRes.errors...)
		tc.Failures = append(tc.Failures, assertRes.failures...)
		// if retry > 1 && (len(assertRes.failures) > 0 || len(assertRes.errors) > 0) {
		// 	tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
		// }
		tc.Systemout.Value += assertRes.systemout
		tc.Systemerr.Value += assertRes.systemerr

		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			l.Errorf("Testcase %s: errors: %v, failures: %v", colorFailure("FAILURE"), tc.Errors, tc.Failures)
			break
		}
		l.Infof("End step with %s (%.3f seconds)", colorSuccess("SUCCESS"), time.Since(t0).Seconds())
	}

	// Update the progression display before exiting
	p.runnnig = false
	p.success = len(tc.Errors) == 0 && len(tc.Failures) == 0
	// Sleep to let the display being refesh
	time.Sleep(100 * time.Millisecond)

	return nil
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

	for varname, assigment := range stepAssignment.Assignments {
		l.Debugf("Processing %s assignment", varname)
		varValue, has := tcVars[assigment.From]
		if !has {
			varValue, has = tcVars[tcName+"."+assigment.From]
			if !has {
				err := fmt.Errorf("%s reference not found in %v", assigment.From, tcVars)
				l.Errorf("%v", err)
				//tc.Failures = append(tc.Failures, Failure{Value: RemoveNotPrintableChar(err.Error())})
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
			l.Debugf("Assign '%s' from regexp '%v', value '%s'", varname, regex, submatches[1])
			result.Add(varname, submatches[1])
		}
	}
	return result, true, nil
}
