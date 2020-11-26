package web

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sclevine/agouti"

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
	wd   *agouti.WebDriver
	Page *agouti.Page
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
	var args = venom.StringVarFromCtx(ctx, "web.args")
	var prefs = venom.StringMapInterfaceVarFromCtx(ctx, "web.prefs")

	switch driver {
	case "chrome":
		webCtx.wd = agouti.ChromeDriver(
			agouti.ChromeOptions("args", args),
			agouti.ChromeOptions("prefs", prefs),
		)
	case "gecko":
		webCtx.wd = agouti.GeckoDriver()
	default:
		webCtx.wd = agouti.PhantomJS()
	}

	var timeout = venom.IntVarFromCtx(ctx, "web.timeout")
	if timeout > 0 {
		webCtx.wd.Timeout = time.Duration(timeout) * time.Second
	} else {
		webCtx.wd.Timeout = 180 * time.Second // default value
	}

	webCtx.wd.Debug = venom.BoolVarFromCtx(ctx, "web.debug")

	if err := webCtx.wd.Start(); err != nil {
		return ctx, errors.Wrapf(err, "Unable start web driver")
	}

	// Init Page
	var err error
	webCtx.Page, err = webCtx.wd.NewPage()
	if err != nil {
		return ctx, errors.Wrapf(err, "Unable create new page")
	}

	var resizePage bool
	var width = venom.IntVarFromCtx(ctx, "web.width")
	var height = venom.IntVarFromCtx(ctx, "web.height")
	if width > 0 && height > 0 {
		resizePage = true
	}

	// Resize Page
	if resizePage {
		if err := webCtx.Page.Size(width, height); err != nil {
			return ctx, fmt.Errorf("Unable resize page: %s", err)
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

	result, err := e.runAction(ctx, webCtx.Page)
	if err != nil {
		if errg := generateErrorHTMLFile(ctx, webCtx.Page, slug.Make(webCtx.Page.String())); errg != nil {
			venom.Warn(ctx, "Error while generate HTML file: %v", errg)
			return nil, err
		}
		return nil, err
	}

	// take a screenshot
	if e.Screenshot != "" {
		if err := webCtx.Page.Screenshot(e.Screenshot); err != nil {
			return nil, err
		}
		if err := generateErrorHTMLFile(ctx, webCtx.Page, slug.Make(webCtx.Page.String())); err != nil {
			venom.Warn(ctx, "Error while generate HTML file: %v", err)
			return nil, err
		}
	}

	// Get page title (Check the absence of popup before the page title collect to avoid error)
	if _, err := webCtx.Page.PopupText(); err != nil {
		title, err := webCtx.Page.Title()
		if err != nil {
			return nil, err
		}
		result.Title = title
	}

	// Get page url (Check the absence of popup before the page url collect to avoid error)
	if _, err := webCtx.Page.PopupText(); err != nil {
		url, errU := webCtx.Page.URL()
		if errU != nil {
			return nil, fmt.Errorf("Cannot get URL: %s", errU)
		}
		result.URL = url
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func (e Executor) runAction(ctx context.Context, page *agouti.Page) (*Result, error) {
	r := &Result{}
	if e.Action.Click != nil {
		s, err := find(page, e.Action.Click.Find, r)
		if err != nil {
			return nil, err
		}
		if err := s.Click(); err != nil {
			return nil, err
		}
		if e.Action.Click.Wait != 0 {
			time.Sleep(time.Duration(e.Action.Click.Wait) * time.Second)
		}
	} else if e.Action.Fill != nil {
		for _, f := range e.Action.Fill {
			s, err := findOne(page, f.Find, r)
			if err != nil {
				return nil, err
			}
			if err := s.Fill(f.Text); err != nil {
				return nil, err
			}
			if f.Key != nil {
				if err := s.SendKeys(Keys[*f.Key]); err != nil {
					return nil, err
				}
			}
		}
	} else if e.Action.Find != "" {
		s, err := find(page, e.Action.Find, r)
		if err != nil {
			return nil, err
		} else if s != nil {
			if count, errCount := s.Count(); errCount == nil && count == 1 {
				if elements, errElements := s.Elements(); errElements == nil {
					if text, errText := elements[0].GetText(); errText == nil {
						r.Text = text
					}
					if value, errValue := elements[0].GetAttribute("value"); errValue == nil {
						r.Value = value
					}
				}
			}
		}
	} else if e.Action.Navigate != nil {
		if err := page.Navigate(e.Action.Navigate.URL); err != nil {
			return nil, err
		}
		if e.Action.Navigate.Reset {
			if err := page.Reset(); err != nil {
				return nil, err
			}
			if err := page.Navigate(e.Action.Navigate.URL); err != nil {
				return nil, err
			}
		}
	} else if e.Action.Wait != 0 {
		time.Sleep(time.Duration(e.Action.Wait) * time.Second)
	} else if e.Action.ConfirmPopup {
		if err := page.ConfirmPopup(); err != nil {
			return nil, err
		}
	} else if e.Action.CancelPopup {
		if err := page.CancelPopup(); err != nil {
			return nil, err
		}
	} else if e.Action.Select != nil {
		s, err := findOne(page, e.Action.Select.Find, r)
		if err != nil {
			return nil, err
		}
		if err := s.Select(e.Action.Select.Text); err != nil {
			return nil, err
		}
		if e.Action.Select.Wait != 0 {
			time.Sleep(time.Duration(e.Action.Select.Wait) * time.Second)
		}
	} else if e.Action.UploadFile != nil {
		s, err := find(page, e.Action.UploadFile.Find, r)
		if err != nil {
			return nil, err
		}
		for _, f := range e.Action.UploadFile.Files {
			if err := s.UploadFile(f); err != nil {
				return nil, err
			}
		}
		if e.Action.UploadFile.Wait != 0 {
			time.Sleep(time.Duration(e.Action.UploadFile.Wait) * time.Second)
		}
	} else if e.Action.SelectFrame != nil {
		s, err := findOne(page, e.Action.SelectFrame.Find, r)
		if err != nil {
			return nil, err
		}
		if elements, errElements := s.Elements(); errElements == nil {
			if errSelectFrame := page.Session().Frame(elements[0]); errSelectFrame != nil {
				return nil, errSelectFrame
			}
		} else {
			return nil, errElements
		}
	} else if e.Action.SelectRootFrame {
		if err := page.SwitchToRootFrame(); err != nil {
			return nil, err
		}
	} else if e.Action.NextWindow {
		if err := page.NextWindow(); err != nil {
			return nil, err
		}
	} else if e.Action.HistoryAction != "" {
		switch strings.ToLower(e.Action.HistoryAction) {
		case "back":
			if err := page.Back(); err != nil {
				return nil, err
			}
		case "refresh":
			if err := page.Refresh(); err != nil {
				return nil, err
			}
		case "forward":
			if err := page.Forward(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("History action '%s' is invalid", e.Action.HistoryAction)
		}
	}
	return r, nil
}

func find(page *agouti.Page, search string, r *Result) (*agouti.Selection, error) {
	s := page.Find(search)
	if s == nil {
		return nil, fmt.Errorf("Cannot find element %s", search)
	}
	nbElement, errC := s.Count()
	if errC != nil {
		if !strings.Contains(errC.Error(), "element not found") {
			return nil, fmt.Errorf("Cannot count element %s: %s", search, errC)
		}
		nbElement = 0
	}
	r.Find = nbElement
	return s, nil
}

func findOne(page *agouti.Page, search string, r *Result) (*agouti.Selection, error) {
	s := page.Find(search)
	if s == nil {
		return nil, fmt.Errorf("Cannot find element %s", search)
	}
	nbElement, errC := s.Count()
	if errC != nil {
		return nil, fmt.Errorf("Cannot find element %s: %s", search, errC)
	}
	if nbElement != 1 {
		return nil, fmt.Errorf("Find %d elements", nbElement)
	}
	return s, nil
}

// generateErrorHTMLFile generates an HTML file in error case to identify clearly the error
func generateErrorHTMLFile(ctx context.Context, page *agouti.Page, name string) error {
	html, err := page.HTML()
	if err != nil {
		return err
	}
	filename := name + ".dump.html"
	venom.Info(ctx, "Content of the HTML page is saved in %s", filename)
	return ioutil.WriteFile(filename, []byte(html), 0644)
}
