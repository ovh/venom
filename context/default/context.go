package defaultctx

import "github.com/runabove/venom"

// Context Type name
const Name = "default"

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	ctx := &TestCaseContext{}
	ctx.Name = Name
	return ctx
}

// TestCaseContext represents the context of a testcase
type TestCaseContext struct {
	venom.TestCaseContextStruct
	datas map[string]interface{}
}

// Init Initialize the context
func (tcc *TestCaseContext) Init() error {
	return nil
}

// Close the context
func (tcc *TestCaseContext) Close() error {
	return nil
}
