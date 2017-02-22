package surf

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/headzoo/surf"
	"github.com/mitchellh/mapstructure"

	"github.com/runabove/venom"
)

// Name of executor
const Name = "surf"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor struct
type Executor struct {
	URL string `json:"url" yaml:"url"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds float64  `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string   `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
}

// Run execute TestStep
func (Executor) Run(l *log.Entry, aliases venom.Aliases, step venom.TestStep) (venom.ExecutorResult, error) {

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}
	r := Result{Executor: t}

	start := time.Now()

	bow := surf.NewBrowser()
	err := bow.Open(t.URL)
	if err != nil {
		return nil, err
	}

	r.Title = bow.Title()
	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	return dump.ToMap(r, dump.WithDefaultLowerCaseFormatter())
}
