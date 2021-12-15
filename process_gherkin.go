package venom

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/cucumber/gherkin-go/v19"
	"github.com/cucumber/messages-go/v16"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ParseGherkin parses tests suite to check context and variables
func (v *GherkinVenom) ParseGherkin(ctx context.Context, path []string) error {
	filesPath, err := getFilesPath(path, ".feature")
	if err != nil {
		return err
	}

	if err := v.readGherkinFiles(ctx, filesPath); err != nil {
		return err
	}

	return nil
}

func (v *GherkinVenom) readGherkinFiles(ctx context.Context, filesPath []string) (err error) {
	var idx int64
	for _, f := range filesPath {
		log.Info("Reading ", f)
		rawGherkinDoc, err := parseGherkin(f, autoIncrement(&idx))
		if err != nil {
			return err
		}
		feature := v.parseGherkinFeature(ctx, rawGherkinDoc.Feature)
		fmt.Printf("%+v\n", feature)
	}
	return nil
}

func (v *GherkinVenom) parseGherkinFeature(ctx context.Context, feature *messages.Feature) GherkinFeature {
	gf := GherkinFeature{
		Text: feature.Name,
	}
	for _, child := range feature.Children {
		gs := v.parseGherkinFeatureScenario(ctx, child.Scenario)
		gf.Scenarios = append(gf.Scenarios, gs)
	}
	return gf
}

func (v *GherkinVenom) parseGherkinFeatureScenario(ctx context.Context, scenario *messages.Scenario) GherkinScenario {
	gscenario := GherkinScenario{
		Text: scenario.Name,
	}
	for _, step := range scenario.Steps {
		gs := v.parseGherkinFeatureScenarioSteps(ctx, step)
		gscenario.Steps = append(gscenario.Steps, gs)
	}
	return gscenario
}

func (v *GherkinVenom) parseGherkinFeatureScenarioSteps(ctx context.Context, step *messages.Step) GherkinStep {
	return GherkinStep{
		Keywork: step.Keyword,
		Text:    step.Text,
	}
}

func autoIncrement(sequence *int64) func() string {
	return func() string {
		i := atomic.AddInt64(sequence, 1)
		return strconv.FormatInt(i, 10)
	}
}

func parseGherkin(path string, autoIncrementFunc func() string) (*rawGherkinFeature, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	var buf bytes.Buffer
	gherkinDocument, err := gherkin.ParseGherkinDocument(io.TeeReader(reader, &buf), autoIncrementFunc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	gherkinDocument.Uri = path
	pickles := gherkin.Pickles(*gherkinDocument, path, autoIncrementFunc)

	f := rawGherkinFeature{GherkinDocument: gherkinDocument, Pickles: pickles, Content: buf.Bytes()}
	return &f, nil
}
