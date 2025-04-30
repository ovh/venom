package playwright

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	playwrightgo "github.com/playwright-community/playwright-go"
)

const Name = "playwright"

type Executor struct {
	URL      string   `json:"url" yaml:"url"`
	Browser  string   `json:"browser" yaml:"browser"`
	Actions  []string `json:"actions" yaml:"actions"`
	Headless bool     `json:"headless" yaml:"headless"`
}

func New() venom.Executor {
	return &Executor{
		Headless: true,
	}
}

type Result struct {
	Page     *Page `json:"page" yaml:"page"`
	Document *Page `json:"document" yaml:"document"` // alias to Page
}

type Page struct {
	Location *url.URL   `json:"location" yaml:"location"`
	Body     string     `json:"body" yaml:"body"`
	Query    *PageQuery `json:"query" yaml:"query"`
	Scripts  []string   `json:"scripts" yaml:"scripts"`
	CSSFiles []string   `json:"css_files" yaml:"css_files"`
}

// PageQuery allows users to assert the page.Body using css selectrors
type PageQuery struct {
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []venom.Assertion{"page.body ShouldNotBeEmpty"}}
}

// Run execute TestStep of type playwright
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	pageURL, err := url.Parse(e.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL passed to playright executor: %s", e.URL)
	}

	browsers := make([]string, 0)
	if e.Browser != "" && slices.Contains[[]string, string]([]string{"chromium", "firefox"}, e.Browser) {
		browsers = append(browsers, e.Browser)
	} else {
		browsers = append(browsers, "chromium")
	}
	err = playwrightgo.Install(&playwrightgo.RunOptions{
		Browsers: browsers,
	})
	if err != nil {
		return nil, fmt.Errorf("could not launch playwright: %w", err)
	}

	pw, err := playwrightgo.Run()
	if err != nil {
		return nil, fmt.Errorf("could not launch playwright: %w", err)
	}
	browser, err := pw.Chromium.Launch(playwrightgo.BrowserTypeLaunchOptions{
		Headless: playwrightgo.Bool(e.Headless), // should we expose this option?
	})
	if err != nil {
		return nil, fmt.Errorf("could not launch Chromium: %w", err)
	}
	context, err := browser.NewContext()
	if err != nil {
		return nil, fmt.Errorf("could not create context: %w", err)
	}
	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	_, err = page.Goto(e.URL)
	if err != nil {
		return nil, fmt.Errorf("could not goto: %w", err)
	}

	err = performActions(ctx, page, e.Actions)
	if err != nil {
		return nil, err
	}

	pageBodyBytes, err := page.Content()
	if err != nil {
		return nil, fmt.Errorf("could not goto: %w", err)
	}

	err = browser.Close()
	if err != nil {
		return nil, fmt.Errorf("could not close browser: %w", err)
	}
	err = pw.Stop()
	if err != nil {
		return nil, fmt.Errorf("could not stop Playwright: %w", err)
	}

	pageResult := &Page{
		Location: pageURL,
		Body:     string(pageBodyBytes),
		Query:    nil,
	}

	return Result{
		Page:     pageResult,
		Document: pageResult,
	}, nil
}

// parseActionLine parses a line containing an Action expression and returns
// the actionName, the arguments (rest of the line), the actionFunc or an error
func parseActionLine(actionLine string) (string, string, ActionFunc, error) {
	if actionLine == "" {
		return "", "", nil, fmt.Errorf("action line MUST not be empty")
	}
	parts := strings.SplitN(strings.TrimSpace(actionLine), " ", 2)
	if len(parts) < 2 {
		return "", "", nil, fmt.Errorf("action line MUST have atleast two arguments: ACTION <selector>")
	}
	actionName, arguments := parts[0], parts[1]
	actionFunc, ok := actionMap[actionName]
	if !ok {
		return "", "", nil, fmt.Errorf("invalid or unsupported action specified '%s'", actionName)
	}
	return actionName, arguments, actionFunc, nil
}

func performActions(ctx context.Context, page playwrightgo.Page, actions []string) error {
	for _, action := range actions {
		actionName, arguments, actionFunc, err := parseActionLine(action)
		if err != nil {
			return err
		}

		venom.Debug(ctx, fmt.Sprintf("perform action '%s' with arguments '%v'\n", actionName, arguments))

		selectorAndArgs := strings.SplitN(strings.TrimSpace(arguments), " ", 2)
		selector := removeQuotes(selectorAndArgs[0])

		var actErr error
		if len(selectorAndArgs) <= 1 {
			actErr = actionFunc(page, selector, nil)
		} else {
			actErr = actionFunc(page, selector, removeQuotes(selectorAndArgs[1]))
		}
		if actErr != nil {
			return actErr
		}
	}
	return nil
}
