package web

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/sclevine/agouti"

	"github.com/ovh/venom"
	"github.com/ovh/venom/context/webctx"
	"github.com/ovh/venom/executors"
)

// Name of executor
const Name = "web"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor struct
type Executor struct {
	Action     Action `json:"action,omitempty" yaml:"action"`
	Screenshot string `json:"screenshot,omitempty" yaml:"screenshot"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Find        int      `json:"find,omitempty" yaml:"find,omitempty"`
	HTML        string   `json:"html,omitempty" yaml:"html,omitempty"`
	TimeSeconds float64  `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string   `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	URL         string   `json:"url,omitempty" yaml:"url,omitempty"`
	Text        string   `json:"text,omitempty" yaml:"text,omitempty"`
	Value       string   `json:"value,omitempty" yaml:"value,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// Run execute TestStep
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	var ctx *webctx.WebTestCaseContext
	switch testCaseContext.(type) {
	case *webctx.WebTestCaseContext:
		ctx = testCaseContext.(*webctx.WebTestCaseContext)
	default:
		return nil, fmt.Errorf("Web executor need a Web context")
	}

	start := time.Now()

	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	r := &Result{Executor: e}

	// Check action to realise
	if e.Action.Click != nil {
		s, err := find(testCaseContext, ctx.Page, e.Action.Click.Find, r)
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
			s, err := findOne(testCaseContext, ctx.Page, f.Find, r)
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
		s, err := find(testCaseContext, ctx.Page, e.Action.Find, r)
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
		if err := ctx.Page.Navigate(e.Action.Navigate.Url); err != nil {
			return nil, err
		}
		if e.Action.Navigate.Reset {
			if err := ctx.Page.Reset(); err != nil {
				return nil, err
			}
			if err := ctx.Page.Navigate(e.Action.Navigate.Url); err != nil {
				return nil, err
			}
		}
	} else if e.Action.Wait != 0 {
		time.Sleep(time.Duration(e.Action.Wait) * time.Second)
	} else if e.Action.ConfirmPopup {
		if ctx.TestCase.Context["driver"] != "phantomjs" {
			if err := ctx.Page.ConfirmPopup(); err != nil {
				return nil, err
			}
		}
	} else if e.Action.CancelPopup {
		if ctx.TestCase.Context["driver"] != "phantomjs" {
			if err := ctx.Page.CancelPopup(); err != nil {
				return nil, err
			}
		}
	} else if e.Action.Select != nil {
		s, err := findOne(testCaseContext, ctx.Page, e.Action.Select.Find, r)
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
		s, err := find(testCaseContext, ctx.Page, e.Action.UploadFile.Find, r)
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
		s, err := findOne(testCaseContext, ctx.Page, e.Action.SelectFrame.Find, r)
		if err != nil {
			return nil, err
		}
		if elements, errElements := s.Elements(); errElements == nil {
			if errSelectFrame := ctx.Page.Session().Frame(elements[0]); errSelectFrame != nil {
				return nil, errSelectFrame
			}
		} else {
			return nil, errElements
		}
	} else if e.Action.SelectRootFrame {
		if err := ctx.Page.SwitchToRootFrame(); err != nil {
			return nil, err
		}
	} else if e.Action.NextWindow {
		if err := ctx.Page.NextWindow(); err != nil {
			return nil, err
		}
	}

	// take a screenshot
	if e.Screenshot != "" {
		if err := ctx.Page.Screenshot(e.Screenshot); err != nil {
			return nil, err
		}
	}

	// Get page title (Check the absence of popup before the page title collect to avoid error)
	if _, err := ctx.Page.PopupText(); err != nil {
		title, err := ctx.Page.Title()
		if err != nil {
			return nil, err
		}
		r.Title = title
	}

	// Get page url (Check the absence of popup before the page url collect to avoid error)
	if _, err := ctx.Page.PopupText(); err != nil {
		url, errU := ctx.Page.URL()
		if errU != nil {
			return nil, fmt.Errorf("Cannot get URL: %s", errU)
		}
		r.URL = url
	}

	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	return executors.Dump(r)
}

func find(testCaseContext venom.TestCaseContext, page *agouti.Page, search string, r *Result) (*agouti.Selection, error) {
	s := page.Find(search)
	if s == nil {
		generateHtmlFile(testCaseContext, page)
		return nil, fmt.Errorf("Cannot find element %s", search)
	}
	nbElement, errC := s.Count()
	if errC != nil {
		generateHtmlFile(testCaseContext, page)
		if !strings.Contains(errC.Error(), "element not found") {
			return nil, fmt.Errorf("Cannot count element %s: %s", search, errC)
		}
		nbElement = 0
	}
	r.Find = nbElement
	return s, nil
}

func findOne(testCaseContext venom.TestCaseContext, page *agouti.Page, search string, r *Result) (*agouti.Selection, error) {
	s := page.Find(search)
	if s == nil {
		generateHtmlFile(testCaseContext, page)
		return nil, fmt.Errorf("Cannot find element %s", search)
	}
	nbElement, errC := s.Count()
	if errC != nil {
		generateHtmlFile(testCaseContext, page)
		return nil, fmt.Errorf("Cannot find element %s: %s", search, errC)
	}
	if nbElement != 1 {
		generateHtmlFile(testCaseContext, page)
		return nil, fmt.Errorf("Find %d elements", nbElement)
	}
	return s, nil
}

// Generate an HTML file in error case to identify clearly the error
func generateHtmlFile(context venom.TestCaseContext, page *agouti.Page) {
	fmt.Printf("GenerateHtmlFile ...\n")
	html, err := page.HTML()
	if err == nil {
		fileName := context.GetName() + ".html"
		fmt.Printf("Error during an element selection, the content of the HTML page is saved in %s\n", fileName)
		ioutil.WriteFile(fileName, []byte(html), 0644)
	}
}
