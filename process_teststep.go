package venom

import (
	"context"
	"fmt"
	"time"
)

//RunTestStep executes a venom testcase is a venom context
func (v *Venom) RunTestStep(ctx context.Context, e ExecutorRunner, ts *TestSuite, tc *TestCase, stepNumber int, step TestStep) interface{} {
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
		result, err = runTestStepExecutor(ctx, e, ts, step)
		if err != nil {
			// we save the failure only if it's the last attempt
			if retry == e.Retry() {
				failure := newFailure(*tc, stepNumber, "", err)
				tc.Failures = append(tc.Failures, *failure)
			}
			continue
		}

		Debug(ctx, "result: %+v", result)

		if h, ok := e.(executorWithDefaultAssertions); ok {
			assertRes = applyAssertions(result, *tc, stepNumber, step, h.GetDefaultAssertions())
		} else {
			assertRes = applyAssertions(result, *tc, stepNumber, step, nil)
		}

		mapResult := GetExecutorResult(result)
		tc.ComputedVars.AddAll(H(mapResult))

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

func runTestStepExecutor(ctx context.Context, e ExecutorRunner, ts *TestSuite, step TestStep) (interface{}, error) {
	ctx = context.WithValue(ctx, ContextKey("executor"), e.Name())

	if e.Timeout() == 0 {
		return e.Run(ctx, step, ts.WorkDir)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(e.Timeout())*time.Second)
	defer cancel()

	ch := make(chan interface{})
	cherr := make(chan error)
	go func(e ExecutorRunner, step TestStep) {
		result, err := e.Run(ctx, step, ts.WorkDir)
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
