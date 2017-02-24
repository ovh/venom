package venom

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
)

func runTestStep(ctx context.Context, e *executorWrap, ts *TestSuite, tc *TestCase, step TestStep, templater *Templater, aliases map[string]string, l *log.Entry, detailsLevel string) {

	var isOK bool
	var errors []Failure
	var failures []Failure
	var systemerr, systemout string

	var retry int

	for retry = 0; retry <= e.retry && !isOK; retry++ {
		if retry > 1 && !isOK {
			log.Debugf("Sleep %d, it's %d attempt", e.delay, retry)
			time.Sleep(time.Duration(e.delay) * time.Second)
		}

		result, err := runTestStepExecutor(ctx, e, ts, step, templater, aliases, l)

		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: err.Error()})
			continue
		}

		ts.Templater.Add(tc.Name, result)

		log.Debugf("result:%+v", ts.Templater)

		if h, ok := e.executor.(executorWithDefaultAssertions); ok {
			isOK, errors, failures, systemout, systemerr = applyChecks(result, step, h.GetDefaultAssertions(), l)
		} else {
			isOK, errors, failures, systemout, systemerr = applyChecks(result, step, nil, l)
		}
		if isOK {
			break
		}
	}
	tc.Errors = append(tc.Errors, errors...)
	tc.Failures = append(tc.Failures, failures...)
	if retry > 1 && (len(failures) > 0 || len(errors) > 0) {
		tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
	}
	tc.Systemout.Value += systemout
	tc.Systemerr.Value += systemerr
}

func runTestStepExecutor(ctx context.Context, e *executorWrap, ts *TestSuite, step TestStep, templater *Templater, aliases map[string]string, l *log.Entry) (ExecutorResult, error) {
	if e.timeout == 0 {
		return e.executor.Run(ctx, l, aliases, step)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(e.timeout)*time.Second)
	defer cancel()

	ch := make(chan ExecutorResult)
	cherr := make(chan error)
	go func(e *executorWrap, step TestStep, l *log.Entry) {
		result, err := e.executor.Run(ctxTimeout, l, aliases, step)
		cherr <- err
		ch <- result
	}(e, step, l)

	select {
	case err := <-cherr:
		return nil, err
	case result := <-ch:
		return result, nil
	case <-ctxTimeout.Done():
		return nil, fmt.Errorf("Timeout after %d second(s)", e.timeout)
	}
}
