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
		if err := os.MkdirAll(v.OutputDir, os.FileMode(0o755)); err != nil {
			return errors.Wrapf(err, "unable to create output dir")
		}
	}

	var err error
	logFile := filepath.Join(v.OutputDir, computeOutputFilename("venom.log"))
	v.LogOutput, err = os.OpenFile(logFile, os.O_CREATE|os.O_RDWR, os.FileMode(0o644))
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

	err = v.registerUserExecutors(ctx)
	if err != nil {
		return errors.Wrapf(err, "unable to register user executors")
	}

	missingVars := []string{}
	extractedVars := []string{}
	for i := range v.Tests.TestSuites {
		ts := &v.Tests.TestSuites[i]
		ts.Vars.Add("venom.testsuite", ts.Name)

		Info(ctx, "Parsing testsuite %s", ts.Filepath)
		tvars, textractedVars, err := v.parseTestSuite(ts)
		if err != nil {
			return err
		}

		for k := range ts.Vars {
			textractedVars = append(textractedVars, k)
		}

		Debug(ctx, "Testsuite (%s) variables: %s", ts.Filepath, strings.Join(textractedVars, ","))

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
	for i := range v.Tests.TestSuites {

		v.Tests.TestSuites[i].Start = time.Now()
		// ##### RUN Test Suite Here
		if err := v.runTestSuite(ctx, &v.Tests.TestSuites[i]); err != nil {
			return err
		}

		v.Tests.TestSuites[i].End = time.Now()
		v.Tests.TestSuites[i].Duration = v.Tests.TestSuites[i].End.Sub(v.Tests.TestSuites[i].Start).Seconds()
	}
	v.Tests.End = time.Now()
	v.Tests.Duration = v.Tests.End.Sub(v.Tests.Start).Seconds()

	var isFailed bool
	var nSkip int
	for i := range v.Tests.TestSuites {
		if v.Tests.TestSuites[i].Status == StatusFail {
			isFailed = true
			break
		} else if v.Tests.TestSuites[i].Status == StatusSkip {
			nSkip++
		}
	}
	if isFailed {
		v.Tests.Status = StatusFail
	} else if nSkip > 0 && nSkip == len(v.Tests.TestSuites) {
		v.Tests.Status = StatusSkip
	} else {
		v.Tests.Status = StatusPass
	}

	Debug(ctx, "final status: %s", v.Tests.Status)

	return nil
}
