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

	err = performActions(page, e.Actions)
	if err != nil {
		return nil, err
	}

	pageBodyBytes, err := page.Content()
	if err != nil {
		return nil, fmt.Errorf("could not goto: %w", err)
	}
	// TODO: run the assertions in here ...

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

func performActions(page playwrightgo.Page, actions []string) error {
	for _, action := range actions {
		fmt.Println("perform action step", action)
		parts := strings.SplitN(strings.TrimSpace(action), " ", 2)
		actionName, arguments := parts[0], parts[1]
		actionFunc, ok := actionMap[actionName]
		if !ok {
			return fmt.Errorf("invalid or unsupported action specified '%s'", actionName)
		}
		selectorAndArgs := strings.SplitN(strings.TrimSpace(arguments), " ", 2)
		selector := removeQuotes(selectorAndArgs[0])
		var err error
		if len(selectorAndArgs) <= 1 {
			err = actionFunc(page, selector, nil)
		} else {
			err = actionFunc(page, selector, removeQuotes(selectorAndArgs[1]))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
