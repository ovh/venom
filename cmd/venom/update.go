package main

import "github.com/ovh/venom/lib/cmd"

var updateCmd = cmd.Cmd{
	Name: "update",
	Desc: "Update venom official binaries",
}

var updateFunc = func(v cmd.Values) *cmd.Error {
	return nil
}
