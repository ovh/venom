package venom

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
)

func runTestCase(ts *TestSuite, tc *TestCase, bars map[string]*pb.ProgressBar, aliases map[string]string, l *log.Entry, detailsLevel string) {
	l.Debugf("Init context")
	ctx, errContext := getContextWrap(tc)
	if errContext != nil {
		tc.Errors = append(tc.Errors, Failure{Value: errContext.Error()})
		return
	}

	l = l.WithField("x.testcase", tc.Name)
	l.Infof("start")

	for _, stepIn := range tc.TestSteps {

		step, erra := ts.Templater.Apply(stepIn)
		if erra != nil {
			tc.Errors = append(tc.Errors, Failure{Value: erra.Error()})
			break
		}

		e, err := getExecutorWrap(step)
		if err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: err.Error()})
			break
		}

		runTestStep(ctx, e, ts, tc, step, ts.Templater, aliases, l, detailsLevel)

		if detailsLevel != DetailsLow {
			bars[ts.Package].Increment()
		}
		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			break
		}
	}
	l.Infof("end")
}
