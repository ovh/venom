package venom

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"

	"github.com/pkg/errors"
)

//go:embed venom_output.html
var templateHTML1 string

//go:embed venom_output2.html
var templateHTML2 string

type TestsHTML struct {
	Tests     Tests  `json:"tests"`
	JSONValue string `json:"jsonValue"`
}

func outputHTML(testsResult *Tests, version int) ([]byte, error) {
	var buf bytes.Buffer
	var html string

	switch version {
	case 2:
		html = templateHTML2
	default:
		html = templateHTML1
	}

	testJSON, err := json.MarshalIndent(testsResult, "", " ")
	if err != nil {
		return nil, errors.Wrap(err, "unable to make json value")
	}

	testsHTML := TestsHTML{
		Tests:     *testsResult,
		JSONValue: string(testJSON),
	}
	tmpl := template.Must(template.New("reportHTML").Parse(html))
	if err := tmpl.Execute(&buf, testsHTML); err != nil {
		return nil, errors.Wrap(err, "unable to make template")
	}
	return buf.Bytes(), nil
}
