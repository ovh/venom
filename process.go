package venom

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/fsamin/go-dump"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// InitLogger initializes venom logger
func (v *Venom) InitLogger() error {
	v.testsuites = []TestSuite{}
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

	if v.Verbose > 0 {
		var err error
		var logFile = filepath.Join(v.OutputDir, "venom.log")
		_ = os.RemoveAll(logFile)
		v.LogOutput, err = os.OpenFile(logFile, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
		if err != nil {
			return errors.Wrapf(err, "unable to write log file")
		}

		v.PrintlnTrace("writing " + logFile)

		logrus.SetOutput(v.LogOutput)
	} else {
		logrus.SetOutput(ioutil.Discard)
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
func (v *Venom) Parse(path []string) error {
	filesPath, err := getFilesPath(path)
	if err != nil {
		return err
	}

	if err := v.readFiles(filesPath); err != nil {
		return err
	}

	missingVars := []string{}
	extractedVars := []string{}
	for i := range v.testsuites {
		ts := &v.testsuites[i]
		ts.Vars.Add("venom.testsuite", ts.Name)

		Info(context.Background(), "Parsing testsuite %s : %+v", ts.Package, ts.Vars)
		tvars, textractedVars, err := v.parseTestSuite(ts)
		if err != nil {
			return err
		}

		Debug(context.TODO(), "Testsuite (%s) variables: %+v", ts.Package, ts.Vars)
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

	vars, err := dump.ToStringMap(v.variables)
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
		return fmt.Errorf("Missing variables %v", reallyMissingVars)
	}

	return nil
}

// Process runs tests suite and return a Tests result
func (v *Venom) Process(ctx context.Context, path []string) (*Tests, error) {
	testsResult := &Tests{}
	Debug(ctx, "nb testsuites: %d", len(v.testsuites))
	for i := range v.testsuites {
		v.runTestSuite(ctx, &v.testsuites[i])
		v.computeStats(testsResult, &v.testsuites[i])
	}

	return testsResult, nil
}

func (v *Venom) computeStats(testsResult *Tests, ts *TestSuite) {
	testsResult.TestSuites = append(testsResult.TestSuites, *ts)
	if ts.Failures > 0 || ts.Errors > 0 {
		testsResult.TotalKO += (ts.Failures + ts.Errors)
	} else {
		testsResult.TotalOK += len(ts.TestCases) - (ts.Failures + ts.Errors)
	}
	if ts.Skipped > 0 {
		testsResult.TotalSkipped += ts.Skipped
	}

	testsResult.Total = testsResult.TotalKO + testsResult.TotalOK + testsResult.TotalSkipped
}
