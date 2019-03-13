package venom

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)

//Logger is basically an interface for logrus.Entry
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

func LoggerWithField(l Logger, key string, i interface{}) Logger {
	logrusLogger, ok := l.(*logrus.Logger)
	if ok {
		return logrusLogger.WithField(key, i)
	}

	logrusEntry, ok := l.(*logrus.Entry)
	if ok {
		return logrusEntry.WithField(key, i)
	}
	return &LoggerWithPrefix{
		parent:      l,
		prefixKey:   key,
		prefixValue: i,
	}
}

type LoggerWithPrefix struct {
	parent      Logger
	prefixKey   string
	prefixValue interface{}
}

func (l LoggerWithPrefix) Debugf(format string, args ...interface{}) {
	s := fmt.Sprintf("%s=%v", l.prefixKey, l.prefixValue)
	l.parent.Debugf(s+"\t"+format, args...)
}
func (l LoggerWithPrefix) Infof(format string, args ...interface{}) {
	s := fmt.Sprintf("%s=%v", l.prefixKey, l.prefixValue)
	l.parent.Infof(s+"\t"+format, args...)
}
func (l LoggerWithPrefix) Warnf(format string, args ...interface{}) {
	s := fmt.Sprintf("%s=%v", l.prefixKey, l.prefixValue)
	l.parent.Warnf(s+"\t"+format, args...)
}
func (l LoggerWithPrefix) Warningf(format string, args ...interface{}) {
	s := fmt.Sprintf("%s=%v", l.prefixKey, l.prefixValue)
	l.parent.Warningf(s+"\t"+format, args...)
}
func (l LoggerWithPrefix) Errorf(format string, args ...interface{}) {
	s := fmt.Sprintf("%s=%v", l.prefixKey, l.prefixValue)
	l.parent.Errorf(s+"\t"+format, args...)
}
func (l LoggerWithPrefix) Fatalf(format string, args ...interface{}) {
	s := fmt.Sprintf("%s=%v", l.prefixKey, l.prefixValue)
	l.parent.Fatalf(s+"\t"+format, args...)
}

type LogFormatter struct {
	// Can be set to the override the default quoting character "
	// with something else. For example: ', or `.
	QuoteCharacter string

	// Color scheme to use.
	colorScheme *compiledColorScheme

	// Whether the logger's out is to a terminal.
	isTerminal bool

	sync.Once
}

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	//lastKeyIdx := len(keys) - 1

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	prefixFieldClashes(entry.Data)
	timestampFormat := defaultTimestampFormat

	f.Do(func() { f.init(entry) })

	if f.isTerminal {
		var colorScheme *compiledColorScheme
		if f.colorScheme == nil {
			colorScheme = defaultCompiledColorScheme
		}
		f.printColored(b, entry, keys, timestampFormat, colorScheme)
	} else {
		//		if !f.DisableTimestamp {
		//			f.appendKeyValue(b, "time", entry.Time.Format(timestampFormat), true)
		//		}
		//		f.appendKeyValue(b, "level", entry.Level.String(), true)
		//		if entry.Message != "" {
		//			f.appendKeyValue(b, "msg", entry.Message, lastKeyIdx >= 0)
		//		}
		//		for i, key := range keys {
		//			f.appendKeyValue(b, key, entry.Data[key], lastKeyIdx != i)
		//		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil

}

const defaultTimestampFormat = time.RFC3339

var (
	baseTimestamp      time.Time    = time.Now()
	defaultColorScheme *ColorScheme = &ColorScheme{
		InfoLevelStyle:  "green",
		WarnLevelStyle:  "yellow",
		ErrorLevelStyle: "red",
		FatalLevelStyle: "red",
		PanicLevelStyle: "red",
		DebugLevelStyle: "blue",
		PrefixStyle:     "cyan",
		TimestampStyle:  "black+h",
	}
	noColorsColorScheme *compiledColorScheme = &compiledColorScheme{
		InfoLevelColor:  ansi.ColorFunc(""),
		WarnLevelColor:  ansi.ColorFunc(""),
		ErrorLevelColor: ansi.ColorFunc(""),
		FatalLevelColor: ansi.ColorFunc(""),
		PanicLevelColor: ansi.ColorFunc(""),
		DebugLevelColor: ansi.ColorFunc(""),
		PrefixColor:     ansi.ColorFunc(""),
		TimestampColor:  ansi.ColorFunc(""),
	}
	defaultCompiledColorScheme *compiledColorScheme = compileColorScheme(defaultColorScheme)
)

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

type ColorScheme struct {
	InfoLevelStyle  string
	WarnLevelStyle  string
	ErrorLevelStyle string
	FatalLevelStyle string
	PanicLevelStyle string
	DebugLevelStyle string
	PrefixStyle     string
	PrefixStyle2    string
	TimestampStyle  string
}

type compiledColorScheme struct {
	InfoLevelColor  func(string) string
	WarnLevelColor  func(string) string
	ErrorLevelColor func(string) string
	FatalLevelColor func(string) string
	PanicLevelColor func(string) string
	DebugLevelColor func(string) string
	PrefixColor     func(string) string
	TimestampColor  func(string) string
}

func getCompiledColor(main string, fallback string) func(string) string {
	var style string
	if main != "" {
		style = main
	} else {
		style = fallback
	}
	return ansi.ColorFunc(style)
}

func compileColorScheme(s *ColorScheme) *compiledColorScheme {
	var c = compiledColorScheme{
		InfoLevelColor:  getCompiledColor(s.InfoLevelStyle, defaultColorScheme.InfoLevelStyle),
		WarnLevelColor:  getCompiledColor(s.WarnLevelStyle, defaultColorScheme.WarnLevelStyle),
		ErrorLevelColor: getCompiledColor(s.ErrorLevelStyle, defaultColorScheme.ErrorLevelStyle),
		FatalLevelColor: getCompiledColor(s.FatalLevelStyle, defaultColorScheme.FatalLevelStyle),
		PanicLevelColor: getCompiledColor(s.PanicLevelStyle, defaultColorScheme.PanicLevelStyle),
		DebugLevelColor: getCompiledColor(s.DebugLevelStyle, defaultColorScheme.DebugLevelStyle),
		PrefixColor:     getCompiledColor(s.PrefixStyle, defaultColorScheme.PrefixStyle),
		TimestampColor:  getCompiledColor(s.TimestampStyle, defaultColorScheme.TimestampStyle),
	}
	return &c
}

func (f *LogFormatter) init(entry *logrus.Entry) {
	if len(f.QuoteCharacter) == 0 {
		f.QuoteCharacter = "\""
	}
	if entry.Logger != nil {
		f.isTerminal = f.checkIfTerminal(entry.Logger.Out)
	}
}

func (f *LogFormatter) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (f *LogFormatter) SetColorScheme(colorScheme *ColorScheme) {
	f.colorScheme = compileColorScheme(colorScheme)
}

func (f *LogFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string, colorScheme *compiledColorScheme) {
	var levelColor func(string) string
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = colorScheme.InfoLevelColor
	case logrus.WarnLevel:
		levelColor = colorScheme.WarnLevelColor
	case logrus.ErrorLevel:
		levelColor = colorScheme.ErrorLevelColor
	case logrus.FatalLevel:
		levelColor = colorScheme.FatalLevelColor
	case logrus.PanicLevel:
		levelColor = colorScheme.PanicLevelColor
	default:
		levelColor = colorScheme.DebugLevelColor
	}

	if entry.Level != logrus.WarnLevel {
		levelText = entry.Level.String()
	} else {
		levelText = "warn"
	}

	levelText = strings.ToUpper(levelText)

	level := levelColor(fmt.Sprintf("%5s ", levelText))
	message := entry.Message

	prefix := ""
	var prefixKeys = []string{"testsuite", "testcase", "step", "executor"}
	for _, k := range prefixKeys {
		v, has := entry.Data[k]
		if has {
			vs := emphasis(v.(string), 9)
			vs = vs + "/"
			prefix += vs
		}
	}
	timestamp := fmt.Sprintf("[%04d]", miniTS())
	prefix = colorScheme.PrefixColor(fmt.Sprintf("%-40s", prefix))
	fmt.Fprintf(b, "%s %s%s %s", colorScheme.TimestampColor(timestamp), level, prefix, message)

	for _, k := range keys {
		var isPrefix bool
		for _, kP := range prefixKeys {
			if k == kP {
				isPrefix = true
				break
			}
		}
		if !isPrefix {
			v := entry.Data[k]
			fmt.Fprintf(b, " %s=%+v", levelColor(k), v)
		}
	}
}

func prefixFieldClashes(data logrus.Fields) {
	if t, ok := data["time"]; ok {
		data["fields.time"] = t
	}

	if m, ok := data["msg"]; ok {
		data["fields.msg"] = m
	}

	if l, ok := data["level"]; ok {
		data["fields.level"] = l
	}
}

func emphasis(s string, i int) string {
	if len(s) > i {
		return s[:i-1] + "…"
	}
	return s
}
