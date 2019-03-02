package executor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"

	"github.com/sirupsen/logrus"

	lsyslog "github.com/sirupsen/logrus/hooks/syslog"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

type Common interface {
	venom.VenomModule
	venom.Executor
}

type venomExecutorLogger struct {
	*logrus.Logger
	hook *lsyslog.SyslogHook
}

func NewLogger(syslogAddress, level string) (venom.Logger, error) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	logger.SetFormatter(new(logrus.TextFormatter))

	switch level {
	case venom.LogLevelDebug:
		logger.SetLevel(logrus.DebugLevel)
	case venom.LogLevelError:
		logger.SetLevel(logrus.ErrorLevel)
	case venom.LogLevelWarn:
		logger.SetLevel(logrus.WarnLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	var l venomExecutorLogger
	hook, err := lsyslog.NewSyslogHook("tcp", syslogAddress, syslog.LOG_INFO|syslog.LOG_USER, "")
	if err != nil {
		return nil, err
	}

	l.hook = hook
	l.Logger = logger
	l.Logger.AddHook(l.hook)

	return &l, nil
}

func Start(c Common) error {
	var main = cmd.Cmd{
		Name: c.Manifest().Name,
	}

	mainCmd := cmd.NewCommand(main)

	var execute = cmd.Cmd{
		Name: "execute",
		Flags: []cmd.Flag{
			{
				Name: "logger",
			},
			{
				Name: "log-level",
			},
		},
	}
	execute.Run = func(vals cmd.Values) *cmd.Error {
		input, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return cmd.NewError(502, "unable to read stdin: %v", err)
		}

		var step venom.TestStep
		if err := json.Unmarshal(input, &step); err != nil {
			return cmd.NewError(502, "unable to parse stdin: %v", err)
		}

		loggerAddress := vals.GetString("logger")
		logLevel := vals.GetString("log-level")
		logger, err := NewLogger(loggerAddress, logLevel)
		if err != nil {
			return cmd.NewError(502, "logger error: %v", err)
		}

		res, err := c.Run(nil, logger, step)
		if err != nil {
			return cmd.NewError(502, "executor error: %v", err)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(res)
		return nil
	}
	executeCmd := cmd.NewCommand(execute)

	var parse = cmd.Cmd{
		Name: "parse",
		Flags: []cmd.Flag{
			{
				Name: "logger",
			},
			{
				Name: "log-level",
			},
		},
	}
	parse.Run = func(vals cmd.Values) *cmd.Error {
		return nil
	}
	parseCmd := cmd.NewCommand(parse)

	var info = cmd.Cmd{
		Name: "info",
	}
	info.Run = func(vals cmd.Values) *cmd.Error {
		btes, err := json.MarshalIndent(c.Manifest(), "", "  ")
		if err != nil {
			return cmd.NewError(501, "unable to format manifest output: %v", err)
		}
		fmt.Println(string(btes))
		return nil
	}
	infoCmd := cmd.NewCommand(info)

	var assertions = cmd.Cmd{
		Name: "assertions",
	}
	assertions.Run = func(vals cmd.Values) *cmd.Error {
		h, ok := c.(venom.ExecutorWithDefaultAssertions)
		if !ok {
			return nil
		}
		btes, err := json.MarshalIndent(h.GetDefaultAssertions(), "", "  ")
		if err != nil {
			return cmd.NewError(501, "unable to format assertions output: %v", err)
		}
		fmt.Println(string(btes))
		return nil
	}
	assertionsCmd := cmd.NewCommand(assertions)

	mainCmd.AddCommand(executeCmd, parseCmd, infoCmd, assertionsCmd)

	return mainCmd.Execute()
}
