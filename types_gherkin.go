package venom

import "github.com/cucumber/messages-go/v16"

type rawGherkinFeature struct {
	*messages.GherkinDocument
	Pickles []*messages.Pickle
	Content []byte
}

type GherkinFeature struct {
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
