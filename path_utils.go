package venom

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveWorkdirPath joins userPath under workdir and ensures the result
// stays inside workdir. Absolute paths and "../" escapes are rejected.
// Returns the cleaned path, or an error if the input is unsafe.
func ResolveWorkdirPath(workdir, userPath string) (string, error) {
	if filepath.IsAbs(userPath) {
		return "", fmt.Errorf("absolute path %q is not allowed; use a path relative to the testsuite workdir", userPath)
	}

	cleanWorkdir := filepath.Clean(workdir)
	joined := filepath.Join(cleanWorkdir, userPath)

	rel, err := filepath.Rel(cleanWorkdir, joined)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", userPath, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the testsuite workdir", userPath)
	}

	return joined, nil
}
