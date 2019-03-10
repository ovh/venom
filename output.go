package venom

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ovh/venom/lib/cmd"
)

var _ io.WriteCloser = new(Output)

type Output struct {
	target io.Writer
	buffer bytes.Buffer
}

func NewOutput(target io.Writer) *Output {
	return &Output{target: target}
}

func (w *Output) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

func (w *Output) Close() error {
	_, err := io.Copy(w.target, &w.buffer)
	return err
}

type Progress struct {
	testsuite      string
	testcase       string
	teststepNumber int
	teststepTotal  int
	success        bool
	runnnig        bool
}

var (
	colorTestsuite = getCompiledColor("black+h", "white")
	colorPending   = getCompiledColor("cyan", "white")
	colorSuccess   = getCompiledColor("green", "white")
	colorFailure   = getCompiledColor("red", "white")
	colorExecutor  = getCompiledColor("black+h", "white")
)

func (p *Progress) Display(ctx context.Context, container *cmd.Container) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var display = new(cmd.Display)
	container.Append(display)
	var refresh = time.NewTicker(10 * time.Millisecond)

	for range refresh.C {
		var symbolStatus = "~"
		var colorFunc = colorPending
		if !p.runnnig {
			if p.success {
				symbolStatus = "✓"
				colorFunc = colorSuccess
			} else {
				symbolStatus = "✗"
				colorFunc = colorFailure
			}
		}
		display.Printf(" %s %s %s - steps: %s", colorFunc(symbolStatus), colorTestsuite(p.testsuite), colorFunc(p.testcase), colorFunc(fmt.Sprintf("%d/%d", p.teststepNumber, p.teststepTotal)))
		if ctx.Err() != nil {
			return
		}
	}
}
