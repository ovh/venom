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
	//wd   *agouti.WebDriver
	//Page *agouti.Page
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
	var webCtx WebContext
	var driver = venom.StringVarFromCtx(ctx, "web.driver") // Possible values: chrome, phantomjs, gecko
	var args = venom.StringSliceVarFromCtx(ctx, "web.args")
	//var prefs = venom.StringMapInterfaceVarFromCtx(ctx, "web.prefs")

	switch driver {
	case "gecko":
		webCtx.wd = venomWeb.GeckoDriver(args)
	default:
		webCtx.wd = venomWeb.ChromeDriver(args)
	}

	var timeout = venom.IntVarFromCtx(ctx, "web.timeout")
	if timeout > 0 {
		//webCtx.wd.t = time.Duration(timeout) * time.Second
	} else {
		//webCtx.wd.Timeout = 180 * time.Second // default value
	}

	webCtx.wd.Proxy = venom.StringVarFromCtx(ctx, "web.proxy")
	webCtx.wd.Headless = venom.BoolVarFromCtx(ctx, "web.headless")
	webCtx.wd.Detach = venom.BoolVarFromCtx(ctx, "web.detach")
	webCtx.wd.LogLevel = venom.StringVarFromCtx(ctx, "web.logLevel")

	if err := webCtx.wd.Start(); err != nil {
		return ctx, errors.Wrapf(err, "Unable to start the web driver")
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
			return ctx, fmt.Errorf("Unable to resize the page: %s", err)
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
	/*
		if _, err := webCtx.Page.PopupText(); err != nil {
			title, err := webCtx.session.GetTitle()
			if err != nil {
				return nil, err
			}
			result.Title = title
		}
	*/

	// Get page url (Check the absence of popup before the page url collect to avoid error)
	/*
		if _, err := webCtx.Page.PopupText(); err != nil {
			url, errU := webCtx.session.URL()
			if errU != nil {
				return nil, fmt.Errorf("Cannot get URL: %s", errU)
			}
			result.URL = url
		}
	*/

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func (e Executor) runAction(ctx context.Context, session venomWeb.Session) (*Result, error) {
	r := &Result{}
	if e.Action.Click != nil {
		elt, err := findOne(session, e.Action.Click.Find)
		if err != nil {
			return nil, err
		}
		if err := elt.Click(); err != nil {
			return nil, err
		}
		if e.Action.Click.Wait != 0 {
			time.Sleep(time.Duration(e.Action.Click.Wait) * time.Second)
		}
	} else if e.Action.Fill != nil {
		for _, f := range e.Action.Fill {
			elt, err := findOne(session, f.Find)
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
	} else if e.Action.Find != "" {
		elts, err := find(session, e.Action.Find, r)
		if err != nil {
			return nil, err
		} else if elts != nil && len(elts) > 0 {
			if text, err := elts[0].GetElementText(); err == nil {
				r.Text = text
			}
			if value, err := elts[0].GetElementProperty("value"); err == nil {
				r.Value = value
			}
		}
	} else if e.Action.Navigate != nil {
		if err := session.Navigate(e.Action.Navigate.URL); err != nil {
			return nil, err
		}
		/*
			if e.Action.Navigate.Reset {
				if err := session.Reset(); err != nil {
					return nil, err
				}
				if err := session.Navigate(e.Action.Navigate.URL); err != nil {
					return nil, err
				}
			}
		*/
	} else if e.Action.Wait != 0 {
		time.Sleep(time.Duration(e.Action.Wait) * time.Second)
	} else if e.Action.ConfirmPopup {
		/*
			if err := page.ConfirmPopup(); err != nil {
				return nil, err
			}
		*/
	} else if e.Action.CancelPopup {
		/*
			if err := session.CancelPopup(); err != nil {
				return nil, err
			}
		*/
	} else if e.Action.Select != nil {
		elt, err := findOne(session, e.Action.Select.Find)
		if err != nil {
			return nil, err
		}
		if _, err := elt.Select(e.Action.Select.Text); err != nil {
			return nil, err
		}
		if e.Action.Select.Wait != 0 {
			time.Sleep(time.Duration(e.Action.Select.Wait) * time.Second)
		}
	} else if e.Action.UploadFile != nil {
		elt, err := findOne(session, e.Action.UploadFile.Find)
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
	} else if e.Action.SelectFrame != nil {
		elt, err := findOne(session, e.Action.SelectFrame.Find)
		if err != nil {
			return nil, err
		}
		return nil, session.SwitchToFrame(elt)
	} else if e.Action.SelectRootFrame {
		if err := session.SwitchToParentFrame(); err != nil {
			return nil, err
		}
	} else if e.Action.NextWindow {
		/*
			if err := session.NextWindow(); err != nil {
				return nil, err
			}
		*/
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
			return nil, fmt.Errorf("History action '%s' is invalid", e.Action.HistoryAction)
		}
	}
	return r, nil
}

func find(session venomWeb.Session, search string, r *Result) ([]venomWeb.Element, error) {
	elts, err := session.FindElements(search, common.CSS_SELECTOR)
	if err != nil {
		return nil, err
	}
	r.Find = len(elts)
	return elts, nil
}

// Find element from a selector
func findOne(session venomWeb.Session, search string) (venomWeb.Element, error) {
	return session.FindElement(search, common.CSS_SELECTOR)
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
