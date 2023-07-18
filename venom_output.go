package venom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path"
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

// OutputResult output result to sdtout, files...
func (v *Venom) OutputResult(ctx context.Context) error {
	if v.OutputDir == "" {
		return nil
	}

	suites, err2 := gatherPartialReports(ctx, v.OutputDir, v.OutputFormat)
	if err2 != nil {
		return err2
	}
	v.Tests.TestSuites = suites

	if v.HtmlReport {
		testsResult := &Tests{
			TestSuites:       v.Tests.TestSuites,
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
		var filename = filepath.Join(v.OutputDir, computeOutputFilename("test_results.html"))
		_, _ = v.PrintFunc("Writing html file %s\n", filename)
		if err := os.WriteFile(filename, data, 0600); err != nil {
			return errors.Wrapf(err, "Error while creating file %s", filename)
		}
	}

	return nil
}

func gatherPartialReports(ctx context.Context, directory string, outputFormat string) ([]TestSuite, error) {
	reports := []TestSuite{}
	files, err := filepath.Glob(filepath.Join(directory, "test_results_*."+outputFormat))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		tcFiltered := Tests{}
		bts, err := os.ReadFile(f)
		if err != nil {
			Fatal(ctx, "could not read test result %v", f)
			return nil, err
		}
		errUnMarshal := json.Unmarshal(bts, &tcFiltered)
		if errUnMarshal != nil {
			Fatal(ctx, "Could not read test result %v", f)
			return nil, errUnMarshal
		}
		for _, suite := range tcFiltered.TestSuites {
			reports = append(reports, suite)
		}

	}

	return reports, nil
}
func (v *Venom) GenerateOutputForTestSuite(ctx context.Context, ts *TestSuite) error {
	if v.OutputDir == "" {
		return nil
	}

	tcFiltered := []TestCase{}
	for _, tc := range ts.TestCases {
		if tc.IsEvaluated {
			tcFiltered = append(tcFiltered, tc)
		}
	}
	ts.TestCases = tcFiltered

	testsResult := &Tests{
		TestSuites:       []TestSuite{*ts},
		Status:           ts.Status,
		NbTestsuitesFail: ts.NbTestcasesFail,
		NbTestsuitesPass: ts.NbTestcasesPass,
		NbTestsuitesSkip: ts.NbTestcasesSkip,
		Duration:         ts.Duration,
		Start:            ts.Start,
		End:              ts.End,
	}

	var data []byte
	var err error

	switch v.OutputFormat {
	case "json":
		data, err = json.MarshalIndent(testsResult, "", "  ")
		if err != nil {
			Fatal(ctx, "Error: cannot format output json (%s)", err)
		}
	case "tap":
		data, err = outputTapFormat(*testsResult)
		if err != nil {
			Fatal(ctx, "Error: cannot format output tap (%s)", err)
		}
	case "yml", "yaml":
		data, err = yaml.Marshal(testsResult)
		if err != nil {
			Fatal(ctx, "Error: cannot format output yaml (%s)", err)
		}
	case "xml":
		data, err = outputXMLFormat(*testsResult)
		if err != nil {
			Fatal(ctx, "Error: cannot format output xml (%s)", err)
		}
	case "html":
		Fatal(ctx, "Error: you have to use the --html-report flag")
	}

	fname := strings.TrimSuffix(ts.Filepath, filepath.Ext(ts.Filepath))
	fname = strings.ReplaceAll(fname, "/", "_")
	filename := path.Join(v.OutputDir, "test_results_"+fname+"."+v.OutputFormat)
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("error while creating file %s: %v", filename, err)
	}
	if _, err := v.PrintFunc("Writing file %s\n", filename); err != nil {
		return fmt.Errorf("error while writing in file %s: %v", filename, err)
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

func outputXMLFormat(tests Tests) ([]byte, error) {
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
				systemout.Value += result.Systemout
				systemerr.Value += result.Systemerr
			}

			tcXML := TestCaseXML{
				Classname: ts.Filename,
				Errors:    failuresXML,
				Name:      tc.Name,
				Skipped:   tc.Skipped,
				Systemout: systemout,
				Systemerr: systemerr,
				Time:      tc.Duration,
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
