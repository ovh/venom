package dump

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
)

// KeyFormatterFunc is a type for key formatting
type KeyFormatterFunc func(s string) string

// WithLowerCaseFormatter formats keys in lowercase
func WithLowerCaseFormatter() KeyFormatterFunc {
	return func(s string) string {
		return strings.ToLower(s)
	}
}

// WithDefaultLowerCaseFormatter formats keys in lowercase and apply default formatting
func WithDefaultLowerCaseFormatter() KeyFormatterFunc {
	f := WithDefaultFormatter()
	return func(s string) string {
		return strings.ToLower(f(s))
	}
}

// WithDefaultFormatter is the default formatter
func WithDefaultFormatter() KeyFormatterFunc {
	return func(s string) string {
		s = strings.Replace(s, " ", "_", -1)
		s = strings.Replace(s, "/", "_", -1)
		s = strings.Replace(s, ":", "_", -1)
		return s
	}
}

// NoFormatter doesn't do anything, so to be sure to avoid keys formatting, use only this formatter
func NoFormatter() KeyFormatterFunc {
	return func(s string) string {
		return s
	}
}

// Dump displays the passed parameter properties to standard out such as complete types and all
// pointer addresses used to indirect to the final value.
// See Fdump if you would prefer dumping to an arbitrary io.Writer or Sdump to
// get the formatted result as a string.
func Dump(i interface{}, formatters ...KeyFormatterFunc) error {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	return Fdump(os.Stdout, i, formatters...)
}

// Fdump formats and displays the passed arguments to io.Writer w. It formats exactly the same as Dump.
func Fdump(w io.Writer, i interface{}, formatters ...KeyFormatterFunc) (err error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
		}
	}()
	return fdumpStruct(w, i, nil, formatters...)
}

// Sdump returns a string with the passed arguments formatted exactly the same as Dump.
func Sdump(i interface{}, formatters ...KeyFormatterFunc) (string, error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	var buf bytes.Buffer
	if err := Fdump(&buf, i, formatters...); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fdumpStruct(w io.Writer, i interface{}, roots []string, formatters ...KeyFormatterFunc) error {
	var s reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		s = reflect.ValueOf(i).Elem()
	} else {
		s = reflect.ValueOf(i)
	}

	if !validAndNotEmpty(s) {
		return nil
	}
	switch s.Kind() {
	case reflect.Struct:
		roots = append(roots, s.Type().Name())
		for i := 0; i < s.NumField(); i++ {
			f := s.Field(i)
			if f.Kind() == reflect.Ptr {
				f = f.Elem()
			}
			switch f.Kind() {
			case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
				var data interface{}
				if validAndNotEmpty(f) {
					data = f.Interface()
				} else {
					data = nil
				}
				if err := fdumpStruct(w, data, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
					return err
				}
			default:
				var data interface{}
				if validAndNotEmpty(f) {
					data = f.Interface()
					if f.Kind() == reflect.Interface {
						im, ok := data.(map[string]interface{})
						if ok {
							if err := fDumpMap(w, im, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
								return err
							}
							continue
						}
						am, ok := data.([]interface{})
						if ok {
							if err := fDumpArray(w, am, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
								return err
							}
							continue
						}
					}
					res := fmt.Sprintf("%s.%s: %v\n", strings.Join(sliceFormat(roots, formatters), "."), format(s.Type().Field(i).Name, formatters), data)
					if _, err := w.Write([]byte(res)); err != nil {
						return err
					}
				}
			}
		}
	case reflect.Array, reflect.Slice:
		if err := fDumpArray(w, i, roots, formatters...); err != nil {
			return err
		}
		return nil
	case reflect.Map:
		if err := fDumpMap(w, i, roots, formatters...); err != nil {
			return err
		}
		return nil
	default:
		roots = append(roots, s.Type().Name())
		var data interface{}
		if validAndNotEmpty(s) {
			data = s.Interface()
			if s.Kind() == reflect.Interface {
				im, ok := data.(map[string]interface{})
				if ok {
					return fDumpMap(w, im, roots, formatters...)
				}
				am, ok := data.([]interface{})
				if ok {
					return fDumpArray(w, am, roots, formatters...)
				}
			}
			res := fmt.Sprintf("%s.%s: %v\n", strings.Join(sliceFormat(roots, formatters), "."), format(s.Type().Name(), formatters), data)
			if _, err := w.Write([]byte(res)); err != nil {
				return err
			}
		}
		return nil
	}

	return nil
}

func fDumpArray(w io.Writer, i interface{}, roots []string, formatters ...KeyFormatterFunc) error {
	v := reflect.ValueOf(i)
	for i := 0; i < v.Len(); i++ {
		var l string
		var croots []string
		if len(roots) > 0 {
			l = roots[len(roots)-1:][0]
			croots = roots[:len(roots)-1]
		}
		croots = append(roots, fmt.Sprintf("%s%d", l, i))
		f := v.Index(i)
		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}
		switch f.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			var data interface{}
			if f.IsValid() {
				data = f.Interface()
			} else {
				data = nil
			}
			if err := fdumpStruct(w, data, croots, formatters...); err != nil {
				return err
			}
		default:
			var data interface{}
			if validAndNotEmpty(f) {
				data = f.Interface()
				if f.Kind() == reflect.Interface {
					im, ok := data.(map[string]interface{})
					if ok {
						if err := fDumpMap(w, im, croots, formatters...); err != nil {
							return err
						}
						continue
					}
					am, ok := data.([]interface{})
					if ok {
						if err := fDumpArray(w, am, croots, formatters...); err != nil {
							return err
						}
						continue
					}
				}
				res := fmt.Sprintf("%s: %v\n", strings.Join(sliceFormat(croots, formatters), "."), data)
				if _, err := w.Write([]byte(res)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func fDumpMap(w io.Writer, i interface{}, roots []string, formatters ...KeyFormatterFunc) error {
	v := reflect.ValueOf(i)
	keys := v.MapKeys()
	for _, k := range keys {
		key := fmt.Sprintf("%v", k.Interface())
		roots := append(roots, key)
		value := v.MapIndex(k)
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		switch v.MapIndex(k).Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			if err := fdumpStruct(w, value.Interface(), roots, formatters...); err != nil {
				return err
			}
		default:
			if value.Kind() == reflect.Interface {
				im, ok := value.Interface().(map[string]interface{})
				if ok {
					if err := fDumpMap(w, im, roots, formatters...); err != nil {
						return err
					}
					continue
				}
				am, ok := value.Interface().([]interface{})
				if ok {
					if err := fDumpArray(w, am, roots, formatters...); err != nil {
						return err
					}
					continue
				}
			}
			res := fmt.Sprintf("%s: %v\n", strings.Join(sliceFormat(roots, formatters), "."), value.Interface())
			if _, err := w.Write([]byte(res)); err != nil {
				return err
			}
		}
	}
	return nil
}

type mapWriter struct {
	data map[string]string
}

func (m *mapWriter) Write(p []byte) (int, error) {
	if m.data == nil {
		m.data = map[string]string{}
	}
	tuple := strings.SplitN(string(p), ":", 2)
	if len(tuple) != 2 {
		return 0, errors.New("malformatted bytes")
	}
	tuple[1] = strings.Replace(tuple[1], "\n", "", -1)
	m.data[strings.TrimSpace(tuple[0])] = strings.TrimSpace(tuple[1])
	return len(p), nil
}

// ToMap format passed parameter as a map[string]string. It formats exactly the same as Dump.
func ToMap(i interface{}, formatters ...KeyFormatterFunc) (map[string]string, error) {
	m := mapWriter{}
	err := Fdump(&m, i, formatters...)
	return m.data, err
}

func validAndNotEmpty(v reflect.Value) bool {
	if v.IsValid() {
		if v.Kind() == reflect.String {
			return v.String() != ""
		}
		return true
	}
	return false
}

func sliceFormat(s []string, formatters []KeyFormatterFunc) []string {
	for i := range s {
		s[i] = format(s[i], formatters)
	}
	return s
}

func format(s string, formatters []KeyFormatterFunc) string {
	for _, f := range formatters {
		s = f(s)
	}
	return s
}
