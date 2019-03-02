package venom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Process(t *testing.T) {
	v := New()
	r, err := v.Process([]string{"tests/*.yml"})
	assert.NoError(t, err)
	assert.True(t, len(r.TestSuites) == len(v.testsuites), "not the right number of testsuites", len(r.TestSuites), len(v.testsuites))
	assert.True(t, r.Total >= len(v.testsuites), "total seems wrong", r.Total, len(v.testsuites))
}
