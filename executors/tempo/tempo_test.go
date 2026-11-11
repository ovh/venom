package tempo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

const trace = `{"batches":[{"scopeSpans":[{"spans":[{"spanId":"a"},{"spanId":"b"}]}]}]}`

func TestSaveWaitsForIngestion(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/api/traces/0123456789abcdef0123456789abcdef" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(trace))
	}))
	defer srv.Close()
	dir := t.TempDir()
	saved, file, spans := Save(context.Background(), srv.URL, dir, "0123456789abcdef0123456789abcdef", 5*time.Second)
	if saved != "yes" || spans != 2 {
		t.Fatalf("saved %s spans %d", saved, spans)
	}
	if file != filepath.Join(dir, "0123456789abcdef0123456789abcdef.json") {
		t.Fatalf("file %s", file)
	}
	body, err := os.ReadFile(file)
	if err != nil || string(body) != trace {
		t.Fatalf("body %q err %v", body, err)
	}
}

func TestSaveLateWhenNotIngested(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"batches":[]}`))
	}))
	defer srv.Close()
	saved, _, _ := Save(context.Background(), srv.URL, t.TempDir(), "0123456789abcdef0123456789abcdef", 600*time.Millisecond)
	if saved != "late" {
		t.Fatalf("saved %s", saved)
	}
}

func TestNewTraceID(t *testing.T) {
	if got := NewTraceID("0123456789abcdef0123456789abcdef"); got != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("kept id %s", got)
	}
	got := NewTraceID("auto")
	if !hex32.MatchString(got) || got == NewTraceID("auto") {
		t.Fatalf("new id %s", got)
	}
	if len(NewSpanID()) != 16 {
		t.Fatal("span id length")
	}
}
