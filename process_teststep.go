package venom

import (
	"fmt"
	"time"
)

// RunTestStep executes a venom testcase is a venom context
func (v *Venom) RunTestStep(ctx TestContext, tcName string, stepNumber int, step TestStep, l Logger) (ExecutorResult, assertionsApplied, error) {
	var assertRes assertionsApplied
	var retry int
	var result ExecutorResult

	e, err := v.getExecutorModule(step)
	if err != nil {
		return nil, assertRes, err
	}

	for retry = 0; retry <= e.retry && !assertRes.ok; retry++ {
		if retry > 1 && !assertRes.ok {
			l.Debugf("Sleep %d, it's %d attempt", e.delay, retry)
			time.Sleep(time.Duration(e.delay) * time.Second)
		}

		var err error
		result, err = v.runTestStepExecutor(ctx, step, l, e)
		if err != nil {
			if retry == e.retry {
				return nil, assertRes, err
			}
			continue
		}
		defaultAsserts, err := v.getDefaultAssertions(ctx, e)
		if err != nil {
			l.Warnf("unable to get default assertions: %v", err)
		}
		assertRes = applyChecks(result, tcName, stepNumber, step, defaultAsserts)

		if assertRes.ok {
			break
		}
	}

	return result, assertRes, nil
}

func (v *Venom) getDefaultAssertions(ctx TestContext, e *executorModule) (*StepAssertions, error) {
	return e.GetDefaultAssertions(ctx)
}

func (v *Venom) runTestStepExecutor(ctx TestContext, step TestStep, l Logger, e *executorModule) (ExecutorResult, error) {
	if e.timeout > 0 {
		defer ctx.WithTimeout(time.Duration(e.timeout) * time.Second)
	}
	//TODO: Handle ts.workdir to setup the module execution is the right working directory
	//TODO: Handle an override of the logger
	runner, err := e.New(ctx, v)
	if err != nil {
		return nil, err
	}

	ch := make(chan ExecutorResult)
	cherr := make(chan error)
	go func() {
		result, err := runner.Run(ctx, l, step)
		if err != nil {
			cherr <- err
		} else {
			ch <- result
		}
	}()

	select {
	case err := <-cherr:
		return nil, err
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("Timeout after %d second(s)", e.timeout)
	}
}
