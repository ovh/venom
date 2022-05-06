package venom

import "github.com/fsamin/go-dump"

func (v *Venom) dumpEncoder() *dump.Encoder {
	e := dump.NewDefaultEncoder()
	e.ArrayJSONNotation = true
	if v.Options.DisableNewArraySyntax {
		e.ArrayJSONNotation = false
	}
	if v.Options.EnableExtraField {
		e.ExtraFields.Len = true
		e.ExtraFields.Type = true
		e.ExtraFields.DetailedStruct = true
		e.ExtraFields.DetailedMap = true
		e.ExtraFields.DetailedArray = true
	}
	if v.Options.DisablePreserveCaseOnAssertion {
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	}
	return e
}

// Dump dumps v as a map[string]interface{}.
func (v *Venom) Dump(i interface{}) (map[string]interface{}, error) {
	return v.dumpEncoder().ToMap(i)
}

// DumpString dumps v as a map[string]string{}, key in lowercase
func (v *Venom) DumpString(i interface{}) (map[string]string, error) {
	return v.dumpEncoder().ToStringMap(i)
}

// DumpStringPreserveCase dumps v as a map[string]string{}
/*func DumpStringPreserveCase(v interface{}) (map[string]string, error) {
	e := dump.NewDefaultEncoder()
	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.ExtraFields.DetailedArray = true
	return e.ToStringMap(v)
}*/
