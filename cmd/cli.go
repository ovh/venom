package cmd

import (
	"fmt"
	"os"

	"github.com/ovh/venom"
)

// Exit func display an error message on stderr and exit 1
func Exit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	venom.OSExit(1)
}
