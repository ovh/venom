package web

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/sclevine/agouti"

	"github.com/runabove/venom"
	ctxWeb "github.com/runabove/venom/context/web"
)

// Name of executor
const Name = "web"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor struct
type Executor struct {
	URL    string `json:"url" yaml:"url"`
	Action string `json:"action" yaml:"action"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds float64  `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string   `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	Err         error    `json:"error,omitempty" yaml:"error,omitempty"`
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, l *log.Entry, aliases venom.Aliases, step venom.TestStep) (venom.ExecutorResult, error) {
	start := time.Now()

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}
	r := Result{Executor: t}

	varContext := ctx.Value(venom.ContextKey).(map[string]interface{})
	if varContext == nil {
		r.Err = fmt.Errorf("Executor web need a context")
		return endExecutor(r, start)
	}

	if _, ok := varContext[ctxWeb.ContextPageKey]; !ok {
		r.Err = fmt.Errorf("Executor web need a page in context")
		return endExecutor(r, start)
	}

	page := varContext[ctxWeb.ContextPageKey].(*agouti.Page)
	if page == nil {
		r.Err = fmt.Errorf("page is nil in context")
		return endExecutor(r, start)
	}

	switch t.Action {
	case "navigate":
		if err := page.Navigate("http://www.google.fr"); err != nil {
			r.Err = err
		}
	case "title":
		title, err := page.Title()
		if err != nil {
			r.Err = err
		} else {
			r.Title = title
		}
	}
	return endExecutor(r, start)
}

func endExecutor(r Result, start time.Time) (venom.ExecutorResult, error) {
	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	return dump.ToMap(r, dump.WithDefaultLowerCaseFormatter())
}
