package assertions

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/tj/go-naturaldate"
)

type AssertFunc func(actual interface{}, expected ...interface{}) error

// assertMap contains list of assertions func
var assertMap = map[string]AssertFunc{
	"ShouldEqual":                  ShouldEqual,
	"ShouldNotEqual":               ShouldNotEqual,
	"ShouldAlmostEqual":            ShouldAlmostEqual,
	"ShouldNotAlmostEqual":         ShouldNotAlmostEqual,
	"ShouldNotExist":               ShouldNotExist,
	"ShouldBeNil":                  ShouldBeNil,
	"ShouldNotBeNil":               ShouldNotBeNil,
	"ShouldBeTrue":                 ShouldBeTrue,
	"ShouldBeFalse":                ShouldBeFalse,
	"ShouldBeZeroValue":            ShouldBeZeroValue,
	"ShouldBeGreaterThan":          ShouldBeGreaterThan,
	"ShouldBeGreaterThanOrEqualTo": ShouldBeGreaterThanOrEqualTo,
	"ShouldBeLessThan":             ShouldBeLessThan,
	"ShouldBeLessThanOrEqualTo":    ShouldBeLessThanOrEqualTo,
	"ShouldBeBetween":              ShouldBeBetween,
	"ShouldNotBeBetween":           ShouldNotBeBetween,
	"ShouldBeBetweenOrEqual":       ShouldBeBetweenOrEqual,
	"ShouldNotBeBetweenOrEqual":    ShouldNotBeBetweenOrEqual,
	"ShouldContain":                ShouldContain,
	"ShouldNotContain":             ShouldNotContain,
	"ShouldJSONContain":            ShouldJSONContain,
	"ShouldNotJSONContain":         ShouldNotJSONContain,
	"ShouldJSONContainWithKey":     ShouldJSONContainWithKey,
	"ShouldJSONContainAllWithKey":  ShouldJSONContainAllWithKey,
	"ShouldNotJSONContainWithKey":  ShouldNotJSONContainWithKey,
	"ShouldContainKey":             ShouldContainKey,
	"ShouldNotContainKey":          ShouldNotContainKey,
	"ShouldBeIn":                   ShouldBeIn,
	"ShouldNotBeIn":                ShouldNotBeIn,
	"ShouldBeEmpty":                ShouldBeEmpty,
	"ShouldNotBeEmpty":             ShouldNotBeEmpty,
	"ShouldHaveLength":             ShouldHaveLength,
	"ShouldStartWith":              ShouldStartWith,
	"ShouldNotStartWith":           ShouldNotStartWith,
	"ShouldEndWith":                ShouldEndWith,
	"ShouldNotEndWith":             ShouldNotEndWith,
	"ShouldBeBlank":                ShouldBeBlank,
	"ShouldNotBeBlank":             ShouldNotBeBlank,
	"ShouldContainSubstring":       ShouldContainSubstring,
	"ShouldNotContainSubstring":    ShouldNotContainSubstring,
	"ShouldEqualTrimSpace":         ShouldEqualTrimSpace,
	"ShouldHappenBefore":           ShouldHappenBefore,
	"ShouldHappenOnOrBefore":       ShouldHappenOnOrBefore,
	"ShouldHappenAfter":            ShouldHappenAfter,
	"ShouldHappenOnOrAfter":        ShouldHappenOnOrAfter,
	"ShouldHappenBetween":          ShouldHappenBetween,
	"ShouldTimeEqual":              ShouldTimeEqual,
	"ShouldJSONEqual":              ShouldJSONEqual,
	"ShouldNotJSONEqual":           ShouldNotJSONEqual,
	"ShouldBeArray":                ShouldBeArray,
	"ShouldBeMap":                  ShouldBeMap,
	"ShouldMatchRegex":             ShouldMatchRegex,
}

func Get(s string) (AssertFunc, bool) {
	f, ok := assertMap[s]
	return f, ok
}

func deepEqual(x, y interface{}) bool {
	if !reflect.DeepEqual(x, y) {
		return fmt.Sprintf("%v", x) == fmt.Sprintf("%v", y)
	}
	return true
}

func ShouldBeArray(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	_, err := cast.ToSliceE(actual)
	if err != nil {
		return fmt.Errorf("expected: %v to be an array but was not", actual)
	}
	return nil
}

func ShouldBeMap(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	_, err := cast.ToStringMapE(actual)
	if err != nil {
		return fmt.Errorf("expected: %v to be a map but was not", actual)
	}
	return nil
}

// ShouldEqual receives exactly two parameters and does an equality check.
//
// Example of testsuite file:
//
//	name: Assertions testsuite
//	testcases:
//	- name: test assertion
//	  steps:
//	  - script: echo 'foo'
//	    assertions:
//	    - result.code ShouldEqual 0
func ShouldEqual(actual interface{}, expected ...interface{}) error {
	// if expected is an array, we consider that this array is an array of string
	// so, we concat all values before doing the comparison
	if len(expected) > 0 {
		var args string
		for i := range expected {
			actualS, err := cast.ToStringE(expected[i])
			if err != nil {
				return err
			}
			args += actualS + " "
		}
		if deepEqual(actual, strings.TrimRight(args, " ")) {
			return nil
		}
		return fmt.Errorf("expected: %v got: %v", args, actual)
	}

	if err := need(1, expected); err != nil {
		return err
	}
	if deepEqual(actual, expected[0]) {
		return nil
	}
	return fmt.Errorf("expected: %v got: %v", expected[0], actual)
}

// ShouldMatchRegex receives exactly two parameters and does a regex match check.
func ShouldMatchRegex(actual interface{}, expected ...interface{}) error {
	if need(1, expected) != nil {
		return fmt.Errorf("expected one regex pattern")
	}
	expectedAsString, regexNotString := cast.ToStringE(expected[0])
	if regexNotString != nil {
		return fmt.Errorf("expected a string for regex pattern")
	}
	reg, err := regexp.Compile(expectedAsString)
	if err != nil {
		return err
	}
	actualAsString, castingErr := cast.ToStringE(actual)
	if castingErr != nil {
		return castingErr
	}
	if !reg.MatchString(actualAsString) {
		return fmt.Errorf("value %v not matching pattern : %v", actual, expected[0])
	}
	return nil
}

// ShouldNotEqual receives exactly two parameters and does an inequality check.
func ShouldNotEqual(actual interface{}, expected ...interface{}) error {
	if err := ShouldEqual(actual, expected...); err == nil {
		return fmt.Errorf("not expected: %v got: %v", expected[0], actual)
	}
	return nil
}

// ShouldAlmostEqual makes sure that two parameters are close enough to being equal.
// The acceptable delta may be specified with a third argument.
func ShouldAlmostEqual(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}
	actualF, err := cast.ToFloat64E(actual)
	if err != nil {
		return err
	}

	expectedF, err := cast.ToFloat64E(expected[0])
	if err != nil {
		return err
	}

	deltaF, err := cast.ToFloat64E(expected[1])
	if err != nil {
		return err
	}

	actualDeltaF := math.Abs(actualF - expectedF)

	if actualDeltaF <= deltaF {
		return nil
	}

	return fmt.Errorf("expected: %v(+/- %v) got: %v (%v)", expectedF, deltaF, actualF, actualDeltaF)
}

// ShouldNotAlmostEqual makes sure that two parameters are not close enough to being equal.
// The unacceptable delta may be specified with a third argument.
func ShouldNotAlmostEqual(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}
	actualF, err := cast.ToFloat64E(actual)
	if err != nil {
		return err
	}

	expectedF, err := cast.ToFloat64E(expected[0])
	if err != nil {
		return err
	}

	deltaF, err := cast.ToFloat64E(expected[1])
	if err != nil {
		return err
	}

	if math.Abs(actualF-expectedF) >= deltaF {
		return nil
	}

	return fmt.Errorf("not expected: %v(+/- %v) got: %v (%v)", expected[0], expected[1], actual, math.Abs(actualF-expectedF))
}

// ShouldBeNil receives a single parameter and ensures that it is nil.
func ShouldBeNil(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	if isNil(actual) {
		return nil
	}
	return fmt.Errorf("expected: Nil but is wasn't")
}

// ShouldNotExist receives a single parameter and ensures that it is nil
func ShouldNotExist(actual interface{}, expected ...interface{}) error {
	if ShouldBeNil(actual) != nil {
		return fmt.Errorf("expected not exist but it was")
	}
	return nil
}

// ShouldNotBeNil receives a single parameter and ensures that it is not nil.
func ShouldNotBeNil(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	if !isNil(actual) {
		return nil
	}
	return fmt.Errorf("expected: Not Nil but is was")
}

// ShouldBeTrue receives a single parameter and ensures that it is true.
func ShouldBeTrue(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	b, err := cast.ToBoolE(actual)
	if err != nil {
		return err
	}
	if b {
		return nil
	}
	return fmt.Errorf("expected: True but is wasn't")
}

// ShouldBeFalse receives a single parameter and ensures that it is false.
func ShouldBeFalse(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	b, err := cast.ToBoolE(actual)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	return fmt.Errorf("expected: False but is wasn't")
}

// ShouldBeZeroValue receives a single parameter and ensures that it is
// the Go equivalent of the default value, or "zero" value.
func ShouldBeZeroValue(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	b := actual == nil || reflect.DeepEqual(actual, reflect.Zero(reflect.TypeOf(actual)).Interface())
	if b {
		return nil
	}
	return fmt.Errorf("expected: Zero Value but is wasn't")
}

// ShouldBeGreaterThan receives exactly two parameters and ensures that the first is greater than the second.
func ShouldBeGreaterThan(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	if !areSameTypes(actual, expected[0]) {
		return newAssertionError(needSameType)
	}

	var actualF float64
	var err error
	switch x := actual.(type) {
	case json.Number:
		actualF, err = x.Float64()
		if err != nil {
			return err
		}
	default:
		actualF, err = cast.ToFloat64E(actual)
		if err != nil {
			actualS, err := cast.ToStringE(actual)
			if err != nil {
				return err
			}

			expectedS, err := cast.ToStringE(expected[0])
			if err != nil {
				return err
			}

			if actualS > expectedS {
				return nil
			}

			return fmt.Errorf("expected: %v greater than %v but it wasn't", actual, expected[0])
		}
	}

	expectedF, err := cast.ToFloat64E(expected[0])
	if err != nil {
		return err
	}

	if actualF > expectedF {
		return nil
	}

	return fmt.Errorf("expected: %v greater than %v but it wasn't", actual, expected[0])
}

// ShouldBeGreaterThanOrEqualTo receives exactly two parameters and ensures that the first is greater than or equal to the second.
func ShouldBeGreaterThanOrEqualTo(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	if !areSameTypes(actual, expected[0]) {
		return newAssertionError(needSameType)
	}

	var actualF float64
	var err error
	switch x := actual.(type) {
	case json.Number:
		actualF, err = x.Float64()
		if err != nil {
			return err
		}
	default:
		actualF, err = cast.ToFloat64E(actual)
		if err != nil {
			actualS, err := cast.ToStringE(actual)
			if err != nil {
				return err
			}

			expectedS, err := cast.ToStringE(expected[0])
			if err != nil {
				return err
			}

			if actualS >= expectedS {
				return nil
			}

			return fmt.Errorf("expected: %v greater than or equals to %v but it wasn't", actual, expected[0])

		}
	}

	expectedF, err := cast.ToFloat64E(expected[0])
	if err != nil {
		return err
	}

	if actualF >= expectedF {
		return nil
	}

	return fmt.Errorf("expected: %v greater than or equals to %v but it wasn't", actual, expected[0])
}

// ShouldBeLessThan receives exactly two parameters and ensures that the first is less than the second.
func ShouldBeLessThan(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	if !areSameTypes(actual, expected[0]) {
		return newAssertionError(needSameType)
	}

	var actualF float64
	var err error
	switch x := actual.(type) {
	case json.Number:
		actualF, err = x.Float64()
		if err != nil {
			return err
		}
	default:
		actualF, err = cast.ToFloat64E(actual)
		if err != nil {
			actualS, err := cast.ToStringE(actual)
			if err != nil {
				return err
			}

			expectedS, err := cast.ToStringE(expected[0])
			if err != nil {
				return err
			}

			if actualS < expectedS {
				return nil
			}

			return fmt.Errorf("expected: %v less than %v but it wasn't", actual, expected[0])

		}
	}

	expectedF, err := cast.ToFloat64E(expected[0])
	if err != nil {
		return err
	}

	if actualF < expectedF {
		return nil
	}

	return fmt.Errorf("expected: %v less than %v but it wasn't", actual, expected[0])
}

// ShouldBeLessThanOrEqualTo receives exactly two parameters and ensures that the first is less than or equal to the second.
func ShouldBeLessThanOrEqualTo(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	if !areSameTypes(actual, expected[0]) {
		return newAssertionError(needSameType)
	}

	var actualF float64
	var err error
	switch x := actual.(type) {
	case json.Number:
		actualF, err = x.Float64()
		if err != nil {
			return err
		}
	default:
		actualF, err = cast.ToFloat64E(actual)
		if err != nil {
			actualS, err := cast.ToStringE(actual)
			if err != nil {
				return err
			}

			expectedS, err := cast.ToStringE(expected[0])
			if err != nil {
				return err
			}

			if actualS <= expectedS {
				return nil
			}

			return fmt.Errorf("expected: %v less than or equals to %v but it wasn't", actual, expected[0])
		}
	}

	expectedF, err := cast.ToFloat64E(expected[0])
	if err != nil {
		return err
	}

	if actualF <= expectedF {
		return nil
	}

	return fmt.Errorf("expected '%v' less than or equals to %v but it wasn't", actual, expected[0])
}

// ShouldBeBetween receives exactly two parameters and ensures that the first is less than the second.
func ShouldBeBetween(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}

	if !areSameTypes(expected[0], expected[1]) {
		return newAssertionError(needSameType)
	}

	err1 := ShouldBeLessThan(actual, expected[1])
	err2 := ShouldBeGreaterThan(actual, expected[0])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("expected '%v' between %v and %v but it wasn't", actual, expected[0], expected[1])
	}
	return nil
}

// ShouldNotBeBetween receives exactly three parameters: an actual value, a lower bound, and an upper bound.
// It ensures that the actual value is NOT between both bounds.
func ShouldNotBeBetween(actual interface{}, expected ...interface{}) error {
	if err := ShouldBeBetween(actual, expected...); err != nil {
		if _, ok := err.(*AssertionError); ok {
			return err
		}
		return nil
	}
	return fmt.Errorf("expected '%v' not between %v and %v but it was", actual, expected[0], expected[1])
}

// ShouldBeBetweenOrEqual receives exactly three parameters: an actual value, a lower bound, and an upper bound.
// It ensures that the actual value is between both bounds or equal to one of them.
func ShouldBeBetweenOrEqual(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}

	if !areSameTypes(expected[0], expected[1]) {
		return newAssertionError(needSameType)
	}

	err1 := ShouldBeLessThanOrEqualTo(actual, expected[1])
	err2 := ShouldBeGreaterThanOrEqualTo(actual, expected[0])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("expected '%v' between %v and %v but it wasn't", actual, expected[0], expected[1])
	}
	return nil
}

// ShouldNotBeBetweenOrEqual receives exactly three parameters: an actual value, a lower bound, and an upper bound.
// It ensures that the actual value is nopt between the bounds nor equal to either of them.
func ShouldNotBeBetweenOrEqual(actual interface{}, expected ...interface{}) error {
	if err := ShouldBeBetweenOrEqual(actual, expected...); err != nil {
		if _, ok := err.(*AssertionError); ok {
			return err
		}
		return nil
	}
	return fmt.Errorf("expected '%v' not between or equal to %v and %v but it was", actual, expected[0], expected[1])
}

// ShouldContain receives exactly two parameters. The first is a slice or a single value and the
// second is a proposed member. Membership is determined using ShouldEqual.
func ShouldContain(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	if actual == nil || reflect.TypeOf(actual).Kind() != reflect.Slice {
		return ShouldEqual(actual, expected[0])
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	for i := range actualSlice {
		if ShouldEqual(actualSlice[i], expected[0]) == nil {
			return nil
		}
	}
	return fmt.Errorf("expected '%v' contain %v but it wasnt", actual, expected[0])
}

// ShouldNotContain receives exactly two parameters. The first is a slice and the
// second is a proposed member. Membership is determinied using ShouldEqual.
func ShouldNotContain(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	for i := range actualSlice {
		if ShouldEqual(actualSlice[i], expected[0]) == nil {
			return fmt.Errorf("expected '%v' not contain %v but it was", actual, expected[0])
		}
	}
	return nil
}

// ShouldJSONContain receives exactly two parameters. The first is a slice, the
// second is a proposed JSON member.
// Equality is determined using ShouldJSONEqual.
func ShouldJSONContain(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	for i := range actualSlice {
		if ShouldJSONEqual(actualSlice[i], expected[0]) == nil {
			return nil
		}
	}
	return fmt.Errorf("expected '%v' to contain %v but it wasnt", actual, expected[0])
}

// ShouldNotJSONContain receives exactly two parameters. The first is a slice, the
// second is a proposed JSON member.
// Equality is determined using ShouldJSONEqual.
func ShouldNotJSONContain(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	for i := range actualSlice {
		if ShouldJSONEqual(actualSlice[i], expected[0]) == nil {
			return fmt.Errorf("expected '%v' not contain %v but it was", actual, expected[0])
		}
	}
	return nil
}

// ShouldJSONContainWithKey receives exactly three parameters. The first is a slice, the
// second is a key in the inner slice structure and the third is a proposed value associated to the key.
// Equality is determined using ShouldJSONEqual.
func ShouldJSONContainWithKey(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	if reflect.TypeOf(expected[0]).Kind() != reflect.String {
		return fmt.Errorf("expected '%v' to be a string", expected[0])
	}
	expectedKey := cast.ToString(expected[0])
	for i := range actualSlice {
		if reflect.TypeOf(actualSlice[i]).Kind() != reflect.Map && reflect.TypeOf(actualSlice[i]).Kind() != reflect.Struct {
			return fmt.Errorf("expected '%v' to be a map or a struct", actualSlice[i])
		}
		elem := cast.ToStringMap(actualSlice[i])
		if err != nil {
			return err
		}
		if _, ok := elem[expectedKey]; ok {
			if ShouldJSONEqual(elem[expectedKey], expected[1]) == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("expected '%v' contain a key '%s' with value %v but it wasnt", actual, expectedKey, expected[1])
}

// ShouldJSONContainAllWithKey receives exactly three parameters. The first is a slice, the
// second is a key in the inner slice structure and the third is a proposed value associated to the key.
// Equality is determined using ShouldJSONEqual.
func ShouldJSONContainAllWithKey(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	if reflect.TypeOf(expected[0]).Kind() != reflect.String {
		return fmt.Errorf("expected '%v' to be a string", expected[0])
	}
	expectedKey := cast.ToString(expected[0])
	for i := range actualSlice {
		if reflect.TypeOf(actualSlice[i]).Kind() != reflect.Map && reflect.TypeOf(actualSlice[i]).Kind() != reflect.Struct {
			return fmt.Errorf("expected '%v' to be a map or a struct", actualSlice[i])
		}
		elem := cast.ToStringMap(actualSlice[i])
		if err != nil {
			return err
		}
		if _, ok := elem[expectedKey]; ok {
			if ShouldJSONEqual(elem[expectedKey], expected[1]) != nil {
				return fmt.Errorf("expected '%v' contain a key '%s' with value %v but it wasnt", actual, expectedKey, expected[1])
			}
		}
	}
	return nil
}

// ShouldNotJSONContainWithKey receives exactly three parameters. The first is a slice, the
// second is a key in the inner slice structure and the third is a proposed value associated to the key.
// Equality is determined using ShouldJSONEqual.
func ShouldNotJSONContainWithKey(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}
	actualSlice, err := cast.ToSliceE(actual)
	if err != nil {
		return err
	}
	if reflect.TypeOf(expected[0]).Kind() != reflect.String {
		return fmt.Errorf("expected '%v' to be a string", expected[0])
	}
	expectedKey := cast.ToString(expected[0])
	for i := range actualSlice {
		if reflect.TypeOf(actualSlice[i]).Kind() != reflect.Map && reflect.TypeOf(actualSlice[i]).Kind() != reflect.Struct {
			return fmt.Errorf("expected '%v' to be a map or a struct currently %v", actualSlice[i], reflect.TypeOf(actualSlice[i]).Kind())
		}
		elem := cast.ToStringMap(actualSlice[i])
		if err != nil {
			return err
		}
		if _, ok := elem[expectedKey]; ok {
			if ShouldJSONEqual(elem[expectedKey], expected[1]) == nil {
				return fmt.Errorf("expected '%v' not contain a key '%s' with value %v but it was", actual, expectedKey, expected[1])
			}
		}
	}
	return nil
}

// ShouldContainKey receives exactly two parameters. The first is a map and the
// second is a proposed key.
func ShouldContainKey(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualMap, err := cast.ToStringMapE(actual)
	if err != nil {
		return err
	}
	for k := range actualMap {
		if ShouldEqual(k, expected[0]) == nil {
			return nil
		}
	}
	return fmt.Errorf("expected '%v' contain key %v but it wasnt", actual, expected[0])
}

// ShouldNotContainKey receives exactly two parameters. The first is a map and the
// second is a proposed absent key.
func ShouldNotContainKey(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualMap, err := cast.ToStringMapE(actual)
	if err != nil {
		return err
	}
	for k := range actualMap {
		if ShouldEqual(k, expected[0]) == nil {
			return fmt.Errorf("expected '%v' not contain key %v but it was", actual, expected[0])
		}
	}
	return nil
}

// ShouldBeIn receives at least 2 parameters. The first is a proposed member of the collection
// that is passed in either as the second parameter, or of the collection that is comprised
// of all the remaining parameters. This assertion ensures that the proposed member is in
// the collection (using ShouldEqual).
//
// Example of testsuite file:
//
//	name: Assertions testsuite
//	testcases:
//	  - name: ShouldBeIn
//	    steps:
//	    - script: echo 1
//	      assertions:
//	      - result.systemoutjson ShouldBeIn 1 2
func ShouldBeIn(actual interface{}, expected ...interface{}) error {
	if err := atLeast(1, expected); err != nil {
		return err
	}

	expectedSlice, err := cast.ToSliceE(expected)
	if err != nil {
		return err
	}
	for i := range expectedSlice {
		if ShouldEqual(expectedSlice[i], actual) == nil {
			return nil
		}
	}
	return fmt.Errorf("expected '%v' in %v but it wasnt", actual, expectedSlice)
}

// ShouldNotBeIn receives at least 2 parameters. The first is a proposed member of the collection
// that is passed in either as the second parameter, or of the collection that is comprised
// of all the remaining parameters. This assertion ensures that the proposed member is NOT in
// the collection (using ShouldEqual).
//
// Example of testsuite file:
//
//	name: Assertions testsuite
//	testcases:
//	  - name: ShouldNotBeIn
//	    steps:
//	    - script: echo 3
//	      assertions:
//	      - result.systemoutjson ShouldNotBeIn 1 2
func ShouldNotBeIn(actual interface{}, expected ...interface{}) error {
	if err := atLeast(1, expected); err != nil {
		return err
	}

	if err := ShouldBeIn(actual, expected...); err != nil {
		return nil
	}

	return fmt.Errorf("expected '%v' not in %v but it was", actual, expected)
}

// ShouldBeEmpty receives a single parameter (actual) and determines whether or not
// calling len(actual) would return `0`.
func ShouldBeEmpty(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}

	if actual == nil {
		return nil
	}

	value := reflect.ValueOf(actual)
	switch value.Kind() {
	case reflect.Slice, reflect.Chan, reflect.Map, reflect.String:
		if value.Len() == 0 {
			return nil
		}
	case reflect.Ptr:
		elem := value.Elem()
		kind := elem.Kind()
		if (kind == reflect.Slice || kind == reflect.Array) && elem.Len() == 0 {
			return nil
		}
	}

	return fmt.Errorf("expected '%v' to be empty but it wasn't", actual)
}

// ShouldNotBeEmpty receives a single parameter (actual) and determines whether or not
// calling len(actual) would return a value greater than zero.
func ShouldNotBeEmpty(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}
	if err := ShouldBeEmpty(actual); err != nil {
		return nil
	}
	return fmt.Errorf("expected '%v' not to be empty but it wasn't", actual)
}

// ShouldHaveLength receives 2 parameters. The first is a collection to check
// the length of, the second being the expected length.
func ShouldHaveLength(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	length, err := cast.ToInt64E(expected[0])
	if err != nil {
		return err
	}

	var actualLength int

	value := reflect.ValueOf(actual)
	switch value.Kind() {
	case reflect.Slice, reflect.Chan, reflect.Map, reflect.String:
		actualLength = value.Len()
		if value.Len() == int(length) {
			return nil
		}
	case reflect.Ptr:
		elem := value.Elem()
		kind := elem.Kind()
		actualLength = elem.Len()
		if (kind == reflect.Slice || kind == reflect.Array) && actualLength == int(length) {
			return nil
		}
	}

	return fmt.Errorf("expected '%v' have length of %d but it wasn't (%d)", actual, length, actualLength)
}

// ShouldStartWith receives exactly 2 string parameters and ensures that the first starts with the second.
func ShouldStartWith(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	prefix, err := cast.ToStringE(expected[0])
	if err != nil {
		return err
	}

	if strings.HasPrefix(s, prefix) {
		return nil
	}

	return fmt.Errorf("expected '%v' have prefix %q but it wasn't", s, prefix)
}

// ShouldNotStartWith receives exactly 2 string parameters and ensures that the first does not start with the second.
func ShouldNotStartWith(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	prefix, err := cast.ToStringE(expected[0])
	if err != nil {
		return err
	}

	if strings.HasPrefix(s, prefix) {
		return fmt.Errorf("expected '%v' not have prefix %q but it was", s, prefix)
	}

	return nil
}

// ShouldEndWith receives exactly 2 string parameters and ensures that the first ends with the second.
func ShouldEndWith(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	suffix, err := cast.ToStringE(expected[0])
	if err != nil {
		return err
	}

	if strings.HasSuffix(s, suffix) {
		return nil
	}

	return fmt.Errorf("expected '%v' have suffix %q but it wasn't", s, suffix)
}

// ShouldNotEndWith receives exactly 2 string parameters and ensures that the first does not end with the second.
func ShouldNotEndWith(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	suffix, err := cast.ToStringE(expected[0])
	if err != nil {
		return err
	}

	if strings.HasSuffix(s, suffix) {
		return fmt.Errorf("expected '%v' not have suffix %q but it was", s, suffix)
	}

	return nil
}

// ShouldBeBlank receives exactly 1 string parameter and ensures that it is equal to "".
func ShouldBeBlank(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	if s == "" {
		return nil
	}

	return fmt.Errorf("expected '%v' to be blank but it wasn't", s)
}

// ShouldNotBeBlank receives exactly 1 string parameter and ensures that it is equal to "".
func ShouldNotBeBlank(actual interface{}, expected ...interface{}) error {
	if err := need(0, expected); err != nil {
		return err
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	if s == "" {
		return fmt.Errorf("expected value to not be blank but it was")
	}

	return nil
}

// ShouldContainSubstring receives exactly 2 string parameters and ensures that the first contains the second as a substring.
func ShouldContainSubstring(actual interface{}, expected ...interface{}) error {
	var arg string
	for _, e := range expected {
		arg += fmt.Sprintf("%v ", e)
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	ss := strings.TrimSpace(arg)

	if strings.Contains(s, ss) {
		return nil
	}

	return fmt.Errorf("expected '%v' to contain '%v' but it wasn't", s, ss)
}

// ShouldNotContainSubstring receives exactly 2 string parameters and ensures that the first does NOT contain the second as a substring.
func ShouldNotContainSubstring(actual interface{}, expected ...interface{}) error {
	var arg string
	for _, e := range expected {
		arg += fmt.Sprintf("%v ", e)
	}

	s, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}

	ss := strings.TrimSpace(arg)

	if strings.Contains(s, ss) {
		return fmt.Errorf("expected '%v' to not contain '%v' but it was", s, ss)
	}

	return nil
}

// ShouldEqualTrimSpace receives exactly 2 string parameters and ensures that the first is equal to the second
// after removing all leading and trailing whitespace using strings.TrimSpace(first).
func ShouldEqualTrimSpace(actual interface{}, expected ...interface{}) error {
	actualS, err := cast.ToStringE(actual)
	if err != nil {
		return err
	}
	return ShouldEqual(strings.TrimSpace(actualS), expected...)
}

// ShouldHappenBefore receives exactly 2 time.Time arguments and asserts that the first happens before the second.
// The arguments have to respect the date format RFC3339 (ie. 2006-01-02T15:04:00+07:00) or humanize format (ie. now, tomorrow, yesterday, 5 minutes ago)
//
// Example of testsuite file:
//
//	name: test ShouldHappenBefore
//	vars:
//	  time: 2006-01-02T15:04:05+07:00
//	  time_with_5s_after: 2006-01-02T15:04:10+07:00
//	testcases:
//	- name: test assertion
//	  steps:
//	  - type: exec
//	    script: "echo {{.time}}"
//	    assertions:
//	      - result.systemout ShouldHappenBefore "{{.time_with_5s_after}}"
//	 - name: test assertion with humanize format
//	   steps:
//	   - type: exec
//	     script: "echo {{.venom.datetime}}"
//	     assertions:
//	       - "result.systemout ShouldHappenBefore tomorrow"
//	       - "result.systemout ShouldHappenBefore '5 minutes from now'"
func ShouldHappenBefore(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	actualTime, err := getTimeFromString(actual)
	if err != nil {
		return err
	}
	expectedTime, err := getTimeFromString(expected[0])
	if err != nil {
		return err
	}

	if actualTime.Before(expectedTime) {
		return nil
	}

	return fmt.Errorf("expected '%v' to be before '%v'", actualTime, expectedTime)
}

// ShouldHappenOnOrBefore receives exactly 2 time.Time arguments and asserts that the first happens on or before the second.
// The arguments have to respect the date format RFC3339 (ie. 2006-01-02T15:04:00+07:00) or humanize format (ie. now, tomorrow, yesterday, 5 minutes ago)
//
// Example of testsuite file:
//
//	name: test ShouldHappenOnOrBefore
//	vars:
//	  time: 2006-01-02T15:04:05+07:00
//	  time_with_5s_after: 2006-01-02T15:04:10+07:00
//	testcases:
//	- name: test assertion
//	  steps:
//	  - type: exec
//	    script: "echo {{.time}}"
//	    assertions:
//	      - result.systemout ShouldHappenOnOrBefore "{{.time_with_5s_after}}"
//	 - name: test assertion with humanize format
//	   steps:
//	   - type: exec
//	     script: "echo {{.venom.datetime}}"
//	     assertions:
//	       - "result.systemout ShouldHappenOnOrBefore tomorrow"
//	       - "result.systemout ShouldHappenOnOrBefore '5 minutes from now'"
func ShouldHappenOnOrBefore(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualTime, err := getTimeFromString(actual)
	if err != nil {
		return err
	}
	expectedTime, err := getTimeFromString(expected[0])
	if err != nil {
		return err
	}

	if actualTime.Before(expectedTime) || actualTime.Equal(expectedTime) {
		return nil
	}

	return fmt.Errorf("expected '%v' to be before on on '%v'", actualTime, expectedTime)
}

// ShouldHappenAfter receives exactly 2 time.Time arguments and asserts that the first happens after the second.
// The arguments have to respect the date format RFC3339 (ie. 2006-01-02T15:04:00+07:00) or humanize format (ie. now, tomorrow, yesterday, 5 minutes ago)
//
// Example of testsuite file:
//
//	name: test ShouldHappenAfter
//	vars:
//	  time_with_5s_before: 2006-01-02T15:04:00+07:00
//	  time: 2006-01-02T15:04:05+07:00
//	testcases:
//	- name: test assertion
//	  steps:
//	  - type: exec
//	    script: "echo {{.time}}"
//	    assertions:
//	      - result.systemout ShouldHappenAfter "{{.time_with_5s_before}}"
//	 - name: test assertion with humanize format
//	   steps:
//	   - type: exec
//	     script: "echo {{.venom.datetime}}"
//	     assertions:
//	       - "result.systemout ShouldHappenAfter yesterday"
//	       - "result.systemout ShouldHappenAfter '5 minutes ago'"
func ShouldHappenAfter(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}
	actualTime, err := getTimeFromString(actual)
	if err != nil {
		return err
	}
	expectedTime, err := getTimeFromString(expected[0])
	if err != nil {
		return err
	}

	if actualTime.After(expectedTime) {
		return nil
	}

	return fmt.Errorf("expected '%v' to be after '%v'", actualTime, expectedTime)
}

// ShouldHappenOnOrAfter receives exactly 2 time.Time arguments and asserts that the first happens on or after the second.
// The arguments have to respect the date format RFC3339 (ie. 2006-01-02T15:04:00+07:00) or humanize format (ie. now, tomorrow, yesterday, 5 minutes ago)
//
// Example of testsuite file:
//
//	name: test ShouldHappenOnOrAfter
//	vars:
//	  time_with_5s_before: 2006-01-02T15:04:00+07:00
//	  time: 2006-01-02T15:04:05+07:00
//	testcases:
//	- name: test assertion
//	  steps:
//	  - type: exec
//	    script: "echo {{.time}}"
//	    assertions:
//	      - result.systemout ShouldHappenOnOrAfter "{{.time_with_5s_before}}"
//	 - name: test assertion with humanize format
//	   steps:
//	   - type: exec
//	     script: "echo {{.venom.datetime}}"
//	     assertions:
//	       - "result.systemout ShouldHappenOnOrAfter yesterday"
//	       - "result.systemout ShouldHappenOnOrAfter '5 minutes ago'"
func ShouldHappenOnOrAfter(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	actualTime, err := getTimeFromString(actual)
	if err != nil {
		return err
	}
	expectedTime, err := getTimeFromString(expected[0])
	if err != nil {
		return err
	}

	if actualTime.After(expectedTime) || actualTime.Equal(expectedTime) {
		return nil
	}
	return fmt.Errorf("expected '%v' to be before or on '%v'", actualTime, expectedTime)
}

// ShouldHappenBetween receives exactly 3 time.Time arguments and asserts that the first happens between (not on) the second and third.
// The arguments have to respect the date format RFC3339 (ie. 2006-01-02T15:04:00+07:00) or humanize format (ie. yesterday, 5 minutes ago)
//
// Example of testsuite file:
//
//	name: test ShouldHappenBetween
//	vars:
//	  time_with_5s_before: 2006-01-02T15:04:00+07:00
//	  time: 2006-01-02T15:04:05+07:00
//	  time_with_5s_after: 2006-01-02T15:04:10+07:00
//	testcases:
//	- name: test assertion
//	  steps:
//	  - type: exec
//	    script: "echo {{.time}}"
//	    assertions:
//	      - result.systemout ShouldHappenBetween "{{.time_with_5s_before}}" "{{.time_with_5s_after}}"
//	- name: test assertion with humanize format
//	   steps:
//	   - type: exec
//	     script: "echo {{.venom.datetime}}"
//	     assertions:
//	       - "result.systemout ShouldHappenBetween yesterday tomorrow"
//	       - "result.systemout ShouldHappenBetween '5 minutes ago' '5 minutes from now'"
func ShouldHappenBetween(actual interface{}, expected ...interface{}) error {
	if err := need(2, expected); err != nil {
		return err
	}

	actualTime, err := getTimeFromString(actual)
	if err != nil {
		return err
	}
	min, err := getTimeFromString(expected[0])
	if err != nil {
		return err
	}
	max, err := getTimeFromString(expected[1])
	if err != nil {
		return err
	}

	if actualTime.After(min) && actualTime.Before(max) {
		return nil
	}
	return fmt.Errorf("expected '%v' to be between '%v' and '%v' ", actualTime, min, max)
}

// ShouldTimeEqual receives exactly 2 time.Time arguments and does a time equality check.
// The arguments have to respect the date format RFC3339, as 2006-01-02T15:04:00+07:00
//
// Example of testsuite file:
//
//	name: test ShouldTimeEqual
//	vars:
//	  time_expected: 2006-01-02T13:04:00Z
//	  time: 2006-01-02T15:04:00+02:00
//	testcases:
//	- name: test assertion
//	  steps:
//	  - type: exec
//	    script: "echo {{.time}}"
//	    assertions:
//	      - result.systemout ShouldTimeEqual "{{.time_expected}}"
func ShouldTimeEqual(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	actualTime, err := getTimeFromString(actual)
	if err != nil {
		return err
	}
	expectedTime, err := getTimeFromString(expected[0])
	if err != nil {
		return err
	}
	if actualTime.Equal(expectedTime) {
		return nil
	}
	return fmt.Errorf("expected '%v' to be time equals to '%v' ", actualTime, expectedTime)
}

// ShouldJSONEqual receives exactly 2 JSON arguments and does a JSON equality check.
// The latest JSON spec doesn't only allow objects and arrays, but primitive values are valid JSON as well.
// For object equality keys can be in different order, and whitespace (except in keys or values) is ignored.
// For arrays the order is important, but whitespace (except in values) is ignored.
// String, number, true/false are compared as-is.
// `null` JSON values are currently passed as empty string, and are compared against the "null" string.
//
// For an example scenario see `tests/assertions/ShouldJSONEqual.yml`.
func ShouldJSONEqual(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	switch actual.(type) {
	case map[string]interface{}:
		actualMap, err := cast.ToStringMapE(actual)
		if err != nil {
			return err
		}
		expectedString, err := cast.ToStringE(expected[0])
		if err != nil {
			return err
		}

		// Marshal and unmarshal for later deepequal to work
		actualBytes, err := json.Marshal(actualMap)
		if err != nil {
			return err
		}
		err = json.Unmarshal(actualBytes, &actualMap)
		if err != nil {
			return err
		}

		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedString), &expectedMap)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(actualMap, expectedMap) {
			return nil
		}
		return fmt.Errorf("expected '%v' to be JSON equals to '%v' ", actualMap, expectedMap)
	case []interface{}:
		actualSlice, err := cast.ToSliceE(actual)
		if err != nil {
			return err
		}
		expectedString, err := cast.ToStringE(expected[0])
		if err != nil {
			return err
		}

		// Marshal and unmarshal for later deepequal to work
		actualBytes, err := json.Marshal(actualSlice)
		if err != nil {
			return err
		}
		err = json.Unmarshal(actualBytes, &actualSlice)
		if err != nil {
			return err
		}

		expectedSlice := []interface{}{}
		err = json.Unmarshal([]byte(expectedString), &expectedSlice)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(actualSlice, expectedSlice) {
			return nil
		}
		return fmt.Errorf("expected '%v' to be JSON equals to '%v' ", actualSlice, expectedSlice)
	case string:
		actualString, err := cast.ToStringE(actual)
		if err != nil {
			return err
		}
		expectedString, err := cast.ToStringE(expected[0])
		if err != nil {
			return err
		}

		if actualString == expectedString {
			return nil
		}
		// Special case: Venom passes an empty string when `actual` JSON is JSON's `null`.
		// Above check is already valid when `expected` is an empty string, but
		// the user might have passed `null` explicitly.
		// TODO: This should be changed as soon as Venom passes Go's `nil` for JSON `null` values.
		if actualString == "" && expectedString == "null" {
			return nil
		}
		return fmt.Errorf("expected '%v' to be JSON equals to '%v' ", actualString, expectedString)
	case json.Number:
		actualFloat, err := cast.ToFloat64E(actual)
		if err != nil {
			return err
		}
		expectedFloat, err := cast.ToFloat64E(expected[0])
		if err != nil {
			return err
		}

		if actualFloat == expectedFloat {
			return nil
		}
		return fmt.Errorf("expected '%v' to be JSON equals to '%v' ", actualFloat, expectedFloat)
	case bool:
		actualBool, err := cast.ToBoolE(actual)
		if err != nil {
			return err
		}
		expectedBool, err := cast.ToBoolE(expected[0])
		if err != nil {
			return err
		}

		if actualBool == expectedBool {
			return nil
		}
		return fmt.Errorf("expected '%v' to be JSON equals to '%v' ", actualBool, expectedBool)
	default:
		return fmt.Errorf("unexpected type for actual: %T", actual)
	}
}

// ShouldNotJSONEqual asserts that the actual value is not JSON equal to the expected value
func ShouldNotJSONEqual(actual interface{}, expected ...interface{}) error {
	if err := need(1, expected); err != nil {
		return err
	}

	if err := ShouldJSONEqual(actual, expected...); err == nil {
		return fmt.Errorf("expected %v to not be JSON equals to %v", actual, expected[0])
	}
	return nil
}

func getTimeFromString(in interface{}) (time.Time, error) {
	if t, isTime := in.(time.Time); isTime {
		return t, nil
	}

	s, err := cast.ToStringE(in)
	if err != nil {
		return time.Time{}, errors.Errorf("invalid date provided: %q", in)
	}

	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	t, innerErr := naturaldate.Parse(s, time.Now())
	if innerErr != nil {
		return time.Time{}, errors.Errorf("invalid date provided with %q not in RFC3339 format or humanize format", in)
	}

	return t, nil
}
