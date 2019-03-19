package executor

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

type Common interface {
	venom.VenomModule
	venom.Executor
}

func Start(c Common) error {
	var main = cmd.Cmd{
		Name: c.Manifest().Name,
	}

	mainCmd := cmd.NewCommand(main)

	var execute = cmd.Cmd{
		Name: "execute",
		Flags: []cmd.Flag{
			{
				Name: "logger",
			},
			{
				Name: "log-level",
			},
		},
	}
	execute.Run = getExecutorFunc(c)
	executeCmd := cmd.NewCommand(execute)

	var parse = cmd.Cmd{
		Name: "parse",
		Flags: []cmd.Flag{
			{
				Name: "logger",
			},
			{
				Name: "log-level",
			},
		},
	}
	parse.Run = func(vals cmd.Values) *cmd.Error {
		return nil
	}
	parseCmd := cmd.NewCommand(parse)

	var info = cmd.Cmd{
		Name: "info",
	}
	info.Run = func(vals cmd.Values) *cmd.Error {
		btes, err := yaml.Marshal(c.Manifest())
		if err != nil {
			return cmd.NewError(501, "unable to format manifest output: %v", err)
		}
		fmt.Println(string(btes))
		return nil
	}
	infoCmd := cmd.NewCommand(info)

	var assertions = cmd.Cmd{
		Name: "assertions",
	}
	assertions.Run = func(vals cmd.Values) *cmd.Error {
		h, ok := c.(venom.ExecutorWithDefaultAssertions)
		if !ok {
			return nil
		}
		btes, err := yaml.Marshal(h.GetDefaultAssertions())
		if err != nil {
			return cmd.NewError(501, "unable to format assertions output: %v", err)
		}
		fmt.Println(string(btes))
		return nil
	}
	assertionsCmd := cmd.NewCommand(assertions)

	mainCmd.AddCommand(executeCmd, parseCmd, infoCmd, assertionsCmd)

	return mainCmd.Execute()
}
