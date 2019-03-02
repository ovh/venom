package main

import "github.com/ovh/venom/lib/cmd"

var versionCmd = cmd.Cmd{
	Name: "version",
	Desc: "Display current venom binaries version",
}

var versionFunc = func(v cmd.Values) *cmd.Error {
	return nil
}
