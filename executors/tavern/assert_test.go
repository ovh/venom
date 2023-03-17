package tavern

import (
	"encoding/json"
	"testing"

	diff "github.com/r3labs/diff/v2"
)

func TestAssertResponseType(t *testing.T) {
	err := AssertResponse("test")
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "bad actual type: expected: Result, actual: string" {
		t.Fatalf("bad error message: %s", err.Error())
	}
}

func TestAssertResponseStatusCode(t *testing.T) {
	result := Result{
		Expected: Response{StatusCode: 200},
		Actual:   Response{StatusCode: 200},
	}
	err := AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	result = Result{
		Expected: Response{StatusCode: 200},
		Actual:   Response{StatusCode: 201},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "bad status code: expected: 200, actual: 201" {
		t.Fatalf("bad error message: %s", err.Error())
	}
}

func TestAssertResponseHeaders(t *testing.T) {
	// nominal case (we accept additional headers in actual)
	result := Result{
		Expected: Response{
			Headers: Headers{"Foo": "Bar"},
		},
		Actual: Response{
			Headers: Headers{
				"Foo":  "Bar",
				"Spam": "Eggs",
			},
		},
	}
	err := AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// missing header
	result = Result{
		Expected: Response{
			Headers: Headers{
				"Foo":  "Bar",
				"Spam": "Eggs",
			},
		},
		Actual: Response{
			Headers: Headers{"Foo": "Bar"},
		},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "header 'Spam' not found in response" {
		t.Fatalf("bad error message: %s", err.Error())
	}
	// bad header
	result = Result{
		Expected: Response{
			Headers: Headers{"Foo": "Bar"},
		},
		Actual: Response{
			Headers: Headers{"Foo": "Baz"},
		},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "bad header 'Foo' value: expected: 'Bar', actual: 'Baz'" {
		t.Fatalf("bad error message: %s", err.Error())
	}
	// nominal case with regexp
	result = Result{
		Expected: Response{
			Headers:        Headers{"Foo": "B.r"},
			HeadersRegexps: []string{"Foo"},
		},
		Actual: Response{
			Headers: Headers{"Foo": "Bar"},
		},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// no match with regexp
	result = Result{
		Expected: Response{
			Headers:        Headers{"Foo": "x.*"},
			HeadersRegexps: []string{"Foo"},
		},
		Actual: Response{
			Headers: Headers{"Foo": "Bar"},
		},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should not have succeeded: %v", err)
	}
}

func TestAssertResponseBody(t *testing.T) {
	result := Result{
		Expected: Response{Body: "Foo"},
		Actual:   Response{Body: "Foo"},
	}
	err := AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	result = Result{
		Expected: Response{Body: "Foo"},
		Actual:   Response{Body: "Bar"},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "bad body: expected: 'Foo', actual: 'Bar'" {
		t.Fatalf("bad error message: %s", err.Error())
	}
}

func TestAssertResponseBodyRegexp(t *testing.T) {
	result := Result{
		Expected: Response{BodyRegexp: "F.o"},
		Actual:   Response{Body: "Foo"},
	}
	err := AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded")
	}
	result = Result{
		Expected: Response{BodyRegexp: "x.*"},
		Actual:   Response{Body: "Foo"},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "body doesn't match regexp 'x.*'" {
		t.Fatalf("bad error message: %s", err.Error())
	}
}

func TestAssertResponseJson(t *testing.T) {
	// nominal case
	var expected interface{}
	err := json.Unmarshal([]byte(`{"foo": "bar"}`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	var actual interface{}
	err = json.Unmarshal([]byte(`{"foo": "bar"}`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result := Result{
		Expected: Response{JSON: expected},
		Actual:   Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// bad value
	err = json.Unmarshal([]byte(`{"foo": "baz"}`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result = Result{
		Expected: Response{JSON: expected},
		Actual:   Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != `diffs in json: expected:foo = "bar" != actual:foo = "baz"` {
		t.Fatalf("bad error message: %s", err.Error())
	}
	// error bad types
	err = json.Unmarshal([]byte(`["foo", "bar"]`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result = Result{
		Expected: Response{JSON: expected},
		Actual:   Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed")
	}
	if err.Error() != "generating JSON diff: types do not match (cause count 0)" {
		t.Fatalf("bad error message: %s", err.Error())
	}
}

func TestAssertResponseJsonExcludes(t *testing.T) {
	// nominal case
	var expected interface{}
	err := json.Unmarshal([]byte(`{"foo": "bar"}`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	var actual interface{}
	err = json.Unmarshal([]byte(`{"foo": "baz"}`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result := Result{
		Expected: Response{
			JSON:         expected,
			JSONExcludes: []string{"foo"},
		},
		Actual: Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// star in excludes
	err = json.Unmarshal([]byte(`{"foo": {"bar": "baz"}}`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	err = json.Unmarshal([]byte(`{"foo": {"bar": "spam"}}`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result = Result{
		Expected: Response{
			JSON:         expected,
			JSONExcludes: []string{"*/bar"},
		},
		Actual: Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// index and star in excludes
	err = json.Unmarshal([]byte(`[{"foo": {"bar": "baz"}}]`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	err = json.Unmarshal([]byte(`[{"foo": {"bar": "spam"}}]`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result = Result{
		Expected: Response{
			JSON:         expected,
			JSONExcludes: []string{"0/*/bar"},
		},
		Actual: Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// double star in excludes
	err = json.Unmarshal([]byte(`{"foo": {"bar": {"spam": "eggs"}}}`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	err = json.Unmarshal([]byte(`{"foo": {"bar": {"spam": "fish"}}}`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result = Result{
		Expected: Response{
			JSON:         expected,
			JSONExcludes: []string{"**/spam"},
		},
		Actual: Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
}

func TestAssertResponseJsonRegexps(t *testing.T) {
	// nominal case
	var expected interface{}
	err := json.Unmarshal([]byte(`{"foo": "b.r"}`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	var actual interface{}
	err = json.Unmarshal([]byte(`{"foo": "bar"}`), &actual)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result := Result{
		Expected: Response{
			JSON:        expected,
			JSONRegexps: []string{"foo"},
		},
		Actual: Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// no match
	err = json.Unmarshal([]byte(`{"foo": "x.*"}`), &expected)
	if err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}
	result = Result{
		Expected: Response{
			JSON:        expected,
			JSONRegexps: []string{"foo"},
		},
		Actual: Response{JSON: actual},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatalf("should have failed: %v", err)
	}
	if err.Error() != `diffs in json: expected:foo = "x.*" !~ actual:foo = "bar"` {
		t.Fatalf("bad diff message: %v", err)
	}
}

func TestFilterChangelog(t *testing.T) {
	changelog := []diff.Change{
		{
			Type: diff.UPDATE,
			From: "spam",
			To:   "eggs",
			Path: []string{"foo", "bar"},
		},
	}
	// simple two levels path
	filters := []string{"foo/bar"}
	filtered, err := FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// path with star in first position
	filters = []string{"*/bar"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// path with star in second position
	filters = []string{"foo/*"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// path with two stars
	filters = []string{"*/*"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// path with a double star
	filters = []string{"**"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// path with a double star and path
	filters = []string{"**/bar"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// path with path and a double star
	filters = []string{"foo/**"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("changelog should have been filtered")
	}
	// no filter
	filters = []string{"spam/eggs"}
	filtered, err = FilterChangelogExcludes(changelog, filters)
	if err != nil {
		t.Fatalf("error filtering changelog: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("changelog should not have been filtered")
	}
}

func TestChangeMessage(t *testing.T) {
	change := diff.Change{
		Type: diff.UPDATE,
		From: "spam",
		To:   "eggs",
		Path: []string{"foo", "bar"},
	}
	// update
	message := ChangeMessage(change, nil)
	if message != `expected:foo/bar = "spam" != actual:foo/bar = "eggs"` {
		t.Fatalf("bad change message: %v", message)
	}
	// update with regexp
	message = ChangeMessage(change, []string{"foo/bar"})
	if message != `expected:foo/bar = "spam" !~ actual:foo/bar = "eggs"` {
		t.Fatalf("bad change message: %v", message)
	}
	// create
	change.Type = diff.CREATE
	message = ChangeMessage(change, nil)
	if message != `actual:foo/bar = "eggs" not in expected` {
		t.Fatalf("bad change message: %v", message)
	}
	// delete
	change.Type = diff.DELETE
	message = ChangeMessage(change, nil)
	if message != `expected:foo/bar = "spam" not in actual` {
		t.Fatalf("bad change message: %v", message)
	}
	// unknown
	change.Type = "unknown"
	message = ChangeMessage(change, nil)
	if message != `UNKNOWN TYPE unknown` {
		t.Fatalf("bad change message: %v", message)
	}
}

func TestFormatPath(t *testing.T) {
	path := []string{"foo", "bar"}
	if FormatPath(path) != "foo/bar" {
		t.Fatalf("bad path format: %s", FormatPath(path))
	}
}

func TestPathToRegexp(t *testing.T) {
	path := "foo/bar"
	regex := PathToRegexp(path)
	if regex != "^foo/bar$" {
		t.Fatalf("bad path regexp: %s", regex)
	}
	path = "*/bar"
	regex = PathToRegexp(path)
	if regex != "^[^/]+/bar$" {
		t.Fatalf("bad path regexp: %s", regex)
	}
	path = "foo/*"
	regex = PathToRegexp(path)
	if regex != "^foo/[^/]+$" {
		t.Fatalf("bad path regexp: %s", regex)
	}
	path = "foo/**"
	regex = PathToRegexp(path)
	if regex != "^foo/.*/?$" {
		t.Fatalf("bad path regexp: %s", regex)
	}
	path = "**/bar"
	regex = PathToRegexp(path)
	if regex != "^.*/?bar$" {
		t.Fatalf("bad path regexp: %s", regex)
	}
}

func TestCheckAssertions(t *testing.T) {
	// can't set both body and bodyRegexps assertions
	result := Result{
		Expected: Response{
			Body:       "test",
			BodyRegexp: "test",
		},
	}
	err := AssertResponse(result)
	if err == nil {
		t.Fatal("should have failed")
	}
	if err.Error() != "you can set both body and bodyRegexps assertions" {
		t.Fatalf("bad error message: %s", err.Error())
	}
	// field declared as regexp but not found in headers list
	result = Result{
		Expected: Response{
			Headers:        Headers{"spam": "eggs"},
			HeadersRegexps: []string{"foo"},
		},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatal("should have failed")
	}
	if err.Error() != "field foo declared as regexp but not found in headers list" {
		t.Fatalf("bad error message: %s", err.Error())
	}
	// JSON field can't be excluded and declared as regexp
	result = Result{
		Expected: Response{
			JSONExcludes: []string{"foo"},
			JSONRegexps:  []string{"foo"},
		},
	}
	err = AssertResponse(result)
	if err == nil {
		t.Fatal("should have failed")
	}
	if err.Error() != "JSON field 'foo' can't be excluded and declared as regexp" {
		t.Fatalf("bad error message: %s", err.Error())
	}
}
