package main

import (
	"C"
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/venom"
)

// Name of the executor
const Name = "hello"

// Plugin var is mandatory, it's used by venom to register the executor
var Plugin = Executor{}

// Executor is a venom executor for Hello plugin
type Executor struct {
	Arg string `json:"arg,omitempty" yaml:"arg,omitempty"`
}

// Result represents a step result.
type Result struct {
	Body string `json:"body,omitempty" yaml:"body,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (e Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// Transform step to Executor instance.
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	venom.Debug(ctx, "running plugin Hello with arg %v\n", e.Arg)
	r := Result{Body: fmt.Sprintf("Hello %v", e.Arg)}
	return r, nil
}

// ZeroValueResult return an empty implementation of this executor result
func (e Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return the default assertions of the executor.
func (e Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{}}
}
