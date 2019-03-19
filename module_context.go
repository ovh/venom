package venom

import (
	"context"

	"github.com/pkg/errors"
)

type contextModule interface {
	Manifest() VenomModuleManifest
	New(ctx context.Context, values H) (TestContext, error)
	ExecutorSupported() []string
}

func (v *Venom) getContextModule(s string) (contextModule, error) {
	if s == "" || s == "default" {
		return commonContextModule{}, nil
	}
	return nil, errors.New("unsupported context")
}

type commonContextModule struct{}

func (e commonContextModule) Manifest() VenomModuleManifest {
	return VenomModuleManifest{}
}

func (e commonContextModule) New(ctx context.Context, values H) (TestContext, error) {
	return &commonContext{Context: ctx, values: values}, nil
}

func (e commonContextModule) ExecutorSupported() []string {
	return nil
}

type commonContext struct {
	context.Context
	values           H
	workingDirectory string
}

func (e *commonContext) SetWorkingDirectory(s string) {
	e.workingDirectory = s
}
func (e *commonContext) GetWorkingDirectory() string {
	return e.workingDirectory
}

func (e *commonContext) Bag() H {
	return e.values
}
