package web

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/kevinramage/venomWeb/common"
	venomWeb "github.com/kevinramage/venomWeb/wrapper"
	"github.com/ovh/venom"
)

var (
	_ venom.Executor          = new(Executor)
	_ venom.ExecutorWithSetup = new(Executor)
)

// Key of context element in testsuite file
const (
	Name       = "web"
	ContextKey = venom.ContextKey("webContext")
)

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

type WebContext struct {
	wd      venomWeb.WebDriver
	session venomWeb.Session
}

// Executor struct
type Executor struct {
	Action     Action `json:"action,omitempty" yaml:"action"`
	Screenshot string `json:"screenshot,omitempty" yaml:"screenshot"`
}

// Result represents a step result
type Result struct {
	Find        int     `json:"find,omitempty" yaml:"find,omitempty"`
	HTML        string  `json:"html,omitempty" yaml:"html,omitempty"`
	TimeSeconds float64 `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	Title       string  `json:"title,omitempty" yaml:"title,omitempty"`
	URL         string  `json:"url,omitempty" yaml:"url,omitempty"`
	Text        string  `json:"text,omitempty" yaml:"text,omitempty"`
	Value       string  `json:"value,omitempty" yaml:"value,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

func (Executor) Setup(ctx context.Context, vars venom.H) (context.Context, error) {
	venom.Info(ctx, "Setup")
	var webCtx WebContext
	var driver = venom.StringVarFromCtx(ctx, "web.driver") // Possible values: chrome, gecko
	var args = venom.StringSliceVarFromCtx(ctx, "web.args")
	var prefs = venom.StringMapInterfaceVarFromCtx(ctx, "web.prefs")

	// Binary
	var binaryPath = venom.StringVarFromCtx(ctx, "web.binaryPath")
	var driverPath = venom.StringVarFromCtx(ctx, "web.driverPath")
	var port = venom.StringVarFromCtx(ctx, "web.driverPort")

	// Instanciate web driver (chrome by default)
	switch driver {
	case "gecko":
		webCtx.wd = venomWeb.GeckoDriver(binaryPath, driverPath, args, prefs, port)
	default:
		webCtx.wd = venomWeb.ChromeDriver(binaryPath, driverPath, args, prefs, port)
	}

	// Configure web driver
	if timeout := venom.IntVarFromCtx(ctx, "web.timeout"); timeout > 0 {
		webCtx.wd.Timeout = time.Duration(timeout) * time.Second
	}
	if debug := venom.BoolVarFromCtx(ctx, "web.debug"); debug {
		webCtx.wd.LogLevel = common.DEBUG
	}
	if proxy := venom.StringVarFromCtx(ctx, "web.proxy"); proxy != "" {
		webCtx.wd.Proxy = proxy
	}
	if headless := venom.BoolVarFromCtx(ctx, "web.headless"); headless {
		webCtx.wd.Headless = headless
	}
	if detach := venom.BoolVarFromCtx(ctx, "web.detach"); detach {
		webCtx.wd.Detach = detach
	}
	if logLevel := venom.StringVarFromCtx(ctx, "web.logLevel"); logLevel != "" {
		webCtx.wd.LogLevel = logLevel
	}

	// Init session
	var err error
	err = webCtx.wd.Start()
	if err != nil {
		return ctx, errors.Wrapf(err, "Unable to start web driver")
	}
	webCtx.session, err = webCtx.wd.NewSession()
	if err != nil {
		return ctx, errors.Wrapf(err, "Unable create new session")
	}

	var resizePage bool
	var width = venom.IntVarFromCtx(ctx, "web.width")
	var height = venom.IntVarFromCtx(ctx, "web.height")
	if width > 0 && height > 0 {
		resizePage = true
	}

	// Resize Page
	if resizePage {
		if err := webCtx.session.Size(width, height); err != nil {
			return ctx, fmt.Errorf("unable resize page: %s", err)
		}
	}

	return context.WithValue(ctx, ContextKey, &webCtx), nil
}

func getWebCtx(ctx context.Context) *WebContext {
	i := ctx.Value(ContextKey)
	if i == nil {
		return nil
	}
	return i.(*WebContext)
}

func (Executor) TearDown(ctx context.Context) error {
	venom.Info(ctx, "TearDown")
	return getWebCtx(ctx).wd.Stop()
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	webCtx := getWebCtx(ctx)

	start := time.Now()

	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	result, err := e.runAction(ctx, webCtx.session)
	if err != nil {
		if errg := generateErrorHTMLFile(ctx, webCtx.session, slug.Make(webCtx.session.String())); errg != nil {
			venom.Warn(ctx, "Error while generating the HTML file: %v", errg)
			return nil, err
		}
		return nil, err
	}

	// take a screenshot
	if e.Screenshot != "" {
		if err := webCtx.session.TakeScreenshot(e.Screenshot); err != nil {
			return nil, err
		}
		if err := generateErrorHTMLFile(ctx, webCtx.session, slug.Make(webCtx.session.String())); err != nil {
			venom.Warn(ctx, "Error while generating the HTML file: %v", err)
			return nil, err
		}
	}

	// Get page title (Check the absence of popup before the page title collect to avoid error)
	result.Title = ""
	if _, err := webCtx.session.GetAlertText(); err != nil {
		title, err := webCtx.session.GetTitle()
		if err != nil {
			return nil, err
		}
		result.Title = title
	}

	// Get page url (Check the absence of popup before the page url collect to avoid error)
	result.URL = ""
	if _, err := webCtx.session.GetAlertText(); err != nil {
		url, err := webCtx.session.GetURL()
		if err != nil {
			return nil, fmt.Errorf("cannot get url: %s", err)
		}
		result.URL = url
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func (e Executor) runAction(ctx context.Context, session venomWeb.Session) (*Result, error) {
	r := &Result{}

	// Click
	if e.Action.Click != nil {

		// Find element
		elt, err := findOne(ctx, session, e.Action.Click.Find, e.Action.Click.SyncTimeout)
		if err != nil {
			return nil, err
		}

		// Click on element
		if err := elt.Click(); err != nil {
			return nil, err
		}

		// Wait
		if e.Action.Click.Wait != 0 {
			time.Sleep(time.Duration(e.Action.Click.Wait) * time.Second)
		}

		// Fill
	} else if e.Action.Fill != nil {
		for _, f := range e.Action.Fill {
			elt, err := findOne(ctx, session, f.Find, f.SyncTimeout)
			if err != nil {
				return nil, err
			}
			if err := elt.SendKeys(f.Text); err != nil {
				return nil, err
			}
			if f.Key != nil {
				if err := elt.SendKeys(Keys[*f.Key]); err != nil {
					return nil, err
				}
			}
		}
	} else if e.Action.Find != nil {
		elts, err := find(ctx, session, e.Action.Find, r)
		if err != nil {
			return nil, err
		} else if len(elts) > 0 {
			if text, err := elts[0].GetElementText(); err == nil {
				r.Text = text
			}
			if value, err := elts[0].GetElementProperty("value"); err == nil {
				r.Value = value
			}
		}

		// Navigate
	} else if e.Action.Navigate != nil {
		if e.Action.Navigate.Reset {
			if err := session.Reset(); err != nil {
				return nil, err
			}
		}
		if err := session.Navigate(e.Action.Navigate.URL); err != nil {
			return nil, err
		}

		// Wait
	} else if e.Action.Wait != 0 {
		time.Sleep(time.Duration(e.Action.Wait) * time.Second)

		// Confirm popup
	} else if e.Action.ConfirmPopup {
		if err := session.AcceptAlert(); err != nil {
			return nil, err
		}

		// Cancel popup
	} else if e.Action.CancelPopup {
		if err := session.DismissAlert(); err != nil {
			return nil, err
		}

		// Select
	} else if e.Action.Select != nil {
		elt, err := findOne(ctx, session, e.Action.Select.Find, e.Action.Select.SyncTimeout)
		if err != nil {
			return nil, err
		}
		if _, err := elt.Select(e.Action.Select.Text); err != nil {
			return nil, err
		}
		if e.Action.Select.Wait != 0 {
			time.Sleep(time.Duration(e.Action.Select.Wait) * time.Second)
		}

		// Upload file
	} else if e.Action.UploadFile != nil {
		elt, err := findOne(ctx, session, e.Action.UploadFile.Find, e.Action.UploadFile.SyncTimeout)
		if err != nil {
			return nil, err
		}
		for _, f := range e.Action.UploadFile.Files {
			if err := elt.UploadFile(f); err != nil {
				return nil, err
			}
		}
		if e.Action.UploadFile.Wait != 0 {
			time.Sleep(time.Duration(e.Action.UploadFile.Wait) * time.Second)
		}

		// Select frame
	} else if e.Action.SelectFrame != nil {
		elt, err := findOne(ctx, session, e.Action.SelectFrame.Find, e.Action.SelectFrame.SyncTimeout)
		if err != nil {
			return nil, err
		}
		err = session.SwitchToFrame(elt)
		if err != nil {
			return nil, err
		}

		// Select root frame
	} else if e.Action.SelectRootFrame {
		if err := session.SwitchToParentFrame(); err != nil {
			return nil, err
		}

		// Next window
	} else if e.Action.NextWindow {
		if err := session.NextWindow(); err != nil {
			return nil, err
		}

		// Back, Forward, Refresh
	} else if e.Action.HistoryAction != "" {
		switch strings.ToLower(e.Action.HistoryAction) {
		case "back":
			if err := session.Back(); err != nil {
				return nil, err
			}
		case "refresh":
			if err := session.Refresh(); err != nil {
				return nil, err
			}
		case "forward":
			if err := session.Forward(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("history action '%s' is invalid", e.Action.HistoryAction)
		}

		// Execute
	} else if e.Action.Execute != nil {
		args := []string{}
		if e.Action.Execute.Args != nil && len(e.Action.Execute.Args) > 0 {
			args = e.Action.Execute.Args
		}
		err := session.ExecuteScript(e.Action.Execute.Command, args)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func find(ctx context.Context, session venomWeb.Session, findElement interface{}, r *Result) ([]venomWeb.Element, error) {

	// Identify selector and locator strategy
	selector, locator, err := readFindElement(ctx, findElement)
	if err != nil {
		return nil, err
	}

	// Search elements
	elts, err := session.FindElements(selector, locator)
	if err != nil {
		return nil, err
	}
	r.Find = len(elts)

	// Update result
	if r.Find == 1 {
		r.Text, _ = elts[0].GetElementText()
		r.Value, _ = elts[0].GetElementProperty("value")
	}

	return elts, nil
}

// Find element from a selector
func findOne(ctx context.Context, session venomWeb.Session, findElement interface{}, syncTimeout int64) (venomWeb.Element, error) {

	// Identify selector and locator strategy
	selector, locator, err := readFindElement(ctx, findElement)
	if err != nil {
		return venomWeb.Element{}, err
	}

	// Synchronize element
	if syncTimeout > 0 {
		if err := session.SyncElement(selector, locator, syncTimeout*1000); err != nil {
			return venomWeb.Element{}, err
		}
	}

	// Find element
	return session.FindElement(selector, locator)
}

// Identify selector and locator from a find element
func readFindElement(ctx context.Context, findElement interface{}) (string, string, error) {
	selector := ""
	locator := common.CSS_SELECTOR
	if find, ok := findElement.(string); ok {
		venom.Warning(ctx, "web - findElement : this find element syntax deprecated and will be not supported in next version")
		selector = find
	} else if find, ok := findElement.(map[string]interface{}); ok {
		selector = find["selector"].(string)
		if find["locator"] == "XPATH" {
			locator = common.XPATH_SELECTOR
		} else if find["locator"] != "CSS" {
			return "", "", fmt.Errorf("invalid find element locator '%s' (must be CSS or XPATH)", find["locator"])
		}
	} else {
		return "", "", fmt.Errorf("invalid find element structure %v\n (must be a string or a element with selector and locator attribute)", findElement)
	}

	return selector, locator, nil
}

// generateErrorHTMLFile generates an HTML file in error case to identify clearly the error
func generateErrorHTMLFile(ctx context.Context, session venomWeb.Session, name string) error {
	html, err := session.GetPageSource()
	if err != nil {
		return err
	}
	filename := name + ".dump.html"
	venom.Info(ctx, "Content of the HTML page is saved in %s", filename)
	return os.WriteFile(filename, []byte(html), 0644)
}
