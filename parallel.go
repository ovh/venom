package venom

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
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
