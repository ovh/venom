package venom

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom/assertions"
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
func applyChecks(executorResult *ExecutorResult, tc TestCase, stepNumber int, step TestStep, defaultAssertions *StepAssertions) assertionsApplied {
	res := applyAssertions(*executorResult, tc, stepNumber, step, defaultAssertions)
	if !res.ok {
		return res
	}

	resExtract := applyExtracts(executorResult, step)

	res.errors = append(res.errors, resExtract.errors...)
	res.failures = append(res.failures, resExtract.failures...)
	res.ok = resExtract.ok

	return res
}

func applyAssertions(executorResult ExecutorResult, tc TestCase, stepNumber int, step TestStep, defaultAssertions *StepAssertions) assertionsApplied {
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
		errs, fails := check(tc, stepNumber, assertion, executorResult)
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

func check(tc TestCase, stepNumber int, assertion string, executorResult ExecutorResult) (*Failure, *Failure) {
	assert := splitAssertion(assertion)
	if len(assert) < 2 {
		return nil, newFailure(tc, stepNumber, assertion, errors.New("syntax error"), executorResult)
	}

	actual := executorResult[assert[0]]
	f, ok := assertions.Get(assert[1])
	if !ok {
		return nil, newFailure(tc, stepNumber, assertion, errors.New("assertion not supported"), executorResult)
	}

	args := make([]interface{}, len(assert[2:]))
	for i, v := range assert[2:] {
		var err error
		args[i], err = stringToType(v, actual)
		if err != nil {
			return nil, newFailure(tc, stepNumber, assertion, fmt.Errorf("mismatched type between '%v' and '%v': %v", assert[0], v, err), executorResult)
		}
	}

	if err := f(actual, args...); err != nil {
		return nil, newFailure(tc, stepNumber, assertion, err, executorResult)
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

func findLineNumber(filename, testcase string, stepNumber int, assertion string) (int, error) {
	countLine := 0
	file, err := os.Open(filename)
	if err != nil {
		return countLine, err
	}
	defer file.Close()

	lineFound := false
	testcaseFound := false
	commentBlock := false
	countStep := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		countLine++
		line := strings.Trim(scanner.Text(), " ")
		if strings.HasPrefix(line, "/*") {
			commentBlock = true
			continue
		}
		if strings.HasPrefix(line, "*/") {
			commentBlock = false
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || commentBlock {
			continue
		}
		if !testcaseFound && strings.Contains(line, testcase) {
			testcaseFound = true
			continue
		}
		if testcaseFound && countStep <= stepNumber && strings.Contains(line, "type") {
			countStep++
			continue
		}
		if testcaseFound && countStep > stepNumber && strings.Contains(line, assertion) {
			lineFound = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return countLine, err
	}

	if !lineFound {
		return 0, nil
	}

	return countLine, nil
}
