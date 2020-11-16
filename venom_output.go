package venom

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"reflect"
	"time"

	dump "github.com/fsamin/go-dump"
	"github.com/gosimple/slug"
	tap "github.com/mndrix/tap-go"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// OutputResult output result to sdtout, files...
func (v *Venom) OutputResult(tests Tests, elapsed time.Duration) error {
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

	if v.OutputDir != "" {
		v.PrintFunc("\n") // new line to display files written
		filename := v.OutputDir + "/test_results." + v.OutputFormat
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("Error while creating file %s: %v", filename, err)
		}
		v.PrintFunc("Writing file %s\n", filename)

		for _, ts := range tests.TestSuites {
			for _, tc := range ts.TestCases {
				for _, f := range tc.Failures {
					filename := v.OutputDir + "/" + slug.Make(ts.ShortName) + "." + slug.Make(tc.Name) + ".dump"

					sdump := &bytes.Buffer{}
					dumpEncoder := dump.NewEncoder(sdump)
					dumpEncoder.ExtraFields.DetailedMap = false
					dumpEncoder.ExtraFields.DetailedStruct = false
					dumpEncoder.ExtraFields.Len = false
					dumpEncoder.ExtraFields.Type = false
					dumpEncoder.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}

					//Try to pretty print only the result
					var smartPrinted bool
					for k, v := range f.Result {
						if k == "result" && reflect.TypeOf(v).Kind() != reflect.String {
							dumpEncoder.Fdump(v)
							smartPrinted = true
							break
						}
					}
					//If not succeed print all the stuff
					if !smartPrinted {
						dumpEncoder.Fdump(f.Result)
					}

					output := f.Value + "\n ------ Result: \n" + sdump.String() + "\n ------ Variables:\n"
					for k, v := range ts.Vars {
						output += fmt.Sprintf("%s:%v\n", k, v)
					}
					if err := ioutil.WriteFile(filename, []byte(output), 0644); err != nil {
						return fmt.Errorf("Error while creating file %s: %v", filename, err)
					}
					v.PrintFunc("File %s is written\n", filename)
				}
			}
		}
	}
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
