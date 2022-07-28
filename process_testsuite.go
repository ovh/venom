package venom

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/fatih/color"
	"github.com/gosimple/slug"
	"github.com/ovh/cds/sdk/interpolate"
	log "github.com/sirupsen/logrus"
)

func (v *Venom) runTestSuite(ctx context.Context, ts *TestSuite) {
	if v.Verbose == 3 {
		var filename, filenameCPU, filenameMem string
		if v.OutputDir != "" {
			filename = v.OutputDir + "/"
		}
		filenameCPU = filename + "pprof_cpu_profile_" + ts.Filename + ".prof"
		filenameMem = filename + "pprof_mem_profile_" + ts.Filename + ".prof"
		fCPU, errCPU := os.Create(filenameCPU)
		fMem, errMem := os.Create(filenameMem)
		if errCPU != nil || errMem != nil {
			log.Errorf("error while create profile file CPU:%v MEM:%v", errCPU, errMem)
		} else {
			pprof.StartCPUProfile(fCPU)
			p := pprof.Lookup("heap")
			defer p.WriteTo(fMem, 1)
			defer pprof.StopCPUProfile()
		}
	}

	// Intialiaze the testsuite varibles and compute a first interpolation over them
	ts.Vars.AddAll(v.variables.Clone())
	vars, _ := DumpStringPreserveCase(ts.Vars)
	for k, v := range vars {
		computedV, err := interpolate.Do(fmt.Sprintf("%v", v), vars)
		if err != nil {
			log.Errorf("error while computing variable %s=%q: %v", k, v, err)
		}
		ts.Vars.Add(k, computedV)
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Errorf("failed to get executable path: %v", err)
	} else {
		ts.Vars.Add("venom.executable", exePath)
	}

	ts.Vars.Add("venom.libdir", v.LibDir)
	ts.Vars.Add("venom.testsuite", ts.Name)
	ts.ComputedVars = H{}

	ctx = context.WithValue(ctx, ContextKey("testsuite"), ts.Name)
	Info(ctx, "Starting testsuite")
	defer Info(ctx, "Ending testsuite")

	totalSteps := 0
	for _, tc := range ts.TestCases {
		totalSteps += len(tc.testSteps)
	}

	v.runTestCases(ctx, ts)
}

func (v *Venom) runTestCases(ctx context.Context, ts *TestSuite) {
	var red = color.New(color.FgRed).SprintFunc()
	var green = color.New(color.FgGreen).SprintFunc()
	var cyan = color.New(color.FgCyan).SprintFunc()
	var gray = color.New(color.Attribute(90)).SprintFunc()
	verboseReport := v.Verbose >= 2

	v.Println(" • %s (%s)", ts.Name, ts.Package)

	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		tc.IsEvaluated = true
		v.Print(" \t• %s", tc.Name)
		if verboseReport {
			v.Print("\n")
		}
		tc.Classname = ts.Filename
		var hasFailure bool
		var hasSkipped = len(tc.Skipped) > 0
		if !hasSkipped {
			start := time.Now()
			v.runTestCase(ctx, ts, tc)
			tc.Time = time.Since(start).Seconds()
		}

		if len(tc.Failures) > 0 {
			ts.Failures += len(tc.Failures)
			hasFailure = true
		}
		if len(tc.Errors) > 0 {
			ts.Errors += len(tc.Errors)
			hasFailure = true
		}
		if len(tc.Skipped) > 0 {
			ts.Skipped += len(tc.Skipped)
			hasSkipped = true
		}

		// Verbose mode already reported tests status, so just print them when non-verbose
		indent := ""
		if verboseReport {
			indent = "\t  "
		} else {
			if hasSkipped {
				v.Println(" %s", gray("SKIPPED"))
				continue
			}

			if hasFailure {
				v.Println(" %s", red("FAILURE"))
			} else {
				v.Println(" %s", green("SUCCESS"))
			}
		}

		for _, i := range tc.computedInfo {
			v.Println("\t  %s%s %s", indent, cyan("[info]"), cyan(i))
		}

		for _, i := range tc.computedVerbose {
			v.PrintlnIndentedTrace(i, indent)
		}

		// Verbose mode already reported failures, so just print them when non-verbose
		if !verboseReport && hasFailure {
			for _, f := range tc.Failures {
				v.Println("%s", red(f))
			}
			for _, f := range tc.Errors {
				v.Println("%s", red(f.Value))
			}
		}

		if v.StopOnFailure && (len(tc.Failures) > 0 || len(tc.Errors) > 0) {
			// break TestSuite
			return
		}
		ts.ComputedVars.AddAllWithPrefix(tc.Name, tc.computedVars)
	}
}

//Parse the suite to find unreplaced and extracted variables
func (v *Venom) parseTestSuite(ts *TestSuite) ([]string, []string, error) {
	return v.parseTestCases(ts)
}

//Parse the testscases to find unreplaced and extracted variables
func (v *Venom) parseTestCases(ts *TestSuite) ([]string, []string, error) {
	var vars []string
	var extractsVars []string
	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		tc.originalName = tc.Name
		tc.Name = slug.Make(tc.Name)
		tc.Vars = ts.Vars.Clone()
		tc.Vars.Add("venom.testcase", tc.Name)

		if len(tc.Skipped) == 0 {
			tvars, tExtractedVars, err := v.parseTestCase(ts, tc)
			if err != nil {
				return nil, nil, err
			}
			for _, k := range tvars {
				var found bool
				for i := 0; i < len(vars); i++ {
					if vars[i] == k {
						found = true
						break
					}
				}
				if !found {
					vars = append(vars, k)
				}
			}
			for _, k := range tExtractedVars {
				var found bool
				for i := 0; i < len(extractsVars); i++ {
					if extractsVars[i] == k {
						found = true
						break
					}
				}
				if !found {
					extractsVars = append(extractsVars, k)
				}
			}
		}
	}

	return vars, extractsVars, nil
}
