package webctx

import (
	"fmt"

	"github.com/sclevine/agouti"

	"github.com/runabove/venom"
)

// Context Type name
const Name = "web"

// Key of context element in testsuite file
const (
	Width      = "width"
	Height     = "height"
	Screenshot = "screenshotOnFailure"
)

// Key of element in the testcase context
const (
	ContextDriverKey           = "driver"
	ContextPageKey             = "page"
	ContextScreenshotOnFailure = "screenshotOnFailure"
)

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	return &TestCaseContext{}
}

// TestCaseContex represents the context of a testcase
type TestCaseContext struct{}

// BuildContext build context of type web.
// It creates a new browser
func (TestCaseContext) BuildContext(tc *venom.TestCase) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	// Get web driver
	wd := agouti.PhantomJS()
	if err := wd.Start(); err != nil {
		return nil, fmt.Errorf("Cannot start web driver %s", err)
	}
	vars[ContextDriverKey] = wd

	// Get Page
	page, err := wd.NewPage()
	if err != nil {
		return nil, fmt.Errorf("Cannot create new page %s", err)
	}

	resizePage := false
	if _, ok := tc.Context[Width]; ok {
		if _, ok := tc.Context[Height]; ok {
			resizePage = true
		}
	}

	// Get Page size
	if resizePage {
		var width, height int
		switch tc.Context[Width].(type) {
		case int:
			width = tc.Context[Width].(int)
		default:
			return nil, fmt.Errorf("%s is not an integer: %s", Width, fmt.Sprintf("%s", tc.Context[Width]))
		}
		switch tc.Context[Height].(type) {
		case int:
			height = tc.Context[Height].(int)
		default:
			return nil, fmt.Errorf("%s is not an integer: %s", Height, fmt.Sprintf("%s", tc.Context[Height]))
		}

		if err := page.Size(width, height); err != nil {
			return nil, fmt.Errorf("Cannot resize page: %s", err)
		}
	}
	vars[ContextPageKey] = page

	// Get screenshot param
	if _, ok := tc.Context[Screenshot]; ok {
		switch tc.Context[Screenshot].(type) {
		case bool:
			vars[ContextScreenshotOnFailure] = tc.Context[Screenshot].(bool)
		default:
			return nil, fmt.Errorf("%s must be a boolean, go %s", ContextScreenshotOnFailure, fmt.Sprintf("%s", tc.Context[ContextScreenshotOnFailure]))
		}
	}

	return vars, nil
}
