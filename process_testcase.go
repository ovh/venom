package venom

import (
	"context"

	"github.com/sirupsen/logrus"
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

	if _l, ok := l.(*logrus.Entry); ok {
		l = _l.WithField("x.testcase", tc.Name)
	}

	tc.Vars = H{}
	tc.Vars.AddAll(ts.Vars)
	tc.Vars.Add("venom.testcase", tc.Name)

	for stepNumber, stepIn := range tc.TestSteps {
		if err := stepIn.Interpolate(stepNumber, tc.Vars); err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: RemoveNotPrintableChar(err.Error())})
			break
		}

		res, assertRes, err := v.RunTestStep(ctx, tc.Name, stepNumber, stepIn, l)
		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: RemoveNotPrintableChar(err.Error())})
		}

		tc.Vars.AddAllWithPrefix(tc.Name, res.H())

		//TODO check assertions /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
		tc.Errors = append(tc.Errors, assertRes.errors...)
		tc.Failures = append(tc.Failures, assertRes.failures...)
		// if retry > 1 && (len(assertRes.failures) > 0 || len(assertRes.errors) > 0) {
		// 	tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
		// }
		tc.Systemout.Value += assertRes.systemout
		tc.Systemerr.Value += assertRes.systemerr

		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			break
		}
	}
	return nil
}
