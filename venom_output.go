package venom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	tap "github.com/mndrix/tap-go"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	color.NoColor = true
	if os.Getenv("IS_TTY") == "" || strings.ToLower(os.Getenv("IS_TTY")) == "true" || os.Getenv("IS_TTY") == "1" {
		color.NoColor = false
	}
}

// CleanUpSecrets This method tries to hide all the sensitive variables
func (v *Venom) CleanUpSecrets(testSuite TestSuite) TestSuite {
	for _, testCase := range testSuite.TestCases {
		ctx := v.processSecrets(context.Background(), &testSuite, &testCase)
		for _, result := range testCase.TestStepResults {
			for k, v := range result.ComputedVars {
				if !strings.HasPrefix(k, "venom.") {
					result.ComputedVars[k] = HideSensitive(ctx, v)
				}
			}
			for k, v := range result.InputVars {
				if !strings.HasPrefix(k, "venom.") {
					result.InputVars[k] = HideSensitive(ctx, v)
				}
			}
			for k, v := range testCase.TestCaseInput.Vars {
				if !strings.HasPrefix(k, "venom.") {
					testCase.TestCaseInput.Vars[k] = HideSensitive(ctx, v)
				}
			}
			result.Raw = HideSensitive(ctx, fmt.Sprint(result.Raw))
			result.Interpolated = HideSensitive(ctx, fmt.Sprint(result.Interpolated))
			result.Systemout = HideSensitive(ctx, result.Systemout)
			result.Systemerr = HideSensitive(ctx, result.Systemerr)
		}
	}
	return testSuite
}

// OutputResult output result to sdtout, files...
func (v *Venom) OutputResult() error {
	if v.OutputDir == "" {
		return nil
	}
	cleanedTs := []TestSuite{}
	for i := range v.Tests.TestSuites {
		tcFiltered := []TestCase{}
		for _, tc := range v.Tests.TestSuites[i].TestCases {
			if tc.IsEvaluated {
				tcFiltered = append(tcFiltered, tc)
			}
		}
		v.Tests.TestSuites[i].TestCases = tcFiltered
		ts := v.CleanUpSecrets(v.Tests.TestSuites[i])
		cleanedTs = append(cleanedTs, ts)

		testsResult := &Tests{
			TestSuites:       []TestSuite{ts},
			Status:           v.Tests.Status,
			NbTestsuitesFail: v.Tests.NbTestsuitesFail,
			NbTestsuitesPass: v.Tests.NbTestsuitesPass,
			NbTestsuitesSkip: v.Tests.NbTestsuitesSkip,
			Duration:         v.Tests.Duration,
			Start:            v.Tests.Start,
			End:              v.Tests.End,
		}

		var data []byte
		var err error

		switch v.OutputFormat {
		case "json":
			data, err = json.MarshalIndent(testsResult, "", "  ")
			if err != nil {
				return errors.Wrapf(err, "Error: cannot format output json (%s)", err)
			}
		case "tap":
			data, err = outputTapFormat(*testsResult)
			if err != nil {
				return errors.Wrapf(err, "Error: cannot format output tap (%s)", err)
			}
		case "yml", "yaml":
			data, err = yaml.Marshal(testsResult)
			if err != nil {
				return errors.Wrapf(err, "Error: cannot format output yaml (%s)", err)
			}
		case "xml":
			data, err = outputXMLFormat(*testsResult, v.Verbose)
			if err != nil {
				return errors.Wrapf(err, "Error: cannot format output xml (%s)", err)
			}
		case "html":
			return errors.New("Error: you have to use the --html-report flag")
		}

		fname := strings.TrimSuffix(filepath.Base(ts.Filepath), filepath.Ext(ts.Filepath))
		filename := filepath.Join(v.OutputDir, "test_results_"+fname+"."+v.OutputFormat)
		if err := os.WriteFile(filename, data, 0o600); err != nil {
			return fmt.Errorf("Error while creating file %s: %v", filename, err)
		}
		v.PrintFunc("Writing file %s\n", filename)
	}

	if v.HtmlReport {
		testsResult := &Tests{
			TestSuites:       cleanedTs,
			Status:           v.Tests.Status,
			NbTestsuitesFail: v.Tests.NbTestsuitesFail,
			NbTestsuitesPass: v.Tests.NbTestsuitesPass,
			NbTestsuitesSkip: v.Tests.NbTestsuitesSkip,
			Duration:         v.Tests.Duration,
			Start:            v.Tests.Start,
			End:              v.Tests.End,
		}

		data, err := outputHTML(testsResult)
		if err != nil {
			return errors.Wrapf(err, "Error: cannot format output html")
		}
		filename := filepath.Join(v.OutputDir, computeOutputFilename("test_results.html"))
		v.PrintFunc("Writing html file %s\n", filename)
		if err := os.WriteFile(filename, data, 0o600); err != nil {
			return errors.Wrapf(err, "Error while creating file %s", filename)
		}
	}

	return nil
}

func outputTapFormat(tests Tests) ([]byte, error) {
	tapValue := tap.New()
	buf := new(bytes.Buffer)
	tapValue.Writer = buf
	var total int
	for _, ts := range tests.TestSuites {
		for _, tc := range ts.TestCases {
			total++
			name := ts.Name + " / " + tc.Name
			if len(tc.Skipped) > 0 {
				tapValue.Skip(1, name)
				continue
			}

			for _, testStepResult := range tc.TestStepResults {
				if len(testStepResult.Errors) > 0 {
					tapValue.Fail(name)
					for _, e := range testStepResult.Errors {
						tapValue.Diagnosticf("Error: %s", e.Value)
					}
					continue
				}
			}
			tapValue.Pass(name)
		}
	}
	tapValue.Header(total)

	return buf.Bytes(), nil
}

func outputXMLFormat(tests Tests, verbose int) ([]byte, error) {
	testsXML := TestsXML{}

	for _, ts := range tests.TestSuites {
		tsXML := TestSuiteXML{
			Name:    ts.Name,
			Package: ts.Filepath,
			Time:    fmt.Sprintf("%f", ts.Duration),
		}

		for _, tc := range ts.TestCases {
			switch tc.Status {
			case StatusFail:
				tsXML.Errors++
			case StatusSkip:
				tsXML.Skipped++
			}
			tsXML.Total++

			failuresXML := []FailureXML{}
			systemout := InnerResult{}
			systemerr := InnerResult{}
			for _, result := range tc.TestStepResults {
				for _, failure := range result.Errors {
					failuresXML = append(failuresXML, FailureXML{
						Value: failure.Value,
					})
				}
				if len(result.Errors) > 0 {
					appendCleanValue(&systemout.Value, result.Systemout)
				} else if verbose > 1 {
					appendCleanValue(&systemout.Value, result.Systemout)
				}
				appendCleanValue(&systemerr.Value, result.Systemerr)
			}

			tcXML := TestCaseXML{
				Classname: ts.Filename,
				Errors:    failuresXML,
				Name:      tc.Name,
				Skipped:   tc.Skipped,
				Systemout: systemout,
				Systemerr: systemerr,
				Time:      tc.Duration,
				ID:        tc.ID,
			}
			tsXML.TestCases = append(tsXML.TestCases, tcXML)
		}
		testsXML.TestSuites = append(testsXML.TestSuites, tsXML)
	}

	dataxml, err := xml.MarshalIndent(testsXML, "", "  ")
	if err != nil {
		errors.Wrapf(err, "Error: cannot format xml output")
	}
	data := append([]byte(`<?xml version="1.0" encoding="utf-8"?>`), dataxml...)

	return data, nil
}

func appendCleanValue(dest *string, source string) {
	cleanedValue := strings.ReplaceAll(source, "\x03", "")
	*dest += cleanedValue
}
