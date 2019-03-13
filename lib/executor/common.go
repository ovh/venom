package executor

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	lsyslog "github.com/sirupsen/logrus/hooks/syslog"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

var log venom.Logger = logrus.New()

type Common interface {
	venom.VenomModule
	venom.Executor
}

type venomExecutorLogger struct {
	*logrus.Logger
	hook *lsyslog.SyslogHook
}

func newLogger(syslogAddress, level string) error {
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
		return err
	}

	l.hook = hook
	l.Logger = logger
	l.Logger.AddHook(l.hook)
	log = &l

	return nil
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

		// TODO Set the context
		res, err := c.Run(nil, step)
		if err != nil {
			Errorf("Error: %v", err)
			return cmd.NewError(502, "executor error: %v", err)
		}
		encoder := yaml.NewEncoder(os.Stdout)
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
		btes, err := yaml.Marshal(c.Manifest())
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
		btes, err := yaml.Marshal(h.GetDefaultAssertions())
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

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}
func Warningf(format string, args ...interface{}) {
	log.Warningf(format, args...)
}
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}
func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}
