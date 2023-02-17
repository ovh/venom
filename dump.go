package venom

import (
	"os"
	"strings"

	"github.com/fsamin/go-dump"
)

var preserveCase string

func init() {
	preserveCase = os.Getenv("VENOM_PRESERVE_CASE")
	if preserveCase == "" || preserveCase == "AUTO" {
		preserveCase = "ON"
	}
}

// Dump dumps v as a map[string]interface{}.
func DumpWithPrefix(va interface{}, prefix string) (map[string]interface{}, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true
	e.Prefix = prefix

	// TODO venom >= v1.2 update the PreserveCase behaviour
	if preserveCase == "ON" {
		e.ExtraFields.UseJSONTag = true
		e.Formatters = []dump.KeyFormatterFunc{WithFormatterLowerFirstKey()}
	} else {
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	}

	return e.ToMap(va)
}

// Dump dumps v as a map[string]interface{}.
func Dump(va interface{}) (map[string]interface{}, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true

	// TODO venom >= v1.2 update the PreserveCase behaviour
	if preserveCase == "ON" {
		e.ExtraFields.UseJSONTag = true
		e.Formatters = []dump.KeyFormatterFunc{WithFormatterLowerFirstKey()}
	} else {
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	}

	return e.ToMap(va)
}

// DumpString dumps v as a map[string]string{}, key in lowercase
func DumpString(va interface{}) (map[string]string, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true

	// TODO venom >= v1.2 update the PreserveCase behaviour
	if preserveCase == "ON" {
		e.ExtraFields.UseJSONTag = true
		e.Formatters = []dump.KeyFormatterFunc{WithFormatterLowerFirstKey()}
	} else {
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	}
	return e.ToStringMap(va)
}

// DumpStringPreserveCase dumps v as a map[string]string{}
func DumpStringPreserveCase(va interface{}) (map[string]string, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true
	if preserveCase == "ON" {
		e.ExtraFields.UseJSONTag = true
	}
	return e.ToStringMap(va)
}

func WithFormatterLowerFirstKey() dump.KeyFormatterFunc {
	f := dump.WithDefaultFormatter()
	return func(s string, level int) string {
		if level == 0 {
			return strings.ToLower(f(s, level))
		}
		return f(s, level)
	}
}
