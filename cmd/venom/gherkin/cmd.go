package gherkin

import (
	"github.com/ovh/venom"
	"github.com/ovh/venom/cmd/venom/run"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "gherkin",
}

func init() {
	Cmd.AddCommand(CmdConvert)
	Cmd.AddCommand(CmdRun)
}

func preRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		path = append(path, ".")
	} else {
		path = args[0:]
	}

	v = venom.NewGherkin()
	run.RegisterExecutorsBuiltin(v.Venom)
}
