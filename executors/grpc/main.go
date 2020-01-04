package main

import (
	"github.com/ovh/venom/lib/cmd"
	"github.com/ovh/venom/lib/executor"
)

func main() {
	var e Executor
	if err := executor.Start(e); err != nil {
		cmd.ExitOnError(err)
	}
}
