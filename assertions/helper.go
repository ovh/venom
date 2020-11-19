package assertions

import (
	"fmt"
	"reflect"
)

const (
	needExactValues        = "This assertion requires exactly %d comparison values (you provided %d)."
	needNonEmptyCollection = "This assertion requires at least 1 comparison value (you provided 0)."
	needSameType           = "This assertion requires 2 values of same types."
)

type AssertionError struct {
	cause error
}

func (e *AssertionError) Error() string {
	return e.cause.Error()
}

func newAssertionError(format string, a ...interface{}) *AssertionError {
	return &AssertionError{cause: fmt.Errorf(format, a...)}
}

func need(needed int, expected []interface{}) error {
	if len(expected) != needed {
		return newAssertionError(needExactValues, needed, len(expected))
	}
	return nil
}

func atLeast(minimum int, expected []interface{}) error {
	if len(expected) < minimum {
		return newAssertionError(needNonEmptyCollection)
	}
	return nil
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func areSameTypes(i, j interface{}) bool {
	return reflect.DeepEqual(
		reflect.Zero(reflect.TypeOf(i)).Interface(),
		reflect.Zero(reflect.TypeOf(j)).Interface(),
	)
}
