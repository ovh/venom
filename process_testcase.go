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
func (v *Venom) runTestCase(ts *TestSuite, tc *TestCase, l Logger) error {
	//tcc, err := v.initTestCaseContext(ts, tc)
	//if err != nil {
	//	tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
	//	return
	//}
	//defer tcc.Close()
	start := time.Now()
	defer func() {
		l.Debugf("End runTestCase (%.3f seconds)", time.Since(start).Seconds())
	}()

	if tc.Context == nil {
		tc.Context = &ContextData{Type: "default"}
	}

	ctxMod, err := v.getContextModule(tc.Context.Type)
	if err != nil {
		return err
	}

	ctx, err := ctxMod.New(context.Background(), tc.Context.TestContextValues)
	if err != nil {
		return err
	}
	ctx.SetWorkingDirectory(ts.WorkDir)

	tc.Vars = ts.Vars.Clone()
	tc.Vars.Add("venom.testcase", tc.Name)

	for stepNumber, stepIn := range tc.TestSteps {
		l.Debugf("processing step %d", stepNumber)
		if err := stepIn.Interpolate(stepNumber, tc.Vars); err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
			break
		}
		l := LoggerWithField(l, "step", fmt.Sprintf("#%-2d", stepNumber))

		assign, isAssign, err := ProcessVariableAssigments(tc.Name, tc.Vars, stepIn, l)
		l.Debugf("is an assignment step ? %v", isAssign)

		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: RemoveNotPrintableChar(err.Error())})
		}
		tc.Vars.AddAllWithPrefix(tc.Name, assign)

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

		tc.Vars.AddAllWithPrefix(tc.Name, res.H())

		tc.Errors = append(tc.Errors, assertRes.errors...)
		tc.Failures = append(tc.Failures, assertRes.failures...)
		// if retry > 1 && (len(assertRes.failures) > 0 || len(assertRes.errors) > 0) {
		// 	tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
		// }
		tc.Systemout.Value += assertRes.systemout
		tc.Systemerr.Value += assertRes.systemerr

		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			l.Warnf("testcase failure: errors: %v, failures: %v", tc.Errors, tc.Failures)
			break
		}
		l.Infof("step is a success")
	}
	return nil
}

func ProcessVariableAssigments(tcName string, tcVars H, stepIn TestStep, l Logger) (H, bool, error) {
	var stepAssignment AssignStep
	var result = make(H)
	if err := mapstructure.Decode(stepIn, &stepAssignment); err != nil {
		l.Debugf("step is not a variables assignment step (%v)", err)
		return nil, false, nil
	}

	if len(stepAssignment.Assignments) == 0 {
		return nil, false, nil
	}

	for varname, assigment := range stepAssignment.Assignments {
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
			l.Debugf("assign '%s' value '%s'", varname, varValue)
			result.Add(varname, varValue)
		} else {
			regex, err := regexp.Compile(assigment.Regex)
			if err != nil {
				return nil, true, err
			}
			submatches := regex.FindStringSubmatch(varValue)
			if len(submatches) == 0 {
				result.Add(varname, "")
				continue
			}
			l.Debugf("assign '%s' value '%s'", varname, submatches[1])
			result.Add(varname, submatches[1])
		}
	}
	return result, true, nil
}
