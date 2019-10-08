package venom

import (
	"io"
	"testing"
)

type TestLogger struct {
	t *testing.T
}

var _ io.Writer = TestLogger{}

func (t TestLogger) Write(btes []byte) (int, error) {
	t.t.Logf(string(btes))
	return len(btes), nil
}

func (t TestLogger) Debugf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
func (t TestLogger) Infof(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
func (t TestLogger) Warnf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
func (t TestLogger) Warningf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
func (t TestLogger) Errorf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
func (t TestLogger) Fatalf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
