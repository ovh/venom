package venom

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type contextModule interface {
	Manifest() VenomModuleManifest
	New(ctx context.Context, values TestContextValues) (TestContext, error)
}

type commonContextModule struct{}

func (e commonContextModule) Manifest() VenomModuleManifest {

	return VenomModuleManifest{}
}

func (e commonContextModule) New(ctx context.Context, values TestContextValues) (TestContext, error) {
	return &commonContext{Context: ctx, values: values}, nil
}

type commonContext struct {
	context.Context
	values           TestContextValues
	workingDirectory string
}

func (e *commonContext) SetWorkingDirectory(s string) {
	e.workingDirectory = s
}
func (e *commonContext) GetWorkingDirectory() string {
	return e.workingDirectory
}

func (e *commonContext) Get(key string) interface{} {
	return e.values[key]
}

func (e *commonContext) RunCommand(cmd string, args ...interface{}) error {
	return nil
}

func (e *commonContext) WithTimeout(d time.Duration) (cancel context.CancelFunc) {
	e.Context, cancel = context.WithTimeout(e.Context, d)
	return cancel
}

func (v *Venom) getContextModule(s string) (contextModule, error) {
	if s == "" || s == "default" {
		return commonContextModule{}, nil
	}
	return nil, errors.New("unsupported context")
}
