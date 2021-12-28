package venom

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fatih/color"
	tap "github.com/mndrix/tap-go"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	color.NoColor = true
	if os.Getenv("IS_TTY") == "" || strings.ToLower(os.Getenv("IS_TTY")) == "true" || os.Getenv("IS_TTY") == "1" {
		color.NoColor = false
	}
}

// OutputResult output result to sdtout, files...
func (v *Venom) OutputResult(tests Tests, elapsed time.Duration) error {
	if v.OutputDir == "" {
		return nil
	}
	for i := range tests.TestSuites {
		tcFiltered := []TestCase{}
		for _, tc := range tests.TestSuites[i].TestCases {
			if tc.IsEvaluated {
				tcFiltered = append(tcFiltered, tc)
			}
		}
		tests.TestSuites[i].TestCases = tcFiltered
	}
	var data []byte
	var err error
	switch v.OutputFormat {
	case "json":
		data, err = json.MarshalIndent(tests, "", "  ")
		if err != nil {
			log.Fatalf("Error: cannot format output json (%s)", err)
		}
	case "tap":
		data, err = outputTapFormat(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output tap (%s)", err)
		}
	case "yml", "yaml":
		data, err = yaml.Marshal(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output yaml (%s)", err)
		}
	default:
		dataxml, errm := xml.MarshalIndent(tests, "", "  ")
		if errm != nil {
			log.Fatalf("Error: cannot format xml output: %s", errm)
		}
		data = append([]byte(`<?xml version="1.0" encoding="utf-8"?>`), dataxml...)
	}

	filename := path.Join(v.OutputDir, "test_results."+v.OutputFormat)
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("Error while creating file %s: %v", filename, err)
	}
	v.PrintFunc("Writing file %s\n", filename)

	return nil
}

func outputTapFormat(tests Tests) ([]byte, error) {
	tapValue := tap.New()
	buf := new(bytes.Buffer)
	tapValue.Writer = buf
	tapValue.Header(tests.Total)
	for _, ts := range tests.TestSuites {
		for _, tc := range ts.TestCases {
			name := ts.Name + " / " + tc.Name
			if len(tc.Skipped) > 0 {
				tapValue.Skip(1, name)
				continue
			}

			if len(tc.Errors) > 0 {
				tapValue.Fail(name)
				for _, e := range tc.Errors {
					tapValue.Diagnosticf("Error: %s", e.Value)
				}
				continue
			}

			if len(tc.Failures) > 0 {
				tapValue.Fail(name)
				for _, e := range tc.Failures {
					tapValue.Diagnosticf("Failure: %s", e.Value)
				}
				continue
			}

			tapValue.Pass(name)
		}
	}

	return buf.Bytes(), nil
}
