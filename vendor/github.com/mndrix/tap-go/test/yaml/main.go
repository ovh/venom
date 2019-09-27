package main

import (
	"bytes"
	"io"
	"os"

	tap "github.com/mndrix/tap-go"
)

func main() {
	// collect output for comparison later
	buf := new(bytes.Buffer)
	t := tap.New()
	t.Writer = io.MultiWriter(os.Stdout, buf)

	t.Header(2)
	t.Pass("test for anchoring the YAML block")
	message := map[string]interface{}{
		"message": "testing YAML blocks",
		"code":    3,
	}
	t.YAML(message)
	got := buf.String()
	t.Ok(got == expected, "diagnostics gave expected output")
}
