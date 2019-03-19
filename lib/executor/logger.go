package executor

import (
	"io/ioutil"
	"log/syslog"

	"github.com/ovh/venom"
	"github.com/sirupsen/logrus"
	lsyslog "github.com/sirupsen/logrus/hooks/syslog"
)

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

var log venom.Logger = logrus.New()

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
