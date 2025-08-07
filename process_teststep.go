package venom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/ovh/venom/interpolate"
)

type dumpFile struct {
	Variables H           `json:"variables"`
	TestStep  TestStep    `json:"step"`
	Result    interface{} `json:"result"`
}

// RunTestStep executes a venom testcase is a venom context
func (v *Venom) RunTestStep(ctx context.Context, e ExecutorRunner, tc *TestCase, tsResult *TestStepResult, stepNumber int, rangedIndex int, step TestStep) {
	ctx = context.WithValue(ctx, ContextKey("executor"), e.Name())

	var assertRes AssertionsApplied
	var result interface{}

	for tsResult.Retries = 0; tsResult.Retries <= e.Retry() && !assertRes.OK; tsResult.Retries++ {
		if tsResult.Retries > 1 && !assertRes.OK {
			Debug(ctx, "Sleep %d, it's %d attempt", e.Delay(), tsResult.Retries)
			time.Sleep(time.Duration(e.Delay()) * time.Second)
		}

		var err error
		result, err = v.runTestStepExecutor(ctx, e, tc, tsResult, step)
		if err != nil {
			// we save the failure only if it's the last attempt
			if tsResult.Retries == e.Retry() {
				failure := newFailure(ctx, *tc, stepNumber, rangedIndex, "", err)
				tsResult.appendFailure(*failure)
			}
			continue
		}

		Debug(ctx, "result of executor: %s", HideSensitive(ctx, result))
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

			filename := filepath.Join(v.OutputDir, fmt.Sprintf("%s.%s.testcase.%d.step.%d.%d.dump.json", slug.Make(StringVarFromCtx(ctx, "venom.testsuite.shortName")), slug.Make(tc.Name), tc.number, stepNumber, rangedIndex))

			if err := os.WriteFile(filename, []byte(HideSensitive(ctx, string(output))), 0o644); err != nil {
				Error(ctx, "Error while creating file %s: %v", filename, err)
				return
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
			Info(ctx, info, nil)
			tsResult.ComputedInfo = append(tsResult.ComputedInfo, info)
		}

		if result == nil {
			Debug(ctx, "empty testcase, applying assertions on variables: %v", AllVarsFromCtx(ctx))
			assertRes = applyAssertions(ctx, AllVarsFromCtx(ctx), *tc, stepNumber, rangedIndex, step, nil)
		} else {
			if h, ok := e.(executorWithDefaultAssertions); ok {
				assertRes = applyAssertions(ctx, result, *tc, stepNumber, rangedIndex, step, h.GetDefaultAssertions())
			} else {
				assertRes = applyAssertions(ctx, result, *tc, stepNumber, rangedIndex, step, nil)
			}
		}

		tsResult.AssertionsApplied = assertRes
		tsResult.ComputedVars.AddAll(H(mapResult))

		if !assertRes.OK && len(assertRes.errors) > 0 {
			if e.Type() == "user" {
				generateFailureLinkForUserExecutor(ctx, result, tsResult, tc)
			} else {
				generateFailureLink(ctx, result, tsResult)
			}
		}

		if assertRes.OK {
			break
		}
		failures, err := testConditionalStatement(ctx, tc, e.RetryIf(), tsResult.ComputedVars, "")
		if err != nil {
			tsResult.appendError(fmt.Errorf("Error while evaluating retry condition: %v", err))
			break
		}
		if len(failures) > 0 {
			failure := newFailure(ctx, *tc, stepNumber, rangedIndex, "", fmt.Errorf("retry conditions not fulfilled, skipping %d remaining retries", e.Retry()-tsResult.Retries))
			tsResult.Errors = append(tsResult.Errors, *failure)
			break
		}
	}

	if tsResult.Retries > 1 && len(assertRes.errors) > 0 {
		tsResult.appendFailure(Failure{Value: fmt.Sprintf("It's a failure after %d attempts", tsResult.Retries)})
	}

	if len(assertRes.errors) > 0 {
		tsResult.appendFailure(assertRes.errors...)
	}

	tsResult.Systemerr += assertRes.systemerr + "\n"
	tsResult.Systemout += assertRes.systemout + "\n"
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

func generateFailureLink(ctx context.Context, result interface{}, tsResult *TestStepResult) {
	failureLinkHeader := StringVarFromCtx(ctx, "venom.failure_link_header")
	if failureLinkHeader == "" {
		return
	}

	failureLinkTemplate := StringVarFromCtx(ctx, "venom.failure_link_template")
	if failureLinkTemplate == "" {
		return
	}

	headerValue := extractHeaderValue(ctx, result, failureLinkHeader)
	if headerValue == "" {
		Warn(ctx, "Response header %s not found in response; skipping failure link", failureLinkHeader)
		return
	}

	tsResult.FailureLink = strings.ReplaceAll(failureLinkTemplate, "{{header}}", headerValue)
}

func extractHeaderValue(ctx context.Context, result interface{}, headerName string) string {
	if strResult, ok := result.(string); ok {
		return strResult
	}

	if mapResult, ok := result.(map[string]interface{}); ok {
		if headers, exists := mapResult["headers"]; exists {
			if headersMap, ok := headers.(map[string]interface{}); ok {
				if headerValue, exists := headersMap[headerName]; exists {
					return fmt.Sprintf("%v", headerValue)
				}
			}
		}
	} else {
		val := reflect.ValueOf(result)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if val.Kind() == reflect.Struct {
			headersField := val.FieldByName("Headers")
			if headersField.IsValid() && headersField.Kind() == reflect.Map {
				headerValue := headersField.MapIndex(reflect.ValueOf(headerName))
				if headerValue.IsValid() {
					return fmt.Sprintf("%v", headerValue.Interface())
				}
			}
		}
	}

	return ""
}

func generateFailureLinkForUserExecutor(ctx context.Context, result interface{}, tsResult *TestStepResult, tc *TestCase) {
	failureLinkHeader := StringVarFromCtx(ctx, "venom.failure_link_header")
	if failureLinkHeader == "" {
		return
	}

	failureLinkTemplate := StringVarFromCtx(ctx, "venom.failure_link_template")
	if failureLinkTemplate == "" {
		return
	}

	// Try processed result first
	if mapResult, ok := result.(map[string]interface{}); ok {
		for _, value := range mapResult {
			if stepResultMap, ok := value.(map[string]interface{}); ok {
				if headers, exists := stepResultMap["headers"]; exists {
					if headersMap, ok := headers.(map[string]interface{}); ok {
						if headerValue, exists := headersMap[failureLinkHeader]; exists {
							tsResult.FailureLink = strings.ReplaceAll(failureLinkTemplate, "{{header}}", fmt.Sprintf("%v", headerValue))
							return
						}
					}
				}
			}
		}
	}

	// Check internal step results
	for _, internalStepResult := range tc.TestStepResults {
		if internalStepResult.Raw != nil {
			if headerValue := extractHeaderValue(ctx, internalStepResult.Raw, failureLinkHeader); headerValue != "" {
				tsResult.FailureLink = strings.ReplaceAll(failureLinkTemplate, "{{header}}", headerValue)
				return
			}
		}

		if internalStepResult.ComputedVars != nil {
			headerKey := fmt.Sprintf("result.headers.%s", failureLinkHeader)
			if headerValue, exists := internalStepResult.ComputedVars[headerKey]; exists {
				if headerValueStr, ok := headerValue.(string); ok {
					tsResult.FailureLink = strings.ReplaceAll(failureLinkTemplate, "{{header}}", headerValueStr)
					return
				}
			}
		}
	}

	Warn(ctx, "Response header %s not found in user executor results; skipping failure link", failureLinkHeader)
}
