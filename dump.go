package venom

import "github.com/fsamin/go-dump"

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

func DumpPreserveCase(v interface{}) (map[string]interface{}, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true

	return e.ToMap(v)
}

// DumpString dumps v as a map[string]string{}, key in lowercase
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

// DumpStringPreserveCase dumps v as a map[string]string{}
func DumpStringPreserveCase(v interface{}) (map[string]string, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true
	return e.ToStringMap(v)
}
