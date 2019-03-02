package main

import (
	"fmt"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

var moduleCmd = cmd.Cmd{
	Name: "modules",
	Desc: "Venom modules management",
}

var moduleListCmd = cmd.Cmd{
	Name: "list",
	Desc: "List installed venom modules",
	Run: func(vals cmd.Values) *cmd.Error {
		var v = venom.New()
		if err := checkConfigurationDirectory(v, vals); err != nil {
			return err
		}

		mods, err := v.ListModules()
		if err != nil {
			return cmd.NewError(1, "venom intialization error: unable to list installed modules: %v", err)
		}

		if len(mods) == 0 {
			fmt.Println("no module installed...")
			return nil
		}

		keys := []string{"Name", "Version", "Type", "Description", "Author"}
		data := [][]string{}
		for _, m := range mods {
			manifest := m.Manifest()
			data = append(data, []string{
				manifest.Name,
				manifest.Version,
				manifest.Type,
				manifest.Description,
				manifest.Author,
			})
		}

		cmd.DisplayTable(keys, data)

		return nil
	},
}

var moduleInstallCmd = cmd.Cmd{
	Name: "install",
	Desc: "Install venom modules",
}

var moduleUpdateCmd = cmd.Cmd{
	Name: "update",
	Desc: "Update venom modules",
}
