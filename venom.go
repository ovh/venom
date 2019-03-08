package venom

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	//variable are set with -ldflags "-X github.com/ovh/venom/venom.Version=$(VERSION)"
	Version   = "snapshot"
	GOOS      = "linux"
	GOARCH    = "amd64"
	GITHASH   = "0000000"
	BUILDTIME = ""
)

func New() *Venom {
	v := &Venom{
		LogLevel:        "info",
		LogOutput:       os.Stdout,
		logger:          logrus.New(),
		PrintFunc:       fmt.Printf,
		variables:       map[string]string{},
		EnableProfiling: false,
		IgnoreVariables: []string{},
		OutputFormat:    "xml",
	}
	return v
}

type Venom struct {
	LogLevel  string
	LogOutput io.Writer
	logger    *logrus.Logger

	ConfigurationDirectory string

	PrintFunc func(format string, a ...interface{}) (n int, err error)

	testsuites      []TestSuite
	variables       H
	IgnoreVariables []string
	Parallel        int

	EnableProfiling bool
	OutputFormat    string
	OutputDir       string
	StopOnFailure   bool
}

func (v *Venom) GetLogger() Logger {
	return v.logger
}

type VenomExecutor interface{}

func (v *Venom) AddVariables(variables map[string]string) {
	for k, variable := range variables {
		v.variables[k] = variable
	}
}

func (v *Venom) init() error {
	v.testsuites = []TestSuite{}
	if v.Parallel == 0 {
		v.Parallel = 1
	}

	v.logger = logrus.New()
	formatter := new(LogFormatter)
	v.logger.Formatter = formatter

	switch v.LogLevel {
	case "debug":
		v.logger.SetLevel(logrus.DebugLevel)
	case "info":
		v.logger.SetLevel(logrus.InfoLevel)
	case "error":
		v.logger.SetLevel(logrus.ErrorLevel)
	default:
		v.logger.SetLevel(logrus.WarnLevel)
	}

	v.logger.SetOutput(v.LogOutput)

	return nil
}
