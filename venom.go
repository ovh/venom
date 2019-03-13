package venom

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/ovh/venom/lib/cmd"
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
		LogOutput:       os.Stdout,
		logger:          logrus.New(),
		variables:       map[string]string{},
		EnableProfiling: false,
		ReportDir:       ".",
		Output:          NewOutput(os.Stdout),
	}
	return v
}

type Venom struct {
	LogLevel               string
	LogOutput              io.Writer
	logger                 *logrus.Logger
	ConfigurationDirectory string
	testsuites             []TestSuite
	variables              H
	StopOnFailure          bool
	Parallel               int
	EnableProfiling        bool
	ReportDir              string
	Output                 io.WriteCloser
	Display                *cmd.Container
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

	formatter := new(LogFormatter)
	v.logger.Formatter = formatter
	switch v.LogLevel {
	case "debug":
		v.logger.SetLevel(logrus.DebugLevel)
	case "info":
		v.logger.SetLevel(logrus.InfoLevel)
	case "warn":
		v.logger.SetLevel(logrus.WarnLevel)
	default:
		v.LogOutput = ioutil.Discard
		v.logger.SetLevel(logrus.FatalLevel)
		v.Display = new(cmd.Container)
	}

	v.logger.SetOutput(v.LogOutput)

	return nil
}
