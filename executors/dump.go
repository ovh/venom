package executors

import (
	dump "github.com/fsamin/go-dump"
)

// Dump dumps v as a map[string]interface{}.
func Dump(v interface{}) (map[string]interface{}, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}

	return e.ToMap(v)
}

func DumpString(v interface{}) (map[string]string, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	return e.ToStringMap(v)
}
