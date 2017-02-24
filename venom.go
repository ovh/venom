package venom

import (
	"context"
	"fmt"
	"os"
)

// Version of Venom
// One Line for this, used by release.sh script
// Keep "const Version on one line"
const Version = "0.0.1"

var (
	executors = map[string]Executor{}
	contexts  = map[string]TestCaseContext{}
)

const (
	// ContextKey is key for Test Case Context. this
	// can be used by executors for getting context
	ContextKey = "tcContext"
)

// RegisterExecutor register Test Executors
func RegisterExecutor(name string, e Executor) {
	executors[name] = e
}

// getExecutorWrap initializes a test by name
// no type -> exec is default
func getExecutorWrap(t map[string]interface{}) (*executorWrap, error) {

	var name string
	var retry, delay, timeout int

	if itype, ok := t["type"]; ok {
		name = fmt.Sprintf("%s", itype)
	}

	if name == "" {
		name = "exec"
	}

	retry, errRetry := getAttrInt(t, "retry")
	if errRetry != nil {
		return nil, errRetry
	}
	delay, errDelay := getAttrInt(t, "delay")
	if errDelay != nil {
		return nil, errDelay
	}
	timeout, errTimeout := getAttrInt(t, "timeout")
	if errTimeout != nil {
		return nil, errTimeout
	}

	if e, ok := executors[name]; ok {
		ew := &executorWrap{
			executor: e,
			retry:    retry,
			delay:    delay,
			timeout:  timeout,
		}
		return ew, nil
	}

	return nil, fmt.Errorf("type '%s' is not implemented", name)
}

// RegisterTestCaseContext new register TestCaseContext
func RegisterTestCaseContext(name string, tcc TestCaseContext) {
	contexts[name] = tcc
}

// getContextWrap initializes a context for a testcase
// no type -> parent context
func getContextWrap(tc *TestCase) (context.Context, error) {
	if tc.Context == nil {
		return context.Background(), nil
	}

	var typeName string
	if itype, ok := tc.Context["type"]; ok {
		typeName = fmt.Sprintf("%s", itype)
	}

	if typeName == "" {
		return nil, fmt.Errorf("context type '%s' is not implemented", typeName)
	}

	tcVars, errC := contexts[typeName].BuildContext(tc)
	if errC != nil {
		return nil, fmt.Errorf("Cannot build context type '%s': %s", typeName, errC)
	}
	return context.WithValue(context.Background(), ContextKey, tcVars), nil
}

func getAttrInt(t map[string]interface{}, name string) (int, error) {
	var out int
	if i, ok := t[name]; ok {
		var ok bool
		out, ok = i.(int)
		if !ok {
			return -1, fmt.Errorf("attribute %s '%s' is not an integer", name, i)
		}
	}
	if out < 0 {
		out = 0
	}
	return out, nil
}

// Exit func display an error message on stderr and exit 1
func Exit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
