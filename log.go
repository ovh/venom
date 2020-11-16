package venom

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func initTestLogger(t *testing.T) {
	l := logrus.New()
	logger = logrus.NewEntry(l)
}

var (
	logger *logrus.Entry
	fields = []string{"testsuite", "testcase", "step", "executor"}
)

func fieldsFromContext(ctx context.Context, keys ...string) logrus.Fields {
	var fields = logrus.Fields{}
	if ctx == nil {
		return fields
	}
	for _, k := range keys {
		ck := ContextKey(k)
		i := ctx.Value(ck)
		if i != nil {
			fields[k] = i
		}
	}
	return fields
}

func Debug(ctx context.Context, format string, args ...interface{}) {
	fields := fieldsFromContext(ctx, fields...)
	logger.WithFields(fields).Debugf(format, args...)
}

func Info(ctx context.Context, format string, args ...interface{}) {
	fields := fieldsFromContext(ctx, fields...)
	logger.WithFields(fields).Infof(format, args...)
}

func Warn(ctx context.Context, format string, args ...interface{}) {
	fields := fieldsFromContext(ctx, fields...)
	logger.WithFields(fields).Warnf(format, args...)
}

func Warning(ctx context.Context, format string, args ...interface{}) {
	fields := fieldsFromContext(ctx, fields...)
	logger.WithFields(fields).Warningf(format, args...)
}

func Error(ctx context.Context, format string, args ...interface{}) {
	fields := fieldsFromContext(ctx, fields...)
	logger.WithFields(fields).Errorf(format, args...)
}

func Fatal(ctx context.Context, format string, args ...interface{}) {
	fields := fieldsFromContext(ctx, fields...)
	logger.WithFields(fields).Fatalf(format, args...)
}
