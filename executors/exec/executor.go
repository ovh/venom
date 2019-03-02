package main

import (
	"github.com/ovh/venom"
)

type Executor struct {
	Script string `json:"script,omitempty" yaml:"script,omitempty"`
}

// Result represents a step result
type Result struct {
	Executor      Executor    `json:"executor,omitempty" yaml:"executor,omitempty"`
	Systemout     string      `json:"systemout,omitempty" yaml:"systemout,omitempty"`
	SystemoutJSON interface{} `json:"systemoutjson,omitempty" yaml:"systemoutjson,omitempty"`
	Systemerr     string      `json:"systemerr,omitempty" yaml:"systemerr,omitempty"`
	SystemerrJSON interface{} `json:"systemerrjson,omitempty" yaml:"systemerrjson,omitempty"`
	Err           string      `json:"err,omitempty" yaml:"err,omitempty"`
	Code          string      `json:"code,omitempty" yaml:"code,omitempty"`
	TimeSeconds   float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman     string      `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := venom.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

func (e Executor) Manifest() venom.VenomModuleManifest {
	return venom.VenomModuleManifest{
		Name:    "shell",
		Type:    "executor",
		Version: venom.Version,
	}
}

func (e Executor) Run(ctx venom.TestContext, logger venom.Logger, step venom.TestStep) (venom.ExecutorResult, error) {
	r, _ := venom.Dump(Result{})
	return r, nil
}
