package playwright

import (
	"fmt"
	"strings"

	playwrightgo "github.com/playwright-community/playwright-go"
	"github.com/spf13/cast"
)

type ActionFunc func(page playwrightgo.Page, element string, target ...any) error

var actionMap = map[string]ActionFunc{
	"Click":             ClickAction,
	"DoubleClick":       DoubleClickAction,
	"Doubleclick":       DoubleClickAction,
	"Tap":               TapAction,
	"Focus":             FocusAction,
	"Blur":              BlurAction,
	"Clear":             ClearAction,
	"Fill":              FillAction,
	"Check":             CheckAction,
	"Uncheck":           UncheckAction,
	"FillCheckbox":      CheckAction, // alias for Check
	"Press":             PressAction,
	"PressSequentially": PressSequentiallyAction,
	"Type":              PressSequentiallyAction, // alias for PressSequentially
	"WaitFor":           WaitForSelectorAction,
	"WaitForSelector":   WaitForSelectorAction,
	"WaitForURL":        WaitForURLAction,
}

func removeQuotes(selector string) string {
	return strings.TrimSuffix(strings.TrimPrefix(selector, `"`), `"`)
}

func ClickAction(page playwrightgo.Page, element string, target ...any) error {
	return page.Locator(element).Click()
}

func WaitForSelectorAction(page playwrightgo.Page, selector string, target ...any) error {
	timeout := 10_000.00
	return page.Locator(selector).WaitFor(playwrightgo.LocatorWaitForOptions{
		Timeout: &timeout,
		State:   playwrightgo.WaitForSelectorStateAttached,
	})
}

func WaitForURLAction(page playwrightgo.Page, urlPattern string, target ...any) error {
	timeout := 10_000.00
	return page.WaitForURL(urlPattern, playwrightgo.PageWaitForURLOptions{
		Timeout:   &timeout,
		WaitUntil: playwrightgo.WaitUntilStateCommit,
	})
}

func FillAction(page playwrightgo.Page, element string, target ...any) error {
	if target == nil {
		return fmt.Errorf("need data to fill on '%s'", element)
	}
	return page.Locator(element).First().Fill(cast.ToString(target[0]))
}

func PressAction(page playwrightgo.Page, element string, target ...any) error {
	if target == nil {
		return fmt.Errorf("need key to press on '%s'", element)
	}
	return page.Locator(element).First().Press(cast.ToString(target[0]))
}

func PressSequentiallyAction(page playwrightgo.Page, element string, target ...any) error {
	if target == nil {
		return fmt.Errorf("need key to press on '%s'", element)
	}
	return page.Locator(element).First().PressSequentially(cast.ToString(target[0]))
}

func DoubleClickAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to double click on")
	}
	// TODO: support passing double click options
	return page.Locator(element).First().Dblclick()
}

func TapAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to tap on")
	}
	// TODO: support passing Tap options
	return page.Locator(element).First().Tap()
}

func FocusAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to focus on")
	}
	// TODO: support passing Focus options
	return page.Locator(element).First().Focus()
}

func BlurAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to blur")
	}
	// TODO: support passing Blur options
	return page.Locator(element).First().Blur()
}

func ClearAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to blur")
	}
	// TODO: support passing Clear options
	return page.Locator(element).First().Clear()
}

func CheckAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to check on")
	}
	// TODO: support passing Check options
	return page.Locator(element).First().Check()
}

func UncheckAction(page playwrightgo.Page, element string, target ...any) error {
	if element == "" {
		return fmt.Errorf("need element to uncheck on")
	}
	// TODO: support passing Uncheck options
	return page.Locator(element).First().Uncheck()
}
