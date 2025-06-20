package playwright

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
	"Screenshot":            ScreenshotAction,
	"Upload":                UploadFileAction, // alias for UploadFile
	"UploadFile":            UploadFileAction,
	"UploadFiles":           UploadMultipleFilesAction, // alias for UploadMultipleFiles
	"UploadMultipleFiles":   UploadMultipleFilesAction,
}

func castOptions[Dest any](action *ExecutorAction) (dest *Dest, err error) {
	data, err := json.Marshal(action.Options)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func ClickAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorClickOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorClickOptions")
		}
		return page.Locator(action.Selector).Click(*options)
	}
	return page.Locator(action.Selector).Click()
}

func WaitForSelectorAction(page playwrightgo.Page, action *ExecutorAction) error {
	timeout := 10_000.00
	defaultOptions := playwrightgo.LocatorWaitForOptions{
		Timeout: &timeout,
		State:   playwrightgo.WaitForSelectorStateAttached,
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorWaitForOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorWaitForOptions")
		}
		return page.Locator(action.Selector).WaitFor(*options)
	}
	return page.Locator(action.Selector).WaitFor(defaultOptions)
}

func WaitForURLAction(page playwrightgo.Page, action *ExecutorAction) error {
	timeout := 10_000.00
	defaultOptions := playwrightgo.PageWaitForURLOptions{
		Timeout:   &timeout,
		WaitUntil: playwrightgo.WaitUntilStateCommit,
	}

	if action.Options != nil {
		options, err := castOptions[playwrightgo.PageWaitForURLOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse PageWaitForURLOptions")
		}
		return page.WaitForURL(*options)
	}

	urlPattern := action.Selector
	return page.WaitForURL(urlPattern, defaultOptions)
}

func FillAction(page playwrightgo.Page, action *ExecutorAction) error {
	element := action.Selector
	target := action.Content
	if target == "" {
		return fmt.Errorf("need data to fill on '%s'", element)
	}

	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorFillOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorFillOptions")
		}
		return page.Locator(element).First().Fill(cast.ToString(target), *options)
	}
	return page.Locator(element).First().Fill(cast.ToString(target))
}

func PressAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Content == "" {
		return fmt.Errorf("need key to press on '%s'", action.Selector)
	}

	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorPressOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorPressOptions")
		}
		return page.Locator(action.Selector).First().Press(cast.ToString(action.Content), *options)
	}
	return page.Locator(action.Selector).First().Press(cast.ToString(action.Content))
}

func PressSequentiallyAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Content == "" {
		return fmt.Errorf("need key to press on '%s'", action.Selector)
	}

	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorPressSequentiallyOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorPressSequentiallyOptions")
		}
		return page.Locator(action.Selector).First().PressSequentially(cast.ToString(action.Content), *options)
	}
	return page.Locator(action.Selector).First().PressSequentially(cast.ToString(action.Content))
}

func DoubleClickAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to double click on")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorDblclickOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorDblclickOptions")
		}
		return page.Locator(action.Selector).First().Dblclick(*options)
	}
	return page.Locator(action.Selector).First().Dblclick()
}

func TapAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to tap on")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorTapOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorTapOptions")
		}
		return page.Locator(action.Selector).First().Tap(*options)
	}
	return page.Locator(action.Selector).First().Tap()
}

func FocusAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to focus on")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorFocusOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorFocusOptions")
		}
		return page.Locator(action.Selector).First().Focus(*options)
	}
	return page.Locator(action.Selector).First().Focus()
}

func BlurAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to blur")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorBlurOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorBlurOptions")
		}
		return page.Locator(action.Selector).First().Blur(*options)
	}
	return page.Locator(action.Selector).First().Blur()
}

func ClearAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to blur")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorClearOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorClearOptions")
		}
		return page.Locator(action.Selector).First().Clear(*options)
	}
	return page.Locator(action.Selector).First().Clear()
}

func CheckAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to check on")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorCheckOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorCheckOptions")
		}
		return page.Locator(action.Selector).First().Check(*options)
	}
	return page.Locator(action.Selector).First().Check()
}

func UncheckAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Selector == "" {
		return fmt.Errorf("need element to uncheck on")
	}
	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorUncheckOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorUncheckOptions")
		}
		return page.Locator(action.Selector).First().Uncheck(*options)
	}
	return page.Locator(action.Selector).First().Uncheck()
}

func RefreshAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Options != nil {
		options, err := castOptions[playwrightgo.PageReloadOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse PageReloadOptions")
		}
		_, err = page.Reload(*options)
		return err
	}
	_, err := page.Reload()
	return err
}

func GoBackAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Options != nil {
		options, err := castOptions[playwrightgo.PageGoBackOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse PageGoBackOptions")
		}
		_, err = page.GoBack(*options)
		return err
	}
	_, err := page.GoBack()
	return err
}

func GoForwardAction(page playwrightgo.Page, action *ExecutorAction) error {
	if action.Options != nil {
		options, err := castOptions[playwrightgo.PageGoForwardOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse PageGoForwardOptions")
		}
		_, err = page.GoForward(*options)
		return err
	}
	_, err := page.GoForward()
	return err
}

func GotoAction(page playwrightgo.Page, action *ExecutorAction) error {
	urlPattern := action.Selector
	timeout := 10_000.00
	defaultOptions := playwrightgo.PageGotoOptions{
		Timeout:   &timeout,
		WaitUntil: playwrightgo.WaitUntilStateCommit,
	}

	options := defaultOptions
	if action.Options != nil {
		userOpts, err := castOptions[playwrightgo.PageGotoOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse PageGotoOptions")
		}
		options = *userOpts
	}

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
	_, err := page.Goto(finalURL, options)
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

	selectOptionValues := playwrightgo.SelectOptionValues{
		ValuesOrLabels: &valuesOrLabels,
	}

	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorSelectOptionOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorSelectOptionOptions")
		}
		_, err = page.Locator(action.Selector).First().SelectOption(selectOptionValues, *options)
		return err
	}

	_, err := page.Locator(action.Selector).First().SelectOption(selectOptionValues)
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

	selectOptionValues := playwrightgo.SelectOptionValues{
		ValuesOrLabels: &valuesOrLabels,
	}

	if action.Options != nil {
		options, err := castOptions[playwrightgo.LocatorSelectOptionOptions](action)
		if err != nil {
			return fmt.Errorf("failed to parse LocatorSelectOptionOptions")
		}
		_, err = page.Locator(action.Selector).First().SelectOption(selectOptionValues, *options)
		return err
	}

	_, err := page.Locator(action.Selector).First().SelectOption(selectOptionValues)
	return err
}

func ScreenshotAction(page playwrightgo.Page, action *ExecutorAction) error {
	opts, err := castOptions[playwrightgo.PageScreenshotOptions](action)
	if err != nil {
		return err
	}
	defaultTimeout := 10_000.00
	fullPage := true
	caretOption := playwrightgo.ScreenshotCaret("hide")
	defaultOpts := playwrightgo.PageScreenshotOptions{
		Caret:    &caretOption,
		FullPage: &fullPage,
		Timeout:  &defaultTimeout,
	}
	if opts == nil {
		opts = &defaultOpts
	}
	screenshotBytes, err := page.Screenshot(*opts)
	if err != nil {
		return err
	}
	err = os.WriteFile(action.Content, screenshotBytes, 0o775)
	return err
}

func UploadFileAction(page playwrightgo.Page, action *ExecutorAction) error {
	//   - files should be one of: string, []string, InputFile, []InputFile,
	//     string: local file path
	filename := action.Content
	noWaitAfter := true
	timeout := 10_000.00
	opts, err := castOptions[playwrightgo.LocatorSetInputFilesOptions](action)
	if err != nil {
		return err
	}
	defaultOpts := playwrightgo.LocatorSetInputFilesOptions{
		NoWaitAfter: &noWaitAfter,
		Timeout:     &timeout,
	}
	if opts == nil {
		opts = &defaultOpts
	}
	return page.Locator(action.Selector).First().SetInputFiles(filename, *opts)
}

func UploadMultipleFilesAction(page playwrightgo.Page, action *ExecutorAction) error {
	files := action.Content
	noWaitAfter := true
	timeout := 10_000.00
	opts, err := castOptions[playwrightgo.LocatorSetInputFilesOptions](action)
	if err != nil {
		return err
	}
	defaultOpts := playwrightgo.LocatorSetInputFilesOptions{
		NoWaitAfter: &noWaitAfter,
		Timeout:     &timeout,
	}
	if opts == nil {
		opts = &defaultOpts
	}
	return page.Locator(action.Selector).First().SetInputFiles(files, *opts)
}
