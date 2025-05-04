package root

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/venom"
)

// getTopLevelFolder returns the top level folder of the project
func getTopLevelFolder() string {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(out))
}

// TestRunCmd tests the run command
func TestRunCmd(t *testing.T) {
	var validArgs []string

	validArgs = append(validArgs, "run", filepath.Join(getTopLevelFolder(), "tests", "assertions"))
	rootCmd := New()
	rootCmd.SetArgs(validArgs)
	venom.IsTest = "test"
	assert.Equal(t, 3, len(rootCmd.Commands()))
	err := rootCmd.Execute()
	assert.NoError(t, err)
	rootCmd.Execute()
}
