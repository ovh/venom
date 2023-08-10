package version

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	var validArgs []string

	validArgs = append(validArgs, "version")

	rootCmd := &cobra.Command{
		Use:   "venom",
		Short: "Venom aim to create, manage and run your integration tests with efficiency",
	}
	rootCmd.SetArgs(validArgs)
	rootCmd.AddCommand(Cmd)
	err := rootCmd.Execute()
	assert.NoError(t, err)
}
