package gherkin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ovh/venom"
	"github.com/spf13/cobra"
)

var CmdConvert = &cobra.Command{
	Use:   "convert",
	Short: "Convert gherkin feature files to venom testsuites",
	Long: `
$ venom gherkin convert *.feature`,
	PreRun: preRun,
	RunE: func(cmd *cobra.Command, args []string) error {
		v.VerboseOutput = "stdout"
		v.Verbose = 2
		v.InitLogger()
		err := v.ParseGherkin(context.Background(), path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}

		var mapDestinations = make(map[string]string)
		for _, ts := range v.Venom.Testsuites {
			newFileName := strings.Replace(ts.Filename, ".feature", ".yml", 1)
			mapDestinations[ts.Filename] = newFileName
		}

		return converFiles(v, mapDestinations)
	},
}

func converFiles(v *venom.GherkinVenom, mapDestinations map[string]string) error {
	for _, ts := range v.Venom.Testsuites {
		newFileName, has := mapDestinations[ts.Filename]
		if !has {
			return fmt.Errorf("missing destination filename for %q", ts.Filename)
		}
		venom.Info(context.Background(), "Writing %s", newFileName)

		btes, _ := yaml.Marshal(ts)
		f, err := os.Create(newFileName)
		if err != nil {
			return err
		}
		_, err = f.Write(btes)
		if err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
