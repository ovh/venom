package main

import (
	"fmt"
	"os"

	"github.com/ovh/venom/lib/cmd"
)

const (
	_ConfigurationDir = "configuration-dir"
)

func main() {
	var main = cmd.Cmd{
		Name: "venom",
	}

	runCmd.Flags = append(runCmd.Flags, commonFlags...)
	moduleInstallCmd.Flags = append(moduleInstallCmd.Flags, commonFlags...)
	moduleUpdateCmd.Flags = append(moduleUpdateCmd.Flags, commonFlags...)
	moduleListCmd.Flags = append(moduleListCmd.Flags, commonFlags...)

	runCmd.Run = runFunc
	updateCmd.Run = updateFunc
	versionCmd.Run = versionFunc

	modulesCommand := cmd.NewCommand(moduleCmd)
	modulesCommand.AddCommand(
		cmd.NewCommand(moduleInstallCmd),
		cmd.NewCommand(moduleUpdateCmd),
		cmd.NewCommand(moduleListCmd),
	)

	mainCmd := cmd.NewCommand(main)
	mainCmd.AddCommand(
		cmd.NewCommand(runCmd),
		cmd.NewCommand(updateCmd),
		cmd.NewCommand(versionCmd),
		modulesCommand,
	)

	if err := mainCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(128)
	}
}
