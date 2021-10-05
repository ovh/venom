package tavern

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/ovh/venom/assertions"
	diff "github.com/r3labs/diff/v2"
)

// init registers assertion into assertions map
func init() {
	assertions.Set("AssertResponse", AssertResponse)
}

// AssertResponse compares expected and actual responses and raises an error
// if assertion fails
func AssertResponse(actual interface{}, expected ...interface{}) error {
	result, ok := actual.(Result)
	if !ok {
		return fmt.Errorf("bad actual type: expected: Result, actual: %T", actual)
	}
	// check status code
	if result.Expected.StatusCode != 0 {
		if result.Expected.StatusCode != result.Actual.StatusCode {
			return fmt.Errorf("bad status code: expected: %d, actual: %d",
				result.Expected.StatusCode, result.Actual.StatusCode)
		}
	}
	// check expected headers
	for k, v := range result.Expected.Headers {
		value, ok := result.Actual.Headers[k]
		if !ok {
			return fmt.Errorf("header '%s' not found in response", k)
		}
		if value != v {
			return fmt.Errorf("bad header '%s' value: expected: '%s', actual: '%s'",
				k, v, value)
		}
	}
	// check expected body
	if result.Expected.Body != "" {
		if result.Expected.Body != result.Actual.Body {
			return fmt.Errorf("bad body: expected: '%s', actual: '%s'",
				result.Expected.Body, result.Actual.Body)
		}
	}
	// check expected JSON body
	if result.Expected.JSON != nil {
		if !reflect.DeepEqual(result.Expected.JSON, result.Actual.JSON) {
			changelog, err := diff.Diff(result.Expected.JSON, result.Actual.JSON)
			if err != nil {
				return fmt.Errorf("generating JSON diff: %v", strings.TrimSpace(err.Error()))
			}
			if len(result.Expected.JSONExcludes) != 0 {
				var err error
				changelog, err = FilterChangelog(changelog, result.Expected.JSONExcludes)
				if err != nil {
					return err
				}
			}
			if len(changelog) != 0 {
				var diffs []string
				for _, change := range changelog {
					diffs = append(diffs, ChangeMessage(change))
				}
				changes := strings.Join(diffs, "; ")
				return fmt.Errorf("diffs in json: %s", changes)
			}
		}
	}
	return nil
}

// FilterChangelog filters changelog with JSON excluded fields
func FilterChangelog(changelog []diff.Change, filters []string) ([]diff.Change, error) {
	var filteredChangelog []diff.Change
	for _, change := range changelog {
		filtered := false
		path := FormatPath(change.Path)
		for _, filter := range filters {
			match, err := regexp.MatchString(PathToRegexp(filter), path)
			if err != nil {
				return nil, fmt.Errorf("invalid filter regexp: %v", err)
			}
			if match {
				filtered = true
				continue
			}
		}
		if !filtered {
			filteredChangelog = append(filteredChangelog, change)
		}
	}
	return filteredChangelog, nil
}

// ChangeMessage generates human readable change message
func ChangeMessage(change diff.Change) string {
	if change.Type == diff.UPDATE {
		path := FormatPath(change.Path)
		return fmt.Sprintf(`expected:%s = "%v" != actual:%s = "%v"`, path, change.From, path, change.To)
	}
	if change.Type == diff.CREATE {
		path := FormatPath(change.Path)
		return fmt.Sprintf(`actual:%s = "%v" not in expected`, path, change.To)
	}
	if change.Type == diff.DELETE {
		path := FormatPath(change.Path)
		return fmt.Sprintf(`expected:%s = "%v" not in actual`, path, change.From)
	}
	return fmt.Sprintf("UNKNOWN TYPE %s", change.Type)
}

// FormatPath formats JSON path into human readable string
func FormatPath(path []string) string {
	return strings.Join(path, "/")
}

// PathToRegexp builds a regexp from path
func PathToRegexp(path string) string {
	elements := strings.Split(path, "/")
	var parts []string
	for _, element := range elements {
		if element == "*" {
			parts = append(parts, "[^/]*?")
		} else if element == "**" {
			parts = append(parts, ".*?")
		} else {
			parts = append(parts, element)
		}
	}
	return "^" + strings.Join(parts, "/") + "$"
}
