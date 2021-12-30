package venom

import (
	"strings"

	"github.com/cucumber/messages-go/v16"
)

type rawGherkinFeature struct {
	*messages.GherkinDocument
	Pickles []*messages.Pickle
	Content []byte
}

type GherkinFeature struct {
	Filename  string
	Text      string
	Scenarios []GherkinScenario
}

type GherkinScenario struct {
	Text  string
	Steps []GherkinStep
}

type GherkinStep struct {
	Keywork string
	Text    string
}

func (feature GherkinFeature) String() string {
	s := "# " + feature.Filename + "\nFeature: " + feature.Text + "\n"

	var stringScenarios []string
	for _, scenario := range feature.Scenarios {
		stringScenarios = append(stringScenarios, scenario.String())
	}

	s += strings.Join(stringScenarios, "\n")
	return s
}

func (scenario GherkinScenario) String() string {
	s := "Scenario: " + scenario.Text + "\n"
	var stringSteps []string
	for _, step := range scenario.Steps {
		stringSteps = append(stringSteps, "\t"+step.String())
	}

	s += strings.Join(stringSteps, "\n")
	return s
}

func (step GherkinStep) String() string {
	return step.Keywork + " " + step.Text
}
