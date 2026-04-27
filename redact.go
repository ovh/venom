package venom

import (
	"regexp"
	"strings"
)

// mysqlDSNRegexp matches MySQL-style DSNs of the form
// "user:password@tcp(host:port)/dbname". The capture group keeps the
// username so the redacted value still identifies the connection.
var mysqlDSNRegexp = regexp.MustCompile(`^([^:@/\s]+):[^@]*@`)

// RedactURI returns a copy of s with any embedded credentials replaced by
// "***". Two formats are recognised:
//
//   - URI form ("scheme://user:password@host/..."): the password component
//     of the userinfo is masked.
//   - MySQL DSN form ("user:password@tcp(host:port)/db"): the password is
//     masked, the username is preserved.
//
// Inputs that do not match a known credential pattern are returned
// unchanged, so plain SQLite paths or ":memory:" are passed through.
func RedactURI(s string) string {
	if s == "" {
		return s
	}
	if i := strings.Index(s, "://"); i >= 0 {
		// URI form. We do not use net/url for the rebuild because
		// url.URL.String escapes "*" into "%2A" in the userinfo, which
		// makes the redacted value unreadable.
		scheme := s[:i]
		rest := s[i+3:]
		at := strings.Index(rest, "@")
		if at < 0 {
			return s
		}
		userinfo := rest[:at]
		afterAt := rest[at:] // starts with '@'
		colon := strings.Index(userinfo, ":")
		if colon < 0 {
			return s // no password to redact
		}
		return scheme + "://" + userinfo[:colon] + ":***" + afterAt
	}
	if mysqlDSNRegexp.MatchString(s) {
		return mysqlDSNRegexp.ReplaceAllString(s, "$1:***@")
	}
	return s
}
