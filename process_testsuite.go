package venom

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

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

	// Intialiaze the testsuite variables and compute a first interpolation over them
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

	ts.Vars.Add("venom.outputdir", v.OutputDir)
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

	ts.Status = StatusRun

	// ##### RUN Test Cases Here
	v.runTestCases(ctx, ts)

	var isFailed bool
	var nSkip int
	for _, tc := range ts.TestCases {
		if tc.Status == StatusFail {
			isFailed = true
			ts.NbTestcasesFail++
		} else if tc.Status == StatusSkip {
			nSkip++
			ts.NbTestcasesSkip++
		} else if tc.Status == StatusPass {
			ts.NbTestcasesPass++
		}
	}

	if isFailed {
		ts.Status = StatusFail
		v.Tests.NbTestsuitesFail++
	} else if nSkip > 0 && nSkip == len(ts.TestCases) {
		ts.Status = StatusSkip
		v.Tests.NbTestsuitesSkip++
	} else {
		ts.Status = StatusPass
		v.Tests.NbTestsuitesPass++
	}
}

func (v *Venom) runTestCases(ctx context.Context, ts *TestSuite) {
	verboseReport := v.Verbose >= 1

	v.Println(" • %s (%s)", ts.Name, ts.Filepath)

	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		tc.IsEvaluated = true
		v.Print(" \t• %s", tc.Name)
		var hasFailure bool
		var hasRanged bool
		var hasSkipped = len(tc.Skipped) > 0
		if !hasSkipped {
			start := time.Now()
			tc.Start = start
			ts.Status = StatusRun
			if verboseReport || hasRanged {
				v.Print("\n")
			}
			// ##### RUN Test Case Here
			v.runTestCase(ctx, ts, tc)
			tc.End = time.Now()
			tc.Duration = tc.End.Sub(tc.Start).Seconds()
		}

		skippedSteps := 0
		for _, testStepResult := range tc.TestStepResults {
			if testStepResult.RangedEnable {
				hasRanged = true
			}
			if testStepResult.Status == StatusFail {
				hasFailure = true
			}
			if testStepResult.Status == StatusSkip {
				skippedSteps++
			}
		}

		if hasFailure {
			tc.Status = StatusFail
		} else if skippedSteps == len(tc.TestStepResults) {
			//If all test steps were skipped, consider the test case as skipped
			tc.Status = StatusSkip
		} else if tc.Status != StatusSkip {
			tc.Status = StatusPass
		}

		// Verbose mode already reported tests status, so just print them when non-verbose
		indent := ""
		if hasRanged || verboseReport {
			indent = "\t  "
			// If the testcase was entirely skipped, then the verbose mode will not have any output
			// Print something to inform that the testcase was indeed processed although skipped
			if len(tc.TestStepResults) == 0 {
				v.Println("\t\t%s", Gray("• (all steps were skipped)"))
				continue
			}
		} else {
			if hasFailure {
				v.Println(" %s", Red(StatusFail))
			} else if tc.Status == StatusSkip {
				v.Println(" %s", Gray(StatusSkip))
				continue
			} else {
				v.Println(" %s", Green(StatusPass))
			}
		}

		for _, i := range tc.computedInfo {
			v.Println("\t  %s%s %s", indent, Cyan("[info]"), Cyan(i))
		}

		for _, i := range tc.computedVerbose {
			v.PrintlnIndentedTrace(i, indent)
		}

		// Verbose mode already reported failures, so just print them when non-verbose
		if !hasRanged && !verboseReport && hasFailure {
			for _, testStepResult := range tc.TestStepResults {
				for _, f := range testStepResult.Errors {
					v.Println("%s", Yellow(f.Value))
				}
			}
		}

		if v.StopOnFailure {
			for _, testStepResult := range tc.TestStepResults {
				if len(testStepResult.Errors) > 0 {
					// break TestSuite
					return
				}
			}
		}
		ts.ComputedVars.AddAllWithPrefix(tc.Name, tc.computedVars)
	}
}

// Parse the suite to find unreplaced and extracted variables
func (v *Venom) parseTestSuite(ts *TestSuite) ([]string, []string, error) {
	return v.parseTestCases(ts)
}

// Parse the testscases to find unreplaced and extracted variables
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
