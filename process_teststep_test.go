package venom

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestLogger struct {
	t *testing.T
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

func TestRunTestStep(t *testing.T) {
	v := New()
	v.ConfigurationDirectory = "./dist/executors"
	v.LogLevel = LogLevelDebug
	step := TestStep{
		"type":   "http",
		"method": "GET",
		"url":    "https://jsonplaceholder.typicode.com/todos/1",
		"assertions": []string{
			"result.statuscode ShouldEqual 200",
			"result.timeseconds ShouldBeLessThan 1",
		},
	}

	ctxMod, _ := v.getContextModule("")
	ctx, _ := ctxMod.New(context.Background(), nil)

	res, asserts, err := v.RunTestStep(ctx, "test", 0, step, TestLogger{t})
	assert.NoError(t, err)
	assert.NotNil(t, asserts)
	t.Logf("assertions: %+v", asserts)
	assert.NotNil(t, res)
}
