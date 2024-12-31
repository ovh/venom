package reporting

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
)

//go:embed venom_openapi_output.html
var templateHTML string

type Data struct {
	FullCoverage      float64
	PartialCoverage   float64
	EmptyCoverage     float64
	NumberOfEndpoints int
}

type GroupSummary struct {
	Name           string
	TotalEndpoints int
	CoveredCount   int
	CoveragePct    float64
	Coverages      []EndpointCoverage
}

func OpenApiOutputHtml(coverageData []EndpointCoverage) ([]byte, error) {
	var buf bytes.Buffer

	// Summarize coverage data for top-level stats
	var countFull, countPartial, countEmpty int
	for _, c := range coverageData {
		fmt.Printf("Coverage: %+v\n", c.CoverageType, c.TotalTests)
		switch c.CoverageType {
		case "full":
			countFull++
		case "partial":
			countPartial++
		case "empty":
			countEmpty++
		}
	}

	// Calculate coverage percentages for display
	totalEndpoints := len(coverageData)
	fullPct := float64(countFull) / float64(totalEndpoints) * 100
	partialPct := float64(countPartial) / float64(totalEndpoints) * 100
	emptyPct := float64(countEmpty) / float64(totalEndpoints) * 100

	grouped := GroupCoverageByTag(coverageData)

	reportData := ReportData{
		FullCoverage:    fullPct,
		PartialCoverage: partialPct,
		EmptyCoverage:   emptyPct,
		TotalEndpoints:  len(coverageData),
		CoverageData:    coverageData,
		GroupedCoverage: grouped,
	}

	tmpl := template.Must(template.New("reportHTML").Parse(templateHTML))

	if err := tmpl.Execute(&buf, reportData); err != nil {
		return nil, errors.Wrap(err, "unable to make template")
	}
	return buf.Bytes(), nil
}
