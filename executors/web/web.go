package web

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/sclevine/agouti"

	"github.com/runabove/venom"
	"github.com/runabove/venom/context/webctx"
	"strings"
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
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, l *log.Entry, aliases venom.Aliases, step venom.TestStep) (venom.ExecutorResult, error) {
	start := time.Now()

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}
	r := &Result{Executor: t}

	// Get Web Context
	varContext := ctx.Value(venom.ContextKey).(map[string]interface{})
	if varContext == nil {
		return nil, errors.New("Executor web need a context")
	}
	if _, ok := varContext[webctx.ContextPageKey]; !ok {
		return nil, errors.New("Executor web need a page in context")
	}
	page := varContext[webctx.ContextPageKey].(*agouti.Page)
	if page == nil {
		return nil, errors.New("page is nil in context")
	}

	// Check action to realise
	if t.Action.Click != "" {
		s, err := find(page, t.Action.Click, r)
		if err != nil {
			return nil, failWithScreenshot(varContext, page, err)
		}
		if err := s.Click(); err != nil {
			return nil, failWithScreenshot(varContext, page, fmt.Errorf("Cannot click on element %s: %s", t.Action.Click, err))
		}
	} else if t.Action.Fill != nil {
		for _, f := range t.Action.Fill {
			s, err := findOne(page, f.Find, r)
			if err != nil {
				return nil, failWithScreenshot(varContext, page, err)
			}
			if err := s.Fill(f.Text); err != nil {
				return nil, failWithScreenshot(varContext, page, fmt.Errorf("Cannot fill element %s: %s", f.Find, err))
			}
		}

	} else if t.Action.Find != "" {
		_, err := find(page, t.Action.Find, r)
		if err != nil {
			return nil, failWithScreenshot(varContext, page, err)
		}
	} else if t.Action.Navigate != "" {
		if err := page.Navigate(t.Action.Navigate); err != nil {
			return nil, err
		}
	}

	// take a screenshot
	if t.Screenshot != "" {
		if err := page.Screenshot(t.Screenshot); err != nil {
			return nil, err
		}
	}

	// get page title
	title, err := page.Title()
	if err != nil {
		return nil, failWithScreenshot(varContext, page, fmt.Errorf("Cannot get title: %s", err))
	}
	r.Title = title

	url, errU := page.URL()
	if errU != nil {
		return nil, fmt.Errorf("Cannot get URL: %s", errU)
	}
	r.URL = url

	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	return dump.ToMap(r, dump.WithDefaultLowerCaseFormatter())
}

func failWithScreenshot(contextVars map[string]interface{}, page *agouti.Page, parentError error) error {
	if b, ok := contextVars[webctx.ContextScreenshotOnFailure]; ok {
		if b.(bool) {
			if errS := page.Screenshot("failure.png"); errS != nil {
				log.Warn("Screeshot on failure failed: %s", errS)
			}
		}

	}
	return parentError
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
		return nil, fmt.Errorf("Find %s elements", nbElement)
	}
	return s, nil
}
