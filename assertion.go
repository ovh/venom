package venom

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mitchellh/mapstructure"
	"github.com/smartystreets/assertions"
)

type testingT struct {
	ErrorS []string
}

func (t *testingT) Error(args ...interface{}) {
	for _, a := range args {
		switch v := a.(type) {
		case string:
			t.ErrorS = append(t.ErrorS, v)
		default:
			t.ErrorS = append(t.ErrorS, fmt.Sprintf("%s", v))
		}
	}
}

type assertionsApplied struct {
	ok        bool
	errors    []Failure
	failures  []Failure
	systemout string
	systemerr string
}

// applyChecks apply checks on result, return true if all assertions are OK, false otherwise
func applyChecks(executorResult *ExecutorResult, ts TestSuite, tc TestCase, stepNumber int, step TestStep, defaultAssertions *StepAssertions) assertionsApplied {
	res := applyAssertions(*executorResult, ts, tc, stepNumber, step, defaultAssertions)
	if !res.ok {
		return res
	}

	resExtract := applyExtracts(executorResult, step)

	res.errors = append(res.errors, resExtract.errors...)
	res.failures = append(res.failures, resExtract.failures...)
	res.ok = resExtract.ok

	return res
}

func applyAssertions(executorResult ExecutorResult, ts TestSuite, tc TestCase, stepNumber int, step TestStep, defaultAssertions *StepAssertions) assertionsApplied {
	var sa StepAssertions
	var errors []Failure
	var failures []Failure
	var systemerr, systemout string

	if err := mapstructure.Decode(step, &sa); err != nil {
		return assertionsApplied{
			false,
			[]Failure{{Value: RemoveNotPrintableChar(fmt.Sprintf("error decoding assertions: %s", err))}},
			failures,
			systemout,
			systemerr,
		}
	}

	if len(sa.Assertions) == 0 && defaultAssertions != nil {
		sa = *defaultAssertions
	}

	isOK := true
	for _, assertion := range sa.Assertions {
		errs, fails := check(ts, tc, stepNumber, assertion, executorResult)
		if errs != nil {
			errors = append(errors, *errs)
			isOK = false
		}
		if fails != nil {
			failures = append(failures, *fails)
			isOK = false
		}
	}

	if _, ok := executorResult["result.systemerr"]; ok {
		systemerr = fmt.Sprintf("%v", executorResult["result.systemerr"])
	}

	if _, ok := executorResult["result.systemout"]; ok {
		systemout = fmt.Sprintf("%v", executorResult["result.systemout"])
	}

	return assertionsApplied{isOK, errors, failures, systemout, systemerr}
}

func getLastValidResultFromPath(path string, r ExecutorResult) (string, string) {
	tokens := strings.Split(path, ".")

	for i := len(tokens); i >= 0; i-- {
		newPath := strings.Join(tokens[:i], ".")
		if res, ok := r[newPath]; ok {
			encodedData, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return RemoveNotPrintableChar(fmt.Sprintf("%+v", res)), RemoveNotPrintableChar(newPath)
			}
			return RemoveNotPrintableChar(string(encodedData)), RemoveNotPrintableChar(newPath)
		}
	}

	encodedData, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", r), "."
	}
	return string(encodedData), "."
}

func check(ts TestSuite, tc TestCase, stepNumber int, assertion string, executorResult ExecutorResult) (*Failure, *Failure) {
	assert := splitAssertion(assertion)
	if len(assert) < 2 {
		return &Failure{
			Value: fmt.Sprintf(
				color.YellowString(
					"Failure in %q\nIn test case %q, at step %d\nInvalid assertion %q length should be greater than 2\n",
					ts.Filename,
					tc.Name,
					stepNumber,
					RemoveNotPrintableChar(assertion),
				),
			),
		}, nil
	}

	actual, ok := executorResult[assert[0]]
	if !ok {
		if assert[1] == "ShouldNotExist" {
			return nil, nil
		}

		data, path := getLastValidResultFromPath(assert[0], executorResult)
		return &Failure{
			Value: fmt.Sprintf(
				color.YellowString(
					"Failure in %q\nIn test case %q, at step %d\nCould not access %q in assertion %q.\nThis is what we have at %q:\n",
					ts.Filename,
					tc.Name,
					stepNumber,
					RemoveNotPrintableChar(assert[0]),
					RemoveNotPrintableChar(assertion),
					path,
				) + data + "\n",
			),
		}, nil
	} else if assert[1] == "ShouldNotExist" {
		paths := strings.Split(assert[0], ".")
		if len(paths) > 0 {
			paths = paths[:len(paths)-1]
		}
		data, path := getLastValidResultFromPath(strings.Join(paths, "."), executorResult)
		return &Failure{
			Value: fmt.Sprintf(
				color.YellowString(
					"Failure in %q\nIn test case %q, at step %d\nIn assertion %q, key %q should not exist.\nThis is what we have at %q:\n",
					ts.Filename,
					tc.Name,
					stepNumber,
					RemoveNotPrintableChar(assertion),
					RemoveNotPrintableChar(assert[0]),
					path,
				) + data + "\n",
			),
		}, nil
	}

	f, ok := assertMap[assert[1]]
	if !ok {
		return &Failure{
			Value: fmt.Sprintf(
				color.YellowString(
					"Failure in %q\nIn test case %q, at step %d\nMethod %q in assertion %q is not supported\n",
					ts.Filename,
					tc.Name,
					stepNumber,
					RemoveNotPrintableChar(assert[1]),
					RemoveNotPrintableChar(assertion),
				),
			),
		}, nil
	}
	args := make([]interface{}, len(assert[2:]))
	for i, v := range assert[2:] { // convert []string to []interface for assertions.func()...
		args[i], _ = stringToType(v, actual)
	}

	out := f(actual, args...)

	if out != "" {
		var prefix string
		if stepNumber >= 0 {
			prefix = fmt.Sprintf(
				color.YellowString(
					"Failure in %q\nIn test case %q, at step %d\nAssertion %q failed",
					ts.Filename,
					tc.Name,
					stepNumber,
					RemoveNotPrintableChar(assertion),
				),
			)
		} else {
			// venom used as lib
			prefix = RemoveNotPrintableChar(fmt.Sprintf("assertion: %s", assertion))
		}
		return nil, &Failure{Value: prefix + "\n" + RemoveNotPrintableChar(out) + "\n", Result: executorResult}
	}
	return nil, nil
}

// splitAssertion splits the assertion string a, with support
// for quoted arguments.
func splitAssertion(a string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}
	m := strings.FieldsFunc(a, f)
	for i, e := range m {
		first, _ := utf8.DecodeRuneInString(e)
		last, _ := utf8.DecodeLastRuneInString(e)
		if unicode.In(first, unicode.Quotation_Mark) && first == last {
			m[i] = string([]rune(e)[1 : utf8.RuneCountInString(e)-1])
		}
	}
	return m
}

// assertMap contains list of assertions func
var assertMap = map[string]func(actual interface{}, expected ...interface{}) string{
	// "ShouldNotExist" see func check
	"ShouldEqual":                  assertions.ShouldEqual,
	"ShouldNotEqual":               assertions.ShouldNotEqual,
	"ShouldAlmostEqual":            assertions.ShouldAlmostEqual,
	"ShouldNotAlmostEqual":         assertions.ShouldNotAlmostEqual,
	"ShouldResemble":               assertions.ShouldResemble,
	"ShouldNotResemble":            assertions.ShouldNotResemble,
	"ShouldPointTo":                assertions.ShouldPointTo,
	"ShouldNotPointTo":             assertions.ShouldNotPointTo,
	"ShouldBeNil":                  assertions.ShouldBeNil,
	"ShouldNotBeNil":               assertions.ShouldNotBeNil,
	"ShouldBeTrue":                 assertions.ShouldBeTrue,
	"ShouldBeFalse":                assertions.ShouldBeFalse,
	"ShouldBeZeroValue":            assertions.ShouldBeZeroValue,
	"ShouldBeGreaterThan":          assertions.ShouldBeGreaterThan,
	"ShouldBeGreaterThanOrEqualTo": assertions.ShouldBeGreaterThanOrEqualTo,
	"ShouldBeLessThan":             assertions.ShouldBeLessThan,
	"ShouldBeLessThanOrEqualTo":    assertions.ShouldBeLessThanOrEqualTo,
	"ShouldBeBetween":              assertions.ShouldBeBetween,
	"ShouldNotBeBetween":           assertions.ShouldNotBeBetween,
	"ShouldBeBetweenOrEqual":       assertions.ShouldBeBetweenOrEqual,
	"ShouldNotBeBetweenOrEqual":    assertions.ShouldNotBeBetweenOrEqual,
	"ShouldContain":                assertions.ShouldContain,
	"ShouldNotContain":             assertions.ShouldNotContain,
	"ShouldContainKey":             assertions.ShouldContainKey,
	"ShouldNotContainKey":          assertions.ShouldNotContainKey,
	"ShouldBeIn":                   assertions.ShouldBeIn,
	"ShouldNotBeIn":                assertions.ShouldNotBeIn,
	"ShouldBeEmpty":                assertions.ShouldBeEmpty,
	"ShouldNotBeEmpty":             assertions.ShouldNotBeEmpty,
	"ShouldHaveLength":             assertions.ShouldHaveLength,
	"ShouldStartWith":              assertions.ShouldStartWith,
	"ShouldNotStartWith":           assertions.ShouldNotStartWith,
	"ShouldEndWith":                assertions.ShouldEndWith,
	"ShouldNotEndWith":             assertions.ShouldNotEndWith,
	"ShouldBeBlank":                assertions.ShouldBeBlank,
	"ShouldNotBeBlank":             assertions.ShouldNotBeBlank,
	"ShouldContainSubstring":       ShouldContainSubstring,
	"ShouldNotContainSubstring":    assertions.ShouldNotContainSubstring,
	"ShouldEqualWithout":           assertions.ShouldEqualWithout,
	"ShouldEqualTrimSpace":         assertions.ShouldEqualTrimSpace,
	"ShouldHappenBefore":           assertions.ShouldHappenBefore,
	"ShouldHappenOnOrBefore":       assertions.ShouldHappenOnOrBefore,
	"ShouldHappenAfter":            assertions.ShouldHappenAfter,
	"ShouldHappenOnOrAfter":        assertions.ShouldHappenOnOrAfter,
	"ShouldHappenBetween":          assertions.ShouldHappenBetween,
	"ShouldHappenOnOrBetween":      assertions.ShouldHappenOnOrBetween,
	"ShouldNotHappenOnOrBetween":   assertions.ShouldNotHappenOnOrBetween,
	"ShouldHappenWithin":           assertions.ShouldHappenWithin,
	"ShouldNotHappenWithin":        assertions.ShouldNotHappenWithin,
	"ShouldBeChronological":        assertions.ShouldBeChronological,
}

// ShouldContainSubstring receives exactly more than 2 string parameters and ensures that the first contains the second as a substring.
func ShouldContainSubstring(actual interface{}, expected ...interface{}) string {
	if len(expected) == 1 {
		return assertions.ShouldContainSubstring(actual, expected...)
	}

	var arg string
	for _, e := range expected {
		arg += fmt.Sprintf("%v ", e)
	}
	return assertions.ShouldContainSubstring(actual, strings.TrimSpace(arg))
}

func stringToType(val string, valType interface{}) (interface{}, error) {
	switch valType.(type) {
	case bool:
		return strconv.ParseBool(val)
	case string:
		return val, nil
	case int:
		return strconv.Atoi(val)
	case int8:
		return strconv.ParseInt(val, 10, 8)
	case int16:
		return strconv.ParseInt(val, 10, 16)
	case int32:
		return strconv.ParseInt(val, 10, 32)
	case int64:
		return strconv.ParseInt(val, 10, 64)
	case uint:
		newVal, err := strconv.Atoi(val)
		return uint(newVal), err
	case uint8:
		return strconv.ParseUint(val, 10, 8)
	case uint16:
		return strconv.ParseUint(val, 10, 16)
	case uint32:
		return strconv.ParseUint(val, 10, 32)
	case uint64:
		return strconv.ParseUint(val, 10, 64)
	case float32:
		iVal, err := strconv.ParseFloat(val, 32)
		return float32(iVal), err
	case float64:
		iVal, err := strconv.ParseFloat(val, 64)
		return float64(iVal), err
	case time.Time:
		return time.Parse(time.RFC3339, val)
	case time.Duration:
		return time.ParseDuration(val)
	}
	return val, nil
}
