package venom

import (
	"bufio"
	"context"
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

type AssertionsApplied struct {
	OK         bool `json:"ok" yml:"-"`
	errors     []Failure
	systemout  string
	systemerr  string
	Assertions []AssertionApplied `json:"assertions" yml:"-"`
}
type AssertionApplied struct {
	Assertion Assertion `json:"assertion" yml:"-"`
	IsOK      bool      `json:"isOK" yml:"-"`
}

func applyAssertions(ctx context.Context, r interface{}, tc TestCase, stepNumber int, rangedIndex int, step TestStep, defaultAssertions *StepAssertions) AssertionsApplied {
	var sa StepAssertions
	var errors []Failure
	var systemerr, systemout string

	if err := mapstructure.Decode(step, &sa); err != nil {
		return AssertionsApplied{
			OK:        false,
			errors:    []Failure{{Value: RemoveNotPrintableChar(fmt.Sprintf("error decoding assertions: %s", err))}},
			systemout: systemout,
			systemerr: systemerr,
		}
	}

	if len(sa.Assertions) == 0 && defaultAssertions != nil {
		sa = *defaultAssertions
	}

	executorResult := GetExecutorResult(r)

	isOK := true
	assertions := []AssertionApplied{}
	for _, assertion := range sa.Assertions {
		errs := check(ctx, tc, stepNumber, rangedIndex, assertion, executorResult)
		isAssertionOK := true
		if errs != nil {
			errors = append(errors, *errs)
			isOK = false
			isAssertionOK = false
		}
		assertions = append(assertions, AssertionApplied{
			Assertion: assertion,
			IsOK:      isAssertionOK,
		})
	}

	if _, ok := executorResult["result.systemerr"]; ok {
		systemerr = fmt.Sprintf("%v", executorResult["result.systemerr"])
	}

	if _, ok := executorResult["result.systemout"]; ok {
		systemout = fmt.Sprintf("%v", executorResult["result.systemout"])
	}

	return AssertionsApplied{
		OK:         isOK,
		errors:     errors,
		systemerr:  systemerr,
		systemout:  systemout,
		Assertions: assertions,
	}
}

type assertion struct {
	Actual   interface{}
	Func     assertions.AssertFunc
	Args     []interface{}
	Required bool
}

func parseAssertions(ctx context.Context, s string, input interface{}) (*assertion, error) {
	dump, err := Dump(input)
	if err != nil {
		return nil, errors.New("assertion syntax error")
	}
	assert := splitAssertion(s)
	if len(assert) < 2 {
		return nil, errors.New("assertion syntax error")
	}
	actual := dump[assert[0]]

	// "Must" assertions use same tests as "Should" ones, only the flag changes
	required := false
	if strings.HasPrefix(assert[1], "Must") {
		required = true
		assert[1] = strings.Replace(assert[1], "Must", "Should", 1)
	}

	f, ok := assertions.Get(assert[1])
	if !ok {
		return nil, errors.New("assertion not supported")
	}
	args := make([]interface{}, len(assert[2:]))
	for i, v := range assert[2:] {
		var err error
		args[i], err = stringToType(v, actual)
		if err != nil {
			return nil, fmt.Errorf("mismatched type between '%v' and '%v': %v", assert[0], v, err)
		}
	}
	return &assertion{
		Actual:   actual,
		Func:     f,
		Args:     args,
		Required: required,
	}, nil
}

// check selects the correct assertion function to call depending on typing provided by user
func check(ctx context.Context, tc TestCase, stepNumber int, rangedIndex int, assertion Assertion, r interface{}) *Failure {
	var errs *Failure
	switch t := assertion.(type) {
	case string:
		errs = checkString(ctx, tc, stepNumber, rangedIndex, assertion.(string), r)
	case map[string]interface{}:
		errs = checkBranch(ctx, tc, stepNumber, rangedIndex, assertion.(map[string]interface{}), r)
	default:
		errs = newFailure(ctx, tc, stepNumber, rangedIndex, "", fmt.Errorf("unsupported assertion format: %v", t))
	}
	return errs
}

// checkString evaluate a complex assertion containing logical operators
// it recursively calls checkAssertion for each operand
func checkBranch(ctx context.Context, tc TestCase, stepNumber int, rangedIndex int, branch map[string]interface{}, r interface{}) *Failure {
	// Extract logical operator
	if len(branch) != 1 {
		return newFailure(ctx, tc, stepNumber, rangedIndex, "", fmt.Errorf("expected exactly 1 logical operator but %d were provided", len(branch)))
	}
	var operator string
	for k := range branch {
		operator = k
	}

	// Extract logical operands
	var operands []interface{}
	switch t := branch[operator].(type) {
	case []interface{}:
		operands = branch[operator].([]interface{})
	default:
		return newFailure(ctx, tc, stepNumber, rangedIndex, "", fmt.Errorf("expected %s operands to be an []interface{}, got %v", operator, t))
	}
	if len(operands) == 0 {
		return nil
	}

	// Evaluate assertions (operands)
	var results []string
	assertionsCount := len(operands)
	assertionsSuccess := 0
	for _, assertion := range operands {
		errs := check(ctx, tc, stepNumber, rangedIndex, assertion, r)
		if errs != nil {
			results = append(results, fmt.Sprintf("  - fail: %s", assertion))
		}
		if errs == nil {
			assertionsSuccess++
			results = append(results, fmt.Sprintf("  - pass: %s", assertion))
		}
	}

	// Evaluate operator behaviour
	var err error
	switch operator {
	case "and":
		if assertionsSuccess != assertionsCount {
			err = fmt.Errorf("%d/%d assertions succeeded:\n%s\n", assertionsSuccess, assertionsCount, strings.Join(results, "\n"))
		}
	case "or":
		if assertionsSuccess == 0 {
			err = fmt.Errorf("no assertions succeeded:\n%s\n", strings.Join(results, "\n"))
		}
	case "xor":
		if assertionsSuccess == 0 {
			err = fmt.Errorf("no assertions succeeded:\n%s\n", strings.Join(results, "\n"))
		}
		if assertionsSuccess > 1 {
			err = fmt.Errorf("multiple assertions succeeded but expected only one to succeed:\n%s\n", strings.Join(results, "\n"))
		}
	case "not":
		if assertionsSuccess > 0 {
			err = fmt.Errorf("some assertions succeeded but expected none to succeed:\n%s\n", strings.Join(results, "\n"))
		}
	default:
		return newFailure(ctx, tc, stepNumber, rangedIndex, "", fmt.Errorf("unsupported assertion operator %s", operator))
	}
	if err != nil {
		return newFailure(ctx, tc, stepNumber, rangedIndex, "", err)
	}
	return nil
}

// checkString evaluate a single string assertion
func checkString(ctx context.Context, tc TestCase, stepNumber int, rangedIndex int, assertion string, r interface{}) *Failure {
	assert, err := parseAssertions(context.Background(), assertion, r)
	if err != nil {
		return newFailure(ctx, tc, stepNumber, rangedIndex, assertion, err)
	}

	if err := assert.Func(assert.Actual, assert.Args...); err != nil {
		failure := newFailure(ctx, tc, stepNumber, rangedIndex, assertion, err)
		failure.AssertionRequired = assert.Required
		return failure
	}
	return nil
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
		return iVal, err
	case time.Time:
		return time.Parse(time.RFC3339, val)
	case time.Duration:
		return time.ParseDuration(val)
	}
	return val, nil
}

func findLineNumber(filename, testcase string, stepNumber int, assertion string, infoNumber int) int {
	countLine := 0
	file, err := os.Open(filename)
	if err != nil {
		return countLine
	}
	defer file.Close()

	lineFound := false
	testcaseFound := false
	commentBlock := false
	countStep := 0
	countInfo := 0

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
		if testcaseFound && countStep <= stepNumber && (strings.Contains(line, "type") || strings.Contains(line, "script")) {
			countStep++
			continue
		}

		if testcaseFound && countStep > stepNumber {
			if strings.Contains(line, assertion) {
				lineFound = true
				break
			} else if strings.Contains(strings.ReplaceAll(line, " ", ""), "info:") {
				countInfo++
				if infoNumber == countInfo {
					lineFound = true
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return countLine
	}

	if !lineFound {
		return 0
	}

	return countLine
}

// This evaluates a string of assertions with a given vars scope, and returns a slice of failures (i.e. empty slice = all pass)
func testConditionalStatement(ctx context.Context, tc *TestCase, assertions []string, vars H, text string) ([]string, error) {
	var failures []string
	for _, assertion := range assertions {
		Debug(ctx, "evaluating %s", assertion)
		assert, err := parseAssertions(ctx, assertion, vars)
		if err != nil {
			Error(ctx, "unable to parse assertion: %v", err)
			return failures, err
		}
		if err := assert.Func(assert.Actual, assert.Args...); err != nil {
			s := fmt.Sprintf(text, tc.originalName, err)
			failures = append(failures, s)
		}
	}
	return failures, nil
}
