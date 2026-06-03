package venom

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/gosimple/slug"
	"github.com/ovh/venom/interpolate"
	"github.com/pkg/errors"
)

// crossTCVarRegex matches references to other testcase variables like {{.testA.something}}
var crossTCVarRegex = regexp.MustCompile(`\{\{\s*\.([^.\s}]+)\.`)

// hasCrossTestCaseDependencies checks whether any testcase in the suite references
// variables extracted by another testcase (e.g. {{.testA.myvariable}}).
// If such dependencies exist, parallel execution would be unsafe because a testcase
// might run before the testcase it depends on has finished producing its variables.
// Returns true if dependencies are detected, along with a human-readable description.
func hasCrossTestCaseDependencies(ts *TestSuite) (bool, string) {
	// Build a set of testcase names (slug form, as used in variable references)
	tcNames := make(map[string]struct{}, len(ts.TestCases))
	for i := range ts.TestCases {
		tcNames[slug.Make(ts.TestCases[i].Name)] = struct{}{}
	}

	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		tcSlug := slug.Make(tc.Name)
		for _, rawStep := range tc.RawTestSteps {
			matches := crossTCVarRegex.FindAllStringSubmatch(string(rawStep), -1)
			for _, match := range matches {
				ref := match[1]
				// Skip self-references and known built-in prefixes
				if ref == tcSlug {
					continue
				}
				if ref == "venom" || ref == "value" || ref == "index" || ref == "key" {
					continue
				}
				// If the reference matches another testcase name, we have a dependency
				if _, ok := tcNames[ref]; ok {
					return true, fmt.Sprintf("testcase %q references variable from testcase %q", tc.Name, ref)
				}
			}
		}
	}
	return false, ""
}

func (v *Venom) runTestSuite(ctx context.Context, ts *TestSuite) error {
	if v.Verbose == 3 {
		filenameCPU := filepath.Join(v.OutputDir, "pprof_cpu_profile_"+ts.Filename+".prof")
		filenameMem := filepath.Join(v.OutputDir, "pprof_mem_profile_"+ts.Filename+".prof")
		fCPU, errCPU := os.Create(filenameCPU)
		fMem, errMem := os.Create(filenameMem)
		if errCPU != nil || errMem != nil {
			return fmt.Errorf("error while create profile file CPU:%v MEM:%v", errCPU, errMem)
		} else {
			pprof.StartCPUProfile(fCPU)
			p := pprof.Lookup("heap")
			defer p.WriteTo(fMem, 1)
			defer pprof.StopCPUProfile()
		}
	}

	// Initialize the testsuite variables and compute a first interpolation over them
	ts.Vars.AddAll(v.variables.Clone())
	vars, _ := DumpStringPreserveCase(ts.Vars)
	for k, v := range vars {
		computedV, err := interpolate.Do(fmt.Sprintf("%v", v), vars)
		if err != nil {
			return errors.Wrapf(err, "error while computing variable %s=%q", k, v)
		}
		ts.Vars.Add(k, computedV)
	}

	exePath, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "failed to get executable path")
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
		totalSteps += len(tc.RawTestSteps)
	}

	ts.Vars.Add(("venom.testsuite.totalSteps"), totalSteps)
	ts.Status = StatusRun
	Info(ctx, "With secrets in testsuite")
	for _, v := range ts.Secrets {
		Info(ctx, "secret  %+v", v)
	}
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
	return nil
}

func (v *Venom) runTestCases(ctx context.Context, ts *TestSuite) {
	verboseReport := v.Verbose >= 1

	v.Println(" • %s (%s)", ts.Name, ts.Filepath)
	// If no parallel configured (or <=1) keep sequential behavior
	parallel := ts.Parallel

	// Safety check: if parallel is requested, verify there are no cross-testcase
	// variable dependencies that would make parallel execution unsafe.
	if parallel > 1 {
		if hasDeps, desc := hasCrossTestCaseDependencies(ts); hasDeps {
			Warn(ctx, "Parallel execution disabled for testsuite %q: %s. Falling back to sequential.", ts.Name, desc)
			v.Println(" \t%s", Yellow(fmt.Sprintf("[warn] parallel disabled: %s", desc)))
			parallel = 1
		}
	}

	if parallel <= 1 {
		for i := range ts.TestCases {
			tc := &ts.TestCases[i]
			tc.IsEvaluated = true
			v.Print(" \t• %s", tc.Name)
			var hasFailure bool
			var hasRanged bool
			hasSkipped := len(tc.Skipped) > 0
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
				// If all test steps were skipped, consider the test case as skipped
				tc.Status = StatusSkip
			} else if tc.Status != StatusSkip {
				tc.Status = StatusPass
			}

			// Verbose mode already reported tests status, so just print them when non-verbose
			indent := ""
			if verboseReport {
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

			for _, i := range tc.computedVerbose {
				v.PrintlnIndentedTrace(i, indent)
			}

			// Verbose mode already reported failures, so just print them when non-verbose
			if !verboseReport && hasFailure {
				for _, testStepResult := range tc.TestStepResults {
					if len(testStepResult.ComputedInfo) > 0 || len(testStepResult.Errors) > 0 {
						v.Println(" \t\t• %s", testStepResult.Name)
						for _, f := range testStepResult.ComputedInfo {
							v.Println(" \t\t  %s", Cyan(f))
						}
						for _, f := range testStepResult.Errors {
							v.Println(" \t\t  %s", Yellow(f.Value))
						}
					}
				}
			}

			if v.StopOnFailure {
				for _, testStepResult := range tc.TestStepResults {
					if len(testStepResult.Errors) > 0 {
						// break TestSuite
						for i := range ts.TestCases {
							tc := &ts.TestCases[i]
							if tc.Status == "" {
								tc.Status = StatusSkip
								tc.IsEvaluated = true
								tc.Skipped = append(tc.Skipped, Skipped{Value: "===== stop-on-failure: enabled ====="})
							}
						}
						return
					}
				}
			}
			ts.ComputedVars.AddAllWithPrefix(tc.Name, tc.computedVars)
		}
		return
	}

	// Parallel execution path: create worker pool and buffer per-test output
	type jobResult struct {
		idx    int
		tc     *TestCase
		output string
	}

	jobs := make(chan *TestCase)
	results := make(chan jobResult)

	var wg sync.WaitGroup
	workerCount := parallel
	if workerCount > len(ts.TestCases) {
		workerCount = len(ts.TestCases)
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tc := range jobs {
				// buffer output per testcase to avoid interleaving
				var buf bytes.Buffer
				vv := *v
				vv.PrintFunc = func(format string, a ...interface{}) (n int, err error) {
					return fmt.Fprintf(&buf, format, a...)
				}

				start := time.Now()
				tc.Start = start
				ts.Status = StatusRun
				vv.runTestCase(ctx, ts, tc)
				tc.End = time.Now()
				tc.Duration = tc.End.Sub(tc.Start).Seconds()

				results <- jobResult{tc.number - 1, tc, buf.String()}
			}
		}()
	}

	// feeder
	go func() {
		for i := range ts.TestCases {
			tc := &ts.TestCases[i]
			tc.IsEvaluated = true
			v.Print(" \t• %s", tc.Name)
			// skip cases already marked skipped
			if len(tc.Skipped) == 0 {
				jobs <- tc
			} else {
				// send immediate result for skipped tests
				results <- jobResult{i, tc, ""}
			}
		}
		close(jobs)
	}()

	// collector: collect as workers finish and print/aggregate results serially
	remaining := len(ts.TestCases)
	for remaining > 0 {
		res := <-results
		tc := res.tc

		// write buffered output first if any
		if res.output != "" {
			v.Print("%s", res.output)
		}

		var hasFailure bool
		skippedSteps := 0
		for _, testStepResult := range tc.TestStepResults {
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
			tc.Status = StatusSkip
		} else if tc.Status != StatusSkip {
			tc.Status = StatusPass
		}

		// print summarized status (non-verbose)
		if !verboseReport {
			if hasFailure {
				v.Println(" %s", Red(StatusFail))
			} else if tc.Status == StatusSkip {
				v.Println(" %s", Gray(StatusSkip))
			} else {
				v.Println(" %s", Green(StatusPass))
			}
		}

		for _, i := range tc.computedVerbose {
			v.PrintlnIndentedTrace(i, "\t  ")
		}

		if !verboseReport && hasFailure {
			for _, testStepResult := range tc.TestStepResults {
				if len(testStepResult.ComputedInfo) > 0 || len(testStepResult.Errors) > 0 {
					v.Println(" \t\t• %s", testStepResult.Name)
					for _, f := range testStepResult.ComputedInfo {
						v.Println(" \t\t  %s", Cyan(f))
					}
					for _, f := range testStepResult.Errors {
						v.Println(" \t\t  %s", Yellow(f.Value))
					}
				}
			}
		}

		if v.StopOnFailure {
			for _, testStepResult := range tc.TestStepResults {
				if len(testStepResult.Errors) > 0 {
					// break TestSuite
					for i := range ts.TestCases {
						tc := &ts.TestCases[i]
						if tc.Status == "" {
							tc.Status = StatusSkip
							tc.IsEvaluated = true
							tc.Skipped = append(tc.Skipped, Skipped{Value: "===== stop-on-failure: enabled ====="})
						}
					}
					// drain results and return
					remaining = 0
					break
				}
			}
		}

		ts.ComputedVars.AddAllWithPrefix(tc.Name, tc.computedVars)
		remaining--
	}

	// wait for workers to finish
	wg.Wait()
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
		tc.number = i + 1
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
