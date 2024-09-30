package root

import (
	"github.com/spf13/cobra"

	"github.com/ovh/venom/cmd/venom/run"
	"github.com/ovh/venom/cmd/venom/update"
	"github.com/ovh/venom/cmd/venom/version"
)

// New creates a venom root command.
func New() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "venom",
		Short: "Venom aims to create, manage and run your integration tests with efficiency -- intercom test",
	}
	addCommands(rootCmd)
	return rootCmd
}

// AddCommands adds child commands to the root command rootCmd.
func addCommands(cmd *cobra.Command) {
	cmd.AddCommand(run.Cmd)
	cmd.AddCommand(version.Cmd)
	cmd.AddCommand(update.Cmd)
}
