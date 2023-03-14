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
	if err := CheckAssertions(result.Expected); err != nil {
		return err
	}
	// check status code
	if result.Expected.StatusCode != 0 {
		if result.Expected.StatusCode != result.Actual.StatusCode {
			return fmt.Errorf("bad status code: expected: %d, actual: %d",
				result.Expected.StatusCode, result.Actual.StatusCode)
		}
	}
	// check expected headers
	for key, expected := range result.Expected.Headers {
		actual, ok := result.Actual.Headers[key]
		if !ok {
			return fmt.Errorf("header '%s' not found in response", key)
		}
		if ElementInList(key, result.Expected.HeadersRegexps) {
			match, err := regexp.MatchString(expected, actual)
			if err != nil {
				return fmt.Errorf("bad headers regexp: %v", err)
			}
			if !match {
				return fmt.Errorf("bad header '%s' value: regexp: '%s', doesn't match: '%s'",
					key, expected, actual)
			}
		} else {
			if actual != expected {
				return fmt.Errorf("bad header '%s' value: expected: '%s', actual: '%s'",
					key, expected, actual)
			}
		}
	}
	// check expected body
	if result.Expected.Body != "" {
		if result.Expected.Body != result.Actual.Body {
			return fmt.Errorf("bad body: expected: '%s', actual: '%s'",
				result.Expected.Body, result.Actual.Body)
		}
	}
	// check expected body
	if result.Expected.BodyRegexp != "" {
		match, err := regexp.MatchString(result.Expected.BodyRegexp, result.Actual.Body)
		if err != nil {
			return fmt.Errorf("bad body regexp: %v", err)
		}
		if !match {
			return fmt.Errorf("body doesn't match regexp '%s'", result.Expected.BodyRegexp)
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
				changelog, err = FilterChangelogExcludes(changelog, result.Expected.JSONExcludes)
				if err != nil {
					return err
				}
			}
			if len(result.Expected.JSONRegexps) != 0 {
				var err error
				changelog, err = FilterChangelogRegexps(changelog, result.Expected.JSONRegexps)
				if err != nil {
					return err
				}
			}
			if len(changelog) != 0 {
				var diffs []string
				for _, change := range changelog {
					diffs = append(diffs, ChangeMessage(change, result.Expected.JSONRegexps))
				}
				changes := strings.Join(diffs, "; ")
				return fmt.Errorf("diffs in json: %s", changes)
			}
		}
	}
	return nil
}

// FilterChangelogExcludes filters changelog with JSON excluded fields
func FilterChangelogExcludes(changelog []diff.Change, filters []string) ([]diff.Change, error) {
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

// FilterChangelogRegexps filters changelog with Regexp fields
func FilterChangelogRegexps(changelog []diff.Change, filters []string) ([]diff.Change, error) {
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
				if change.Type == diff.UPDATE {
					from, ok := change.From.(string)
					if !ok {
						return nil, fmt.Errorf("regexp field %s is not a string", path)
					}
					to, ok := change.To.(string)
					if !ok {
						return nil, fmt.Errorf("regexp filter %s is not a string", path)
					}
					m, e := regexp.MatchString(from, to)
					if e != nil {
						return nil, fmt.Errorf("invalid field regexp: %v", err)
					}
					if m {
						filtered = true
					}
				}
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
func ChangeMessage(change diff.Change, jsonRegexps []string) string {
	if change.Type == diff.UPDATE {
		path := FormatPath(change.Path)
		if ElementInList(path, jsonRegexps) {
			return fmt.Sprintf(`expected:%s = "%v" !~ actual:%s = "%v"`, path, change.From, path, change.To)
		}
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
			parts = append(parts, "[^/]+/")
		} else if element == "**" {
			parts = append(parts, ".*/?")
		} else {
			parts = append(parts, element+"/")
		}
	}
	return "^" + strings.TrimSuffix(strings.Join(parts, ""), "/") + "$"
}

// ElementInList tells if given path is in filters list
func ElementInList(path string, filters[]string) bool {
	for _, filter := range filters {
		if filter == path {
			return true
		}
	}
	return false
}

// CheckAssertions check incompatibles assertions
func CheckAssertions(expected Response) error {
	if expected.Body != "" && expected.BodyRegexp != "" {
		return fmt.Errorf("you can set both body and bodyRegexps assertions")
	}
	for _, regexp := range expected.HeadersRegexps {
		if _, ok := expected.Headers[regexp]; !ok {
			return fmt.Errorf("field %s declared as regexp but not found in headers list", regexp)
		}
	}
	for _, regexp := range expected.JSONRegexps {
		if ElementInList(regexp, expected.JSONExcludes) {
			return fmt.Errorf("JSON field '%s' can't be excluded and declared as regexp", regexp)
		}
	}
	return nil
}