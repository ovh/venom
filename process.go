package venom

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gosimple/slug"
	"github.com/sirupsen/logrus"
)

func (v *Venom) init() error {
	v.testsuites = []TestSuite{}
	switch v.LogLevel {
	case "disable":
		v.LogOutput = ioutil.Discard
		logrus.SetLevel(logrus.WarnLevel)
		logrus.SetOutput(v.LogOutput)
		return nil
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "error":
		logrus.SetLevel(logrus.WarnLevel)
	default:
		logrus.SetLevel(logrus.WarnLevel)
	}

	if v.OutputDir != "" {
		if err := os.MkdirAll(v.OutputDir, os.FileMode(0755)); err != nil {
			return fmt.Errorf("unable to create output dir: %v", err)
		}
	}

	var err error
	var logFile = filepath.Join(v.OutputDir, "venom.log")
	_ = os.RemoveAll(logFile)
	v.LogOutput, err = os.OpenFile(logFile, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("unable to write log file: %v", err)
	}

	logrus.SetOutput(v.LogOutput)
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
	if err := v.init(); err != nil {
		return err
	}

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

		Debug(context.TODO(), "ts(%s).Vars: %+v", ts.Package, ts.Vars)
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

	reallyMissingVars := []string{}
	for _, k := range missingVars {
		var varExtracted bool
		for _, e := range extractedVars {
			if strings.HasPrefix(k, e) {
				varExtracted = true
			}
		}
		if !varExtracted {
			var ignored bool
			// ignore {{.venom.var..}}
			if strings.HasPrefix(k, "venom.") {
				continue
			}
			for _, i := range v.IgnoreVariables {
				if strings.HasPrefix(k, i) {
					ignored = true
					break
				}
			}
			if !ignored {
				reallyMissingVars = append(reallyMissingVars, k)
			}
		}
	}

	if len(reallyMissingVars) > 0 {
		return fmt.Errorf("Missing variables %v", reallyMissingVars)
	}

	return nil
}

// Process runs tests suite and return a Tests result
func (v *Venom) Process(ctx context.Context, path []string) (*Tests, error) {
	if err := v.init(); err != nil {
		return nil, err
	}

	filesPath, err := getFilesPath(path)
	if err != nil {
		return nil, err
	}

	if err := v.readFiles(filesPath); err != nil {
		return nil, err
	}

	testsResult := &Tests{}

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
