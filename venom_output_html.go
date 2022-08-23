package venom

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/pkg/errors"
)

//go:embed venom_output.html
var templateHTML string

type TestsHTML struct {
	Tests     Tests  `json:"tests"`
	JSONValue string `json:"jsonValue"`
}

func outputHTML(testsResult *Tests) ([]byte, error) {
	var buf bytes.Buffer

	testJSON, err := json.MarshalIndent(testsResult, "", " ")
	if err != nil {
		return nil, errors.Wrap(err, "unable to make json value")
	}

	testsHTML := TestsHTML{
		Tests:     *testsResult,
		JSONValue: string(testJSON),
	}
	tmpl := template.Must(template.New("reportHTML").Parse(templateHTML))
	if err := tmpl.Execute(&buf, testsHTML); err != nil {
		return nil, errors.Wrap(err, "unable to make template")
	}
	return buf.Bytes(), nil
}
