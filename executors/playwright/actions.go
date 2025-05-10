package playwright

import (
	"fmt"
	"net/url"
	"strings"

	playwrightgo "github.com/playwright-community/playwright-go"
	"github.com/spf13/cast"
)

type ActionFunc func(page playwrightgo.Page, action *ExecutorAction) error

var actionMap = map[string]ActionFunc{
	"Click":                 ClickAction,
	"DoubleClick":           DoubleClickAction,
	"Doubleclick":           DoubleClickAction,
	"Tap":                   TapAction,
	"Focus":                 FocusAction,
	"Blur":                  BlurAction,
	"Clear":                 ClearAction,
	"Fill":                  FillAction,
	"Check":                 CheckAction,
	"Uncheck":               UncheckAction,
	"FillCheckbox":          CheckAction, // alias for Check
	"Press":                 PressAction,
	"PressSequentially":     PressSequentiallyAction,
	"Select":                SelectOptionAction, // alias for SelectOption
	"SelectOption":          SelectOptionAction,
	"SelectMultipleOptions": SelectMultipleOptionsAction,
	"Type":                  PressSequentiallyAction, // alias for PressSequentially
	"WaitFor":               WaitForSelectorAction,
	"WaitForSelector":       WaitForSelectorAction,
	"WaitForURL":            WaitForURLAction,
	"Goto":                  GotoAction,
	"GoBack":                GoBackAction,
	"GoForward":             GoForwardAction,
	"Refresh":               RefreshAction,
}

func removeQuotes(selector string) string {
	return strings.TrimSuffix(strings.TrimPrefix(selector, `"`), `"`)
}

func ClickAction(page playwrightgo.Page, action *ExecutorAction) error {
	return page.Locator(action.Selector).Click()
}

func WaitForSelectorAction(page playwrightgo.Page, action *ExecutorAction) error {
	timeout := 10_000.00
	return page.Locator(action.Selector).WaitFor(playwrightgo.LocatorWaitForOptions{
		Timeout: &timeout,
		State:   playwrightgo.WaitForSelectorStateAttached,
	})
}

func WaitForURLAction(page playwrightgo.Page, action *ExecutorAction) error {
	timeout := 10_000.00
	urlPattern := action.Selector
	return page.WaitForURL(urlPattern, playwrightgo.PageWaitForURLOptions{
		Timeout:   &timeout,
		WaitUntil: playwrightgo.WaitUntilStateCommit,
	})
}

func FillAction(page playwrightgo.Page, action *ExecutorAction) error {
	element := action.Selector
	target := action.Content
	if target == "" {
		return fmt.Errorf("need data to fill on '%s'", element)
	}
	return page.Locator(element).First().Fill(cast.ToString(target))
}

func PressAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Content == "" {
		return fmt.Errorf("need key to press on '%s'", action.Selector)
	}
	return page.Locator(action.Selector).First().Press(cast.ToString(action.Content))
}

func PressSequentiallyAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Content == "" {
		return fmt.Errorf("need key to press on '%s'", action.Selector)
	}
	return page.Locator(action.Selector).First().PressSequentially(cast.ToString(action.Content))
}

func DoubleClickAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to double click on")
	}
	// TODO: support passing double click options
	return page.Locator(action.Selector).First().Dblclick()
}

func TapAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to tap on")
	}
	// TODO: support passing Tap options
	return page.Locator(action.Selector).First().Tap()
}

func FocusAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to focus on")
	}
	// TODO: support passing Focus options
	return page.Locator(action.Selector).First().Focus()
}

func BlurAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to blur")
	}
	// TODO: support passing Blur options
	return page.Locator(action.Selector).First().Blur()
}

func ClearAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to blur")
	}
	// TODO: support passing Clear options
	return page.Locator(action.Selector).First().Clear()
}

func CheckAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to check on")
	}
	// TODO: support passing Check options
	return page.Locator(action.Selector).First().Check()
}

func UncheckAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to uncheck on")
	}
	// TODO: support passing Uncheck options
	return page.Locator(action.Selector).First().Uncheck()
}

func RefreshAction(page playwrightgo.Page, action *ExecutorAction) error {
	_, err := page.Reload()
	return err
}

func GoBackAction(page playwrightgo.Page, action *ExecutorAction) error {
	_, err := page.GoBack()
	return err
}

func GoForwardAction(page playwrightgo.Page, action *ExecutorAction) error {
	_, err := page.GoForward()
	return err
}

func GotoAction(page playwrightgo.Page, action *ExecutorAction) error {
	urlPattern := action.Selector
	timeout := 10_000.00
	finalURL := urlPattern
	if strings.HasPrefix(urlPattern, "/") { // relative url
		parsedURL, err := url.Parse(page.URL())
		if err != nil {
			return err
		}
		u, err := parsedURL.Parse(urlPattern)
		if err != nil {
			return err
		}
		finalURL = u.String()
	}
	_, err := page.Goto(finalURL, playwrightgo.PageGotoOptions{
		Timeout:   &timeout,
		WaitUntil: playwrightgo.WaitUntilStateCommit,
	})
	return err
}

func SelectOptionAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need a <select> element to select option on")
	}
	if action.Content == "" || len(action.Content) < 1 {
		return fmt.Errorf("need a <select> element to select option on")
	}
	valuesOrLabels := make([]string, 0)
	valuesOrLabels = append(valuesOrLabels, cast.ToString(action.Content))

	_, err := page.Locator(action.Selector).First().SelectOption(playwrightgo.SelectOptionValues{
		ValuesOrLabels: &valuesOrLabels,
	})
	return err
}

func SelectMultipleOptionsAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need a <select> element to select option on")
	}
	if action.Content == "" || len(action.Content) <= 1 {
		return fmt.Errorf("need multiple elements to select from the element")
	}
	valuesOrLabels := make([]string, 0)
	// typically target comes to us a single string, so we may need to treat it as
	// a CSV to support selecting multiple options
	for _, item := range strings.Split(cast.ToString(action.Content), ",") {
		if item == "" {
			return fmt.Errorf("need a <select> element to select option on")
		}
		valuesOrLabels = append(valuesOrLabels, cast.ToString(item))
	}

	_, err := page.Locator(action.Selector).First().SelectOption(playwrightgo.SelectOptionValues{
		ValuesOrLabels: &valuesOrLabels,
	})
	return err
}
