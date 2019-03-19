package executor

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
	yaml "gopkg.in/yaml.v2"
)

func getExecutorFunc(c Common) func(vals cmd.Values) *cmd.Error {
	return func(vals cmd.Values) *cmd.Error {
		input, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return cmd.NewError(502, "unable to read stdin: %v", err)
		}

		var step venom.TestStep
		if err := yaml.Unmarshal(input, &step); err != nil {
			return cmd.NewError(502, "unable to parse stdin: %v", err)
		}

		loggerAddress := vals.GetString("logger")
		logLevel := vals.GetString("log-level")
		if err := newLogger(loggerAddress, logLevel); err != nil {
			return cmd.NewError(502, "logger error: %v", err)
		}

		t0 := time.Now()
		name := c.Manifest().Name
		Debugf(name + ".Run> Begin")
		defer func() {
			Debugf(name+".Run> End (%.3f seconds)", time.Since(t0).Seconds())
		}()

		res, err := c.Run(NewContextFromEnv(), step)
		if err != nil {
			Errorf("Error: %v", err)
			return cmd.NewError(502, "executor error: %v", err)
		}
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.Encode(res)
		return nil
	}
}
