package main

import (
	"fmt"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

var versionCmd = cmd.Cmd{
	Name: "version",
	Desc: "Display current venom binary version",
}

var versionFunc = func(v cmd.Values) *cmd.Error {
	fmt.Println("version:", venom.Version)
	fmt.Println("os:", venom.GOOS)
	fmt.Println("arch:", venom.GOARCH)
	fmt.Println("build time:", venom.BUILDTIME)
	fmt.Println("git hash:", venom.GITHASH)
	return nil
}
