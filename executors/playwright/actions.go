package playwright

import (
	"fmt"
	"strings"

	playwrightgo "github.com/playwright-community/playwright-go"
	"github.com/spf13/cast"
)

type ActionFunc func(page playwrightgo.Page, element string, target ...any) error

var actionMap = map[string]ActionFunc{
	"Click":           ClickAction,
	"Fill":            FillAction,
	"Press":           PressAction,
	"WaitFor":         WaitForSelectorAction,
	"WaitForSelector": WaitForSelectorAction,
	"WaitForURL":      WaitForURLAction,
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
