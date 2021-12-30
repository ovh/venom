package venom

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// InitLogger initializes venom logger
func (v *Venom) InitLogger() error {
	v.Testsuites = []TestSuite{}
	if v.Verbose == 0 {
		logrus.SetLevel(logrus.WarnLevel)
	} else {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if v.OutputDir != "" {
		if err := os.MkdirAll(v.OutputDir, os.FileMode(0755)); err != nil {
			return errors.Wrapf(err, "unable to create output dir")
		}
	}

	if v.Verbose > 0 && v.VerboseOutput != "stdout" {
		var err error
		var logFile = filepath.Join(v.OutputDir, v.VerboseOutput)
		_ = os.RemoveAll(logFile)
		v.LogOutput, err = os.OpenFile(logFile, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
		if err != nil {
			return errors.Wrapf(err, "unable to write log file")
		}

		v.PrintlnTrace("writing " + logFile)

		logrus.SetOutput(v.LogOutput)
	} else if v.Verbose > 0 && v.VerboseOutput == "stdout" {
		logrus.SetOutput(os.Stdout)
	} else {
		logrus.SetOutput(io.Discard)
	}

	logrus.SetFormatter(&nested.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"testsuite", "testcase", "step", "executor"},
	})
	logger = logrus.NewEntry(logrus.StandardLogger())

	slug.Lowercase = false

	return nil
}

// Parse parses tests suite to check context and variables
func (v *Venom) Parse(ctx context.Context, path []string) error {
	filesPath, err := getFilesPath(path)
	if err != nil {
		return err
	}

	if err := v.readFiles(ctx, filesPath); err != nil {
		return err
	}

	missingVars := []string{}
	extractedVars := []string{}
	for i := range v.Testsuites {
		ts := &v.Testsuites[i]
		ts.Vars.Add("venom.testsuite", ts.Name)

		Info(ctx, "Parsing testsuite %s : %+v", ts.Package, ts.Vars)
		tvars, textractedVars, err := v.parseTestSuite(ts)
		if err != nil {
			return err
		}

		Debug(ctx, "Testsuite (%s) variables: %+v", ts.Package, ts.Vars)
		for k := range ts.Vars {
			textractedVars = append(textractedVars, k)
		}
		for _, k := range tvars {
			var found bool
			for i := 0; i < len(missingVars); i++ {
				if missingVars[i] == k {
					found = true
					break
				}
			}
			if !found {
				missingVars = append(missingVars, k)
			}
		}
		for _, k := range textractedVars {
			var found bool
			for i := 0; i < len(extractedVars); i++ {
				if extractedVars[i] == k {
					found = true
					break
				}
			}
			if !found {
				extractedVars = append(extractedVars, k)
			}
		}
	}

	vars, err := DumpStringPreserveCase(v.variables)
	if err != nil {
		return errors.Wrapf(err, "unable to parse variables")
	}

	reallyMissingVars := []string{}
	for _, k := range missingVars {
		var varExtracted bool
		for _, e := range extractedVars {
			if strings.HasPrefix(k, e) {
				varExtracted = true
				break
			}
		}
		for t := range vars {
			if t == k {
				varExtracted = true
				break
			}
		}
		if !varExtracted {
			// ignore {{.venom.var..}}
			if strings.HasPrefix(k, "venom.") {
				continue
			}
			reallyMissingVars = append(reallyMissingVars, k)
		}
	}

	if len(reallyMissingVars) > 0 {
		return fmt.Errorf("missing variables %v", reallyMissingVars)
	}

	return nil
}

// Process runs tests suite and return a Tests result
func (v *Venom) Process(ctx context.Context, path []string) (*Tests, error) {
	testsResult := &Tests{}
	Debug(ctx, "nb testsuites: %d", len(v.Testsuites))
	for i := range v.Testsuites {
		v.runTestSuite(ctx, &v.Testsuites[i])
		v.computeStats(testsResult, &v.Testsuites[i])
	}

	return testsResult, nil
}

func (v *Venom) computeStats(testsResult *Tests, ts *TestSuite) {
	testsResult.TestSuites = append(testsResult.TestSuites, *ts)
	if ts.Failures > 0 || ts.Errors > 0 {
		testsResult.TotalKO++
	} else {
		testsResult.TotalOK++
	}
	if ts.Skipped > 0 {
		testsResult.TotalSkipped++
	}

	testsResult.Total = testsResult.TotalKO + testsResult.TotalOK + testsResult.TotalSkipped
}
