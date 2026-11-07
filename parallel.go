package venom

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// pluginMu serializes plugin loading and the plugin map when testcases run in parallel.
var pluginMu sync.Mutex

// parallelFor returns how many testcases of ts may run at once.
//
// The suite runs sequentially (1) when Parallel <= 1, when --stop-on-failure is set,
// when it has a single testcase, or when a testcase references a var exported by an
// earlier testcase of the same suite: a template "{{.<earlier testcase name>.<var>}}"
// found in its steps or skip conditions.
func (v *Venom) parallelFor(ts *TestSuite) int {
	if v.Parallel <= 1 || v.StopOnFailure || len(ts.TestCases) < 2 {
		return 1
	}
	if dependsOnEarlierTestCase(ts) {
		return 1
	}
	if v.Parallel > len(ts.TestCases) {
		return len(ts.TestCases)
	}
	return v.Parallel
}

// dependsOnEarlierTestCase reports whether a testcase of ts reads a var exported
// by an earlier testcase of ts. Exported vars are named "<testcase name>.<var>",
// so a reference to one always holds ".<testcase name>.".
func dependsOnEarlierTestCase(ts *TestSuite) bool {
	for i := 1; i < len(ts.TestCases); i++ {
		text := testCaseText(&ts.TestCases[i])
		for j := 0; j < i; j++ {
			name := ts.TestCases[j].Name
			if name != "" && strings.Contains(text, "."+name+".") {
				return true
			}
		}
	}
	return false
}

func testCaseText(tc *TestCase) string {
	var b strings.Builder
	for _, raw := range tc.RawTestSteps {
		b.Write(raw)
		b.WriteByte('\n')
	}
	for _, s := range tc.Skip {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return b.String()
}

// runTestCasesParallel runs the testcases of ts n at a time. Each testcase prints
// into its own buffer; the buffers are written out whole and in testcase order,
// so the console report reads as a sequential run. Results stay in ts.TestCases
// at their original index, so the JUnit/JSON/YAML/TAP reports keep the order.
func (v *Venom) runTestCasesParallel(ctx context.Context, ts *TestSuite, n int) {
	outputs := make([]bytes.Buffer, len(ts.TestCases))
	exports := make([]bool, len(ts.TestCases))
	done := make([]chan struct{}, len(ts.TestCases))
	for i := range done {
		done[i] = make(chan struct{})
	}

	sem := make(chan struct{}, n)
	go func() {
		for i := range ts.TestCases {
			sem <- struct{}{}
			go func(i int) {
				defer func() { <-sem }()
				defer close(done[i])
				vc := *v
				buf := &outputs[i]
				vc.PrintFunc = func(format string, a ...interface{}) (int, error) {
					return fmt.Fprintf(buf, format, a...)
				}
				_, exports[i] = vc.runOneTestCase(ctx, ts, &ts.TestCases[i])
			}(i)
		}
	}()

	for i := range ts.TestCases {
		<-done[i]
		v.Print("%s", outputs[i].String())
		if exports[i] {
			ts.ComputedVars.AddAllWithPrefix(ts.TestCases[i].Name, ts.TestCases[i].computedVars)
		}
	}
}

// runTimedTestSuite runs one testsuite and records its start, end and duration.
func (v *Venom) runTimedTestSuite(ctx context.Context, ts *TestSuite) error {
	ts.Start = time.Now()
	// ##### RUN Test Suite Here
	if err := v.runTestSuite(ctx, ts); err != nil {
		return err
	}
	ts.End = time.Now()
	ts.Duration = ts.End.Sub(ts.Start).Seconds()
	return nil
}

// parallelSuitesFor returns how many testsuites of the run may go at once.
//
// The run is sequential (1) when ParallelSuites <= 1, when --stop-on-failure is set,
// with -vvv (CPU profiling is process-wide), with a single testsuite, or when a
// testsuite references a var named after a testcase of another testsuite
// ("{{.<testcase name>.<var>}}").
func (v *Venom) parallelSuitesFor() int {
	suites := v.Tests.TestSuites
	if v.ParallelSuites <= 1 || v.StopOnFailure || v.Verbose >= 3 || len(suites) < 2 {
		return 1
	}
	if suitesShareVars(suites) {
		return 1
	}
	if v.ParallelSuites > len(suites) {
		return len(suites)
	}
	return v.ParallelSuites
}

// suitesShareVars reports whether a testsuite reads a var exported by a testcase
// of another testsuite: its text holds ".<testcase name of the other suite>.".
func suitesShareVars(suites []TestSuite) bool {
	for i := range suites {
		var text strings.Builder
		for k := range suites[i].TestCases {
			text.WriteString(testCaseText(&suites[i].TestCases[k]))
		}
		s := text.String()
		for j := range suites {
			if j == i {
				continue
			}
			for k := range suites[j].TestCases {
				name := suites[j].TestCases[k].Name
				if name != "" && strings.Contains(s, "."+name+".") {
					return true
				}
			}
		}
	}
	return false
}

// runTestSuitesParallel runs the testsuites n at a time. Each testsuite prints into
// its own buffer, written out whole and in file order; results stay at their index
// in v.Tests.TestSuites, so the reports keep the file order. The first error in file
// order is returned, as the sequential run would have returned it.
func (v *Venom) runTestSuitesParallel(ctx context.Context, n int) error {
	suites := v.Tests.TestSuites
	outputs := make([]bytes.Buffer, len(suites))
	errs := make([]error, len(suites))
	done := make([]chan struct{}, len(suites))
	for i := range done {
		done[i] = make(chan struct{})
	}

	sem := make(chan struct{}, n)
	go func() {
		for i := range suites {
			sem <- struct{}{}
			go func(i int) {
				defer func() { <-sem }()
				defer close(done[i])
				vc := *v
				buf := &outputs[i]
				vc.PrintFunc = func(format string, a ...interface{}) (int, error) {
					return fmt.Fprintf(buf, format, a...)
				}
				errs[i] = vc.runTimedTestSuite(ctx, &suites[i])
			}(i)
		}
	}()

	var first error
	for i := range suites {
		<-done[i]
		v.Print("%s", outputs[i].String())
		if errs[i] != nil {
			if first == nil {
				first = errs[i]
			}
			continue
		}
		v.countTestSuite(&suites[i])
	}
	return first
}
