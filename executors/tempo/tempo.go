// Package tempo saves a trace from a Grafana Tempo HTTP API to a file, and exposes that as the `tempo` executor.
//
// A trace is saved when GET <tempo_url>/api/traces/<trace_id> answers 200 with at least one batch (or resourceSpans).
// Save polls every 500 ms up to `wait`; when the trace is not ingested yet, a late fetch keeps polling in the background
// (every 2 s up to LateWait) while the venom process lives, and the result says "late".
package tempo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/venom"
)

// Name of the executor.
const Name = "tempo"

// LateWait bounds the background fetch of a trace not ingested within the step wait.
var LateWait = 90 * time.Second

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

// New returns a new executor.
func New() venom.Executor {
	return &Executor{}
}

// Executor saves one trace: trace_id, tempo_url, trace_dir, wait (seconds, default 20).
type Executor struct {
	TraceID  string `json:"trace_id" yaml:"trace_id" mapstructure:"trace_id"`
	TempoURL string `json:"tempo_url" yaml:"tempo_url" mapstructure:"tempo_url"`
	TraceDir string `json:"trace_dir" yaml:"trace_dir" mapstructure:"trace_dir"`
	Wait     int    `json:"wait" yaml:"wait" mapstructure:"wait"`
}

// Result of the executor.
type Result struct {
	TraceID    string `json:"trace_id" yaml:"trace_id"`
	TraceSaved string `json:"trace_saved" yaml:"trace_saved"`
	TraceFile  string `json:"trace_file" yaml:"trace_file"`
	Spans      int    `json:"spans" yaml:"spans"`
}

// ZeroValueResult returns an empty result.
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions: the trace is saved.
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []venom.Assertion{"result.trace_saved ShouldEqual yes"}}
}

// Run saves the trace.
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.WeakDecode(step, &e); err != nil {
		return nil, err
	}
	if !hex32.MatchString(e.TraceID) {
		return nil, fmt.Errorf("tempo: trace_id %q is not 32 lower-case hex", e.TraceID)
	}
	wait := time.Duration(e.Wait) * time.Second
	if e.Wait <= 0 {
		wait = 20 * time.Second
	}
	saved, file, spans := Save(ctx, e.TempoURL, e.TraceDir, e.TraceID, wait)
	return Result{TraceID: e.TraceID, TraceSaved: saved, TraceFile: file, Spans: spans}, nil
}

// NewTraceID returns 32 hex chars of a W3C trace id; a valid given id is kept.
func NewTraceID(given string) string {
	if hex32.MatchString(given) {
		return given
	}
	return randomHex(16)
}

// NewSpanID returns 16 hex chars of a W3C parent span id.
func NewSpanID() string {
	return randomHex(8)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Save fetches the trace into <dir>/<id>.json. It returns "yes" when saved within wait, "late" when a background fetch
// goes on, "no" when there is nothing to save to; the file path; the span count of a saved trace.
func Save(ctx context.Context, tempoURL, dir, id string, wait time.Duration) (string, string, int) {
	if tempoURL == "" || dir == "" {
		return "no", "", 0
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "no", "", 0
	}
	file := filepath.Join(dir, id+".json")
	deadline := time.Now().Add(wait)
	for {
		if body, ok := fetch(ctx, tempoURL, id); ok {
			if err := write(file, body); err == nil {
				return "yes", file, spanCount(body)
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	go func() {
		end := time.Now().Add(LateWait)
		for time.Now().Before(end) {
			time.Sleep(2 * time.Second)
			if body, ok := fetch(context.Background(), tempoURL, id); ok {
				_ = write(file, body)
				return
			}
		}
	}()
	return "late", file, 0
}

func fetch(ctx context.Context, tempoURL, id string) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tempoURL+"/api/traces/"+id, nil)
	if err != nil {
		return nil, false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	return body, spanCount(body) > 0 || batches(body) > 0
}

type traceDoc struct {
	Batches       []batch `json:"batches"`
	ResourceSpans []batch `json:"resourceSpans"`
}

type batch struct {
	ScopeSpans []struct {
		Spans []json.RawMessage `json:"spans"`
	} `json:"scopeSpans"`
	InstrumentationLibrarySpans []struct {
		Spans []json.RawMessage `json:"spans"`
	} `json:"instrumentationLibrarySpans"`
}

func parse(body []byte) traceDoc {
	var d traceDoc
	_ = json.Unmarshal(body, &d)
	return d
}

func batches(body []byte) int {
	d := parse(body)
	return len(d.Batches) + len(d.ResourceSpans)
}

func spanCount(body []byte) int {
	d := parse(body)
	n := 0
	for _, b := range append(d.Batches, d.ResourceSpans...) {
		for _, s := range b.ScopeSpans {
			n += len(s.Spans)
		}
		for _, s := range b.InstrumentationLibrarySpans {
			n += len(s.Spans)
		}
	}
	return n
}

func write(file string, body []byte) error {
	tmp := file + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}
