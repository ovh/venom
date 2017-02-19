package venom

import (
	"strings"
)

// Templater contains templating values on a testsuite
type Templater struct {
	values map[string]string
}

func newTemplater(values map[string]string) *Templater {
	return &Templater{values: values}
}

// Add add data to templater
func (tmpl *Templater) Add(prefix string, values map[string]string) {
	for k, v := range values {
		tmpl.values[prefix+"."+k] = v
	}
}

// Apply apply vars on string
func (tmpl *Templater) Apply(s string) string {
	for k, v := range tmpl.values {
		s = strings.Replace(s, "{{."+k+"}}", v, -1)
	}
	return s
}
