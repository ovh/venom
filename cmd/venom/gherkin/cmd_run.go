package gherkin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/venom"
	"github.com/ovh/venom/cmd/venom/run"
)

var (
	v    *venom.GherkinVenom
	path []string
)

func init() {
	run.InitCmdFlags(CmdRun)
}

var CmdRun = &cobra.Command{
	Use:   "run",
	Short: "Run Gherkin tests",
	Long: `
$ venom gherkin run *.feature`,
	PreRun: preRun,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := run.InitCmdWithVenom(&v.Venom, cmd, nil); err != nil {
			return err
		}

		venom.Info(context.Background(), "Running venom in gherkin mode (beta)")

		// path is set by preRun
		if err := v.ParseGherkin(context.Background(), path); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return err
		}

		var mapDestinations = make(map[string]string)
		var generatedFiles []string
		for _, ts := range v.Venom.Testsuites {
			base := filepath.Base(ts.Filename)
			base = strings.Replace(base, ".feature", ".yml", 1)
			tmpdir, err := os.MkdirTemp("", "venom-generated-testsuites-*")
			if err != nil {
				return err
			}
			newFileName := filepath.Join(tmpdir, base)
			generatedFiles = append(generatedFiles, newFileName)
			mapDestinations[ts.Filename] = newFileName
		}

		if err := converFiles(v, mapDestinations); err != nil {
			return err
		}

		if err := run.RunCmdWithVenom(&v.Venom, cmd, generatedFiles); err != nil {
			return err
		}

		return nil
	},
}
