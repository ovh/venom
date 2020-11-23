package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/gosimple/slug"
	"github.com/ovh/cds/sdk/interpolate"

	"github.com/ovh/venom/executors"
)

type dumpFile struct {
	Variables H           `json:"variables"`
	TestStep  TestStep    `json:"step"`
	Result    interface{} `json:"result"`
}

//RunTestStep executes a venom testcase is a venom context
func (v *Venom) RunTestStep(ctx context.Context, e ExecutorRunner, tc *TestCase, stepNumber int, step TestStep) interface{} {
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
		result, err = v.runTestStepExecutor(ctx, e, step)
		if err != nil {
			// we save the failure only if it's the last attempt
			if retry == e.Retry() {
				failure := newFailure(*tc, stepNumber, "", err)
				tc.Failures = append(tc.Failures, *failure)
			}
			continue
		}

		Debug(ctx, "result of runTestStepExecutor: %+v", result)
		mapResult := GetExecutorResult(result)
		mapResultString, _ := executors.DumpString(result)

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
			filename := path.Join(oDir, fmt.Sprintf("%s.%s.step.%d.dump.json", slug.Make(StringVarFromCtx(ctx, "venom.testsuite.shortName")), slug.Make(tc.Name), stepNumber))

			if err := ioutil.WriteFile(filename, []byte(output), 0644); err != nil {
				return fmt.Errorf("Error while creating file %s: %v", filename, err)
			}
			tc.computedVerbose = append(tc.computedVerbose, fmt.Sprintf("writing %s", filename))
		}

		for _, i := range e.Info() {
			info, err := interpolate.Do(i, mapResultString)
			if err != nil {
				Error(ctx, "unable to parse %q: %v", i, err)
				continue
			}
			if info == "" {
				continue
			}
			filename := StringVarFromCtx(ctx, "venom.testsuite.filename")
			info += fmt.Sprintf(" (%s:%d)", filename, findLineNumber(filename, tc.originalName, stepNumber, i))
			Info(ctx, info)
			tc.computedInfo = append(tc.computedInfo, info)
		}

		if h, ok := e.(executorWithDefaultAssertions); ok {
			assertRes = applyAssertions(result, *tc, stepNumber, step, h.GetDefaultAssertions())
		} else {
			assertRes = applyAssertions(result, *tc, stepNumber, step, nil)
		}

		tc.computedVars.AddAll(H(mapResult))

		if assertRes.ok {
			break
		}
	}

	tc.Errors = append(tc.Errors, assertRes.errors...)
	tc.Failures = append(tc.Failures, assertRes.failures...)
	if retry > 1 && (len(assertRes.failures) > 0 || len(assertRes.errors) > 0) {
		tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
	}
	tc.Systemout.Value += assertRes.systemout
	tc.Systemerr.Value += assertRes.systemerr

	return result
}

func (v *Venom) runTestStepExecutor(ctx context.Context, e ExecutorRunner, step TestStep) (interface{}, error) {
	ctx = context.WithValue(ctx, ContextKey("executor"), e.Name())

	if e.Timeout() == 0 {
		if e.Type() == "user" {
			return v.RunUserExecutor(ctx, e.GetExecutor().(UserExecutor), step)
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
			result, err = v.RunUserExecutor(ctx, e.GetExecutor().(UserExecutor), step)
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
