package venom

import (
	"context"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func Test_OutputResult(t *testing.T) {
	reports, err := gatherPartialReports(context.Background(), filepath.Join("tests", "report"), "json")
	if err != nil {
		t.Errorf("failed")
	} else {
		assert.NotNil(t, reports)
		assert.Equal(t, 3, len(reports))
		for _, report := range reports {
			assert.NotNil(t, report.TestCases)
			assert.NotNil(t, report.Vars)
			assert.NotNil(t, report.ShortName)
			assert.NotNil(t, report.Filename)
			assert.NotNil(t, report.Filepath)
			assert.NotNil(t, report.Status)
			assert.NotNil(t, report.Duration)
			assert.NotNil(t, report.End)
			assert.NotNil(t, report.NbTestcasesPass)
			assert.NotNil(t, report.NbTestcasesFail)
			assert.NotNil(t, report.NbTestcasesSkip)
		}

	}
}
