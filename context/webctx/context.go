package webctx

import (
	"fmt"

	"github.com/sclevine/agouti"

	"github.com/runabove/venom"
)

// Context Type name
const Name = "web"
const ContextDriverKey = "driver"
const ContextPageKey = "page"

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	return &TestCaseContext{}
}

// TestCaseContex represents the context of a testcase
type TestCaseContext struct {}

// BuildContext build context of type web.
// It creates a new browser
func (TestCaseContext) BuildContext(tc *venom.TestCase) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	wd := agouti.PhantomJS()
	if err := wd.Start(); err != nil {
		return nil, fmt.Errorf("Cannot start web driver %s", err)
	}
	page, err := wd.NewPage()
	if err != nil {
		return nil, fmt.Errorf("Cannot create new page %s", err)
	}
	vars[ContextPageKey] = page
	vars[ContextDriverKey] = wd
	return vars, nil
}
