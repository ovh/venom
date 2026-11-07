package venom

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sleepExecutor sleeps for "ms" milliseconds and returns {"foo": "baz"}.
type sleepExecutor struct {
	running, peak int32
}

func (s *sleepExecutor) Run(_ context.Context, step TestStep) (interface{}, error) {
	n := atomic.AddInt32(&s.running, 1)
	for {
		p := atomic.LoadInt32(&s.peak)
		if n <= p || atomic.CompareAndSwapInt32(&s.peak, p, n) {
			break
		}
	}
	defer atomic.AddInt32(&s.running, -1)
	ms, _ := step.IntValue("ms")
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return map[string]interface{}{"foo": "baz"}, nil
}

var testLoggerOnce sync.Once

func runSuite(t *testing.T, parallel int, suite string) (*Venom, *sleepExecutor, string, time.Duration) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "suite.yml")
	require.NoError(t, os.WriteFile(file, []byte(suite), 0o644))

	testLoggerOnce.Do(func() { InitTestLogger(t) })
	v := New()
	ex := &sleepExecutor{}
	v.RegisterExecutorBuiltin("sleeper", ex)
	v.Parallel = parallel
	v.OutputDir = dir
	var out bytes.Buffer
	var mu sync.Mutex
	v.PrintFunc = func(format string, a ...interface{}) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return fmt.Fprintf(&out, format, a...)
	}
	ctx := context.Background()
	require.NoError(t, v.Parse(ctx, []string{file}))
	start := time.Now()
	require.NoError(t, v.Process(ctx, []string{file}))
	return v, ex, out.String(), time.Since(start)
}

const threeSleepers = `name: trio
testcases:
- name: one
  steps:
  - type: sleeper
    ms: 400
- name: two
  steps:
  - type: sleeper
    ms: 400
- name: three
  steps:
  - type: sleeper
    ms: 400
`

func TestParallelTestCases(t *testing.T) {
	v, ex, out, d := runSuite(t, 3, threeSleepers)
	assert.Less(t, d, 1000*time.Millisecond, "three 400ms testcases with --parallel=3")
	assert.Equal(t, int32(3), atomic.LoadInt32(&ex.peak))
	ts := v.Tests.TestSuites[0]
	require.Len(t, ts.TestCases, 3)
	for i, name := range []string{"one", "two", "three"} {
		assert.Equal(t, name, ts.TestCases[i].Name)
		assert.Equal(t, StatusPass, ts.TestCases[i].Status)
	}
	assert.Equal(t, StatusPass, v.Tests.Status)
	// console report keeps the testcase order, one whole line per testcase
	iOne, iTwo, iThree := strings.Index(out, "• one"), strings.Index(out, "• two"), strings.Index(out, "• three")
	assert.True(t, iOne >= 0 && iOne < iTwo && iTwo < iThree, out)
	for _, l := range strings.Split(strings.TrimSpace(out), "\n")[1:] {
		assert.Contains(t, l, StatusPass, "line %q", l)
	}
}

func TestSequentialByDefault(t *testing.T) {
	_, ex, _, d := runSuite(t, 1, threeSleepers)
	assert.GreaterOrEqual(t, d, 1200*time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&ex.peak))
}

const dependentSuite = `name: dependent
testcases:
- name: first
  steps:
  - type: sleeper
    ms: 300
    vars:
      got:
        from: foo
- name: second
  steps:
  - type: sleeper
    ms: 300
    assertions:
    - foo ShouldEqual {{.first.got}}
`

func TestDependentTestCasesRunSequentially(t *testing.T) {
	v, ex, _, d := runSuite(t, 4, dependentSuite)
	assert.True(t, dependsOnEarlierTestCase(&v.Tests.TestSuites[0]))
	assert.Equal(t, int32(1), atomic.LoadInt32(&ex.peak))
	assert.GreaterOrEqual(t, d, 600*time.Millisecond)
	assert.Equal(t, StatusPass, v.Tests.Status, "{{.first.got}} must resolve to baz")
}
