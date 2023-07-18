package venom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// InitLogger initializes venom logger
func (v *Venom) InitLogger() error {
	v.Tests.TestSuites = []TestSuite{}

	switch v.Verbose {
	case 1:
		logrus.SetLevel(logrus.InfoLevel)
	case 2:
		logrus.SetLevel(logrus.DebugLevel)
	default:
		logrus.SetLevel(logrus.WarnLevel)
	}

	if v.OutputDir != "" {
		if err := os.MkdirAll(v.OutputDir, os.FileMode(0755)); err != nil {
			return errors.Wrapf(err, "unable to create output dir")
		}
	}

	var err error
	var logFile = filepath.Join(v.OutputDir, computeOutputFilename("venom.log"))
	v.LogOutput, err = os.OpenFile(logFile, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	if err != nil {
		return errors.Wrapf(err, "unable to write log file")
	}
	v.PrintlnTrace("writing " + logFile)
	logrus.SetOutput(v.LogOutput)

	logrus.SetFormatter(&nested.Formatter{
		HideKeys:       true,
		FieldsOrder:    []string{"testsuite", "testcase", "step", "executor"},
		NoColors:       true,
		NoFieldsColors: true,
	})
	logger = logrus.NewEntry(logrus.StandardLogger())

	slug.Lowercase = false

	return nil
}

func computeOutputFilename(filename string) string {
	// example of filename: venom.log
	t := strings.Split(filename, ".")

	if !fileExists(filename) {
		return filename
	}
	for i := 0; ; i++ {
		filename := fmt.Sprintf("%s.%d.%s", t[0], i, t[1])
		if !fileExists(filename) {
			return filename
		}
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
	for i := range v.Tests.TestSuites {
		ts := &v.Tests.TestSuites[i]
		ts.Vars.Add("venom.testsuite", ts.Name)

		Info(ctx, "Parsing testsuite %s : %+v", ts.Filepath, ts.Vars)
		tvars, textractedVars, err := v.parseTestSuite(ts)
		if err != nil {
			return err
		}

		Debug(ctx, "Testsuite (%s) variables: %+v", ts.Filepath, ts.Vars)
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
		// Skip "range" builtin variables
		if strings.HasPrefix(k, "value") || k == "index" || k == "key" {
			continue
		}
		var varExtracted bool
		for _, e := range extractedVars {
			if k == e || strings.HasPrefix(k, e) {
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
func (v *Venom) Process(ctx context.Context, path []string) error {
	v.Tests.Status = StatusRun
	v.Tests.Start = time.Now()
	Debug(ctx, "nb testsuites: %d", len(v.Tests.TestSuites))
	testsSize := len(v.Tests.TestSuites)
	var ts TestSuite
	nSkip := 0
	for i := 0; i < testsSize; i++ {
		ts, v.Tests.TestSuites = v.Tests.TestSuites[0], v.Tests.TestSuites[1:]
		ts.Start = time.Now()
		// ##### RUN Test Suite Here
		if err := v.runTestSuite(ctx, &ts); err != nil {
			v.Tests.Status = StatusFail
			break
		}

		if ts.Status == StatusFail {
			v.Tests.Status = StatusFail
			break
		} else if ts.Status == StatusSkip {
			nSkip++
			if nSkip == testsSize {
				v.Tests.Status = StatusSkip
			}
		} else {
			v.Tests.Status = StatusPass
		}
	}
	v.Tests.End = time.Now()
	v.Tests.Duration = v.Tests.End.Sub(v.Tests.Start).Seconds()
	Debug(ctx, "final status: %s", v.Tests.Status)

	return nil
}
