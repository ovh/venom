package main

import (
	"fmt"
	"os"

	"github.com/ovh/venom"
	"github.com/ovh/venom/cmd/venom/root"
)

func main() {
	if err := root.New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		venom.OSExit(2)
	}
}
