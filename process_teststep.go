package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/gosimple/slug"
	"github.com/ovh/cds/sdk/interpolate"
)

type dumpFile struct {
	Variables H           `json:"variables"`
	TestStep  TestStep    `json:"step"`
	Result    interface{} `json:"result"`
}

//RunTestStep executes a venom testcase is a venom context
func (v *Venom) RunTestStep(ctx context.Context, e ExecutorRunner, tc *TestCase, ts *TestStepResult, stepNumber int, rangedIndex int, step TestStep) interface{} {
	ctx = context.WithValue(ctx, ContextKey("executor"), e.Name())

	var assertRes assertionsApplied
	var retry int
	var result interface{}

	for retry = 0; retry <= e.Retry() && !assertRes.ok; retry++ {
		if retry > 1 && !assertRes.ok {
			Debug(ctx, "Sleep %d, it's %d attempt", e.Delay(), retry)
			time.Sleep(time.Duration(e.Delay()) * time.Second)
		}

		var err error
		result, err = v.runTestStepExecutor(ctx, e, tc, ts, step)
		if err != nil {
			// we save the failure only if it's the last attempt
			if retry == e.Retry() {
				failure := newFailure(*tc, stepNumber, rangedIndex, "", err)
				ts.appendFailure(tc, *failure)
			}
			continue
		}

		Debug(ctx, "result of runTestStepExecutor: %+v", result)
		mapResult := GetExecutorResult(result)
		mapResultString, _ := DumpString(result)

		if v.Verbose >= 2 {
			fdump := dumpFile{
				Result:    result,
				TestStep:  step,
				Variables: AllVarsFromCtx(ctx),
			}
			output, err := json.MarshalIndent(fdump, "", " ")
			if err != nil {
				Error(ctx, "unable to marshal result: %v", err)
			}

			oDir := v.OutputDir
			if oDir == "" {
				oDir = "."
			}
			filename := path.Join(oDir, fmt.Sprintf("%s.%s.step.%d.%d.dump.json", slug.Make(StringVarFromCtx(ctx, "venom.testsuite.shortName")), slug.Make(tc.Name), stepNumber, rangedIndex))

			if err := os.WriteFile(filename, []byte(output), 0644); err != nil {
				return fmt.Errorf("Error while creating file %s: %v", filename, err)
			}
			tc.computedVerbose = append(tc.computedVerbose, fmt.Sprintf("writing %s", filename))
		}

		for ninfo, i := range e.Info() {
			info, err := interpolate.Do(i, mapResultString)
			if err != nil {
				Error(ctx, "unable to parse %q: %v", i, err)
				continue
			}
			if info == "" {
				continue
			}
			filename := StringVarFromCtx(ctx, "venom.testsuite.filename")
			lineNumber := findLineNumber(filename, tc.originalName, stepNumber, i, ninfo+1)
			if lineNumber > 0 {
				info += fmt.Sprintf(" (%s:%d)", filename, lineNumber)
			} else if tc.IsExecutor {
				filename = StringVarFromCtx(ctx, "venom.executor.filename")
				originalName := StringVarFromCtx(ctx, "venom.executor.name")
				lineNumber = findLineNumber(filename, originalName, stepNumber, i, ninfo+1)
				if lineNumber > 0 {
					info += fmt.Sprintf(" (%s:%d)", filename, lineNumber)
				}
			}
			Info(ctx, info)
			tc.computedInfo = append(tc.computedInfo, info)
		}

		if result == nil {
			Debug(ctx, "empty testcase, applying assertions on variables: %v", AllVarsFromCtx(ctx))
			assertRes = applyAssertions(AllVarsFromCtx(ctx), *tc, stepNumber, rangedIndex, step, nil)
		} else {
			if h, ok := e.(executorWithDefaultAssertions); ok {
				assertRes = applyAssertions(result, *tc, stepNumber, rangedIndex, step, h.GetDefaultAssertions())
			} else {
				assertRes = applyAssertions(result, *tc, stepNumber, rangedIndex, step, nil)
			}
		}

		tc.computedVars.AddAll(H(mapResult))

		if assertRes.ok {
			break
		}
		failures, err := testConditionalStatement(ctx, tc, e.RetryIf(), tc.computedVars, "")
		if err != nil {
			return fmt.Errorf("Error while evaluating retry condition: %v", err)
		}
		if len(failures) > 0 {
			failure := newFailure(*tc, stepNumber, rangedIndex, "", fmt.Errorf("retry conditions not fulfilled, skipping %d remaining retries", e.Retry()-retry))
			tc.Failures = append(tc.Failures, *failure)
			break
		}
	}

	ts.appendError(tc, assertRes.errors...)
	ts.appendFailure(tc, assertRes.failures...)
	if retry > 1 && (len(assertRes.failures) > 0 || len(assertRes.errors) > 0) {
		ts.appendFailure(tc, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
	}
	tc.Systemout.Value += assertRes.systemout
	tc.Systemerr.Value += assertRes.systemerr

	return result
}

func (v *Venom) runTestStepExecutor(ctx context.Context, e ExecutorRunner, tc *TestCase, ts *TestStepResult, step TestStep) (interface{}, error) {
	ctx = context.WithValue(ctx, ContextKey("executor"), e.Name())

	if e.Timeout() == 0 {
		if e.Type() == "user" {
			return v.RunUserExecutor(ctx, e, tc, ts, step)
		}
		return e.Run(ctx, step)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(e.Timeout())*time.Second)
	defer cancel()

	ch := make(chan interface{})
	cherr := make(chan error)
	go func(e ExecutorRunner, step TestStep) {
		var err error
		var result interface{}
		if e.Type() == "user" {
			result, err = v.RunUserExecutor(ctx, e, tc, ts, step)
		} else {
			result, err = e.Run(ctx, step)
		}
		if err != nil {
			cherr <- err
		} else {
			ch <- result
		}
	}(e, step)

	select {
	case err := <-cherr:
		return nil, err
	case result := <-ch:
		return result, nil
	case <-ctxTimeout.Done():
		return nil, fmt.Errorf("Timeout after %d second(s)", e.Timeout())
	}
}

//Append an error to a test step and its associated test case
func (ts *TestStepResult) appendError(tc *TestCase, failure ...Failure) {
	ts.Errors = append(ts.Errors, failure...)
	tc.Errors = append(tc.Errors, failure...)
}

//Append an assertion failure to a test step and its associated test case
func (ts *TestStepResult) appendFailure(tc *TestCase, failure ...Failure) {
	ts.Failures = append(ts.Failures, failure...)
	tc.Failures = append(tc.Failures, failure...)
}
