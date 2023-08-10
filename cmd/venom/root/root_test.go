package root

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRunCmd(t *testing.T) {
	var validArgs []string

	validArgs = append(validArgs, "run", "../../../tests/assertions")

	rootCmd := New()
	rootCmd.SetArgs(validArgs)
	assert.Equal(t, 3, len(rootCmd.Commands()))
}
