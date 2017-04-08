package venom

import (
	"testing"

	"github.com/runabove/venom"
	"github.com/runabove/venom/context/default"
	"github.com/runabove/venom/context/webctx"
	"github.com/runabove/venom/executors/exec"
	"github.com/runabove/venom/executors/http"
	"github.com/runabove/venom/executors/imap"
	"github.com/runabove/venom/executors/readfile"
	"github.com/runabove/venom/executors/smtp"
	"github.com/runabove/venom/executors/ssh"
	"github.com/runabove/venom/executors/web"
)

func init() {
	venom.RegisterExecutor(exec.Name, exec.New())
	venom.RegisterExecutor(http.Name, http.New())
	venom.RegisterExecutor(imap.Name, imap.New())
	venom.RegisterExecutor(readfile.Name, readfile.New())
	venom.RegisterExecutor(smtp.Name, smtp.New())
	venom.RegisterExecutor(ssh.Name, ssh.New())
	venom.RegisterExecutor(web.Name, web.New())
	venom.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
	venom.RegisterTestCaseContext(webctx.Name, webctx.New())
}

type H map[string]interface{}

type V map[string]string

type T struct {
	*testing.T
	ts   *venom.TestSuite
	tc   *venom.TestCase
	Name string
}

type Logger struct{ t *testing.T }

func (l *Logger) Debugf(format string, args ...interface{})   { l.t.Logf(format, args...) }
func (l *Logger) Infof(format string, args ...interface{})    { l.t.Logf(format, args...) }
func (l *Logger) Printf(format string, args ...interface{})   { l.t.Logf(format, args...) }
func (l *Logger) Warnf(format string, args ...interface{})    { l.t.Logf(format, args...) }
func (l *Logger) Warningf(format string, args ...interface{}) { l.t.Logf(format, args...) }
func (l *Logger) Errorf(format string, args ...interface{})   { l.t.Logf(format, args...) }
func (l *Logger) Fatalf(format string, args ...interface{})   { l.t.Logf(format, args...) }
func (l *Logger) WithField(key string, value interface{}) venom.Logger {
	return l
}

func NewTestCase(t *testing.T, name string, variables map[string]string) *T {
	return &T{
		t,
		&venom.TestSuite{
			Templater: &venom.Templater{Values: variables},
			Name:      name,
		},
		&venom.TestCase{
			Name: name,
		},
		name,
	}
}

func RunTest(t *T, teststep H) V {
	ts := t.ts
	tc := t.tc
	tcc, errContext := venom.ContextWrap(tc)
	if errContext != nil {
		t.Error(errContext)
		return nil
	}
	if err := tcc.Init(); err != nil {
		tc.Errors = append(tc.Errors, venom.Failure{Value: err.Error()})
		t.Error(err)
		return nil
	}
	defer tcc.Close()

	step, erra := ts.Templater.ApplyOnStep(venom.TestStep(teststep))
	if erra != nil {
		t.Error(erra)
		return nil
	}

	e, err := venom.ExecutorWrap(step, tcc)
	if err != nil {
		t.Error(err)
		return nil
	}

	res := venom.RunTestStep(tcc, e, ts, tc, step, ts.Templater, &Logger{t.T}, "low")

	for _, f := range tc.Failures {
		t.Errorf("\r Failure %s", f.Value)
	}

	for _, e := range tc.Errors {
		t.Errorf("\r Error %s", e.Value)
	}

	return V(res)
}
