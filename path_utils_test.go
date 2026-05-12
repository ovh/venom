package venom

import (
	"path/filepath"
	"testing"
)

func TestResolveWorkdirPath(t *testing.T) {
	workdir := filepath.Clean("/tmp/venom-workdir")

	tests := []struct {
		name     string
		userPath string
		wantErr  bool
	}{
		{name: "simple relative path", userPath: "fixtures/data.yml", wantErr: false},
		{name: "nested relative path", userPath: "a/b/c/file.txt", wantErr: false},
		{name: "current dir", userPath: ".", wantErr: false},
		{name: "absolute path rejected", userPath: "/etc/passwd", wantErr: true},
		{name: "parent escape rejected", userPath: "../secret", wantErr: true},
		{name: "deep parent escape rejected", userPath: "a/../../etc/passwd", wantErr: true},
		{name: "exact escape to parent rejected", userPath: "..", wantErr: true},
		{name: "empty path resolves to workdir", userPath: "", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveWorkdirPath(workdir, tt.userPath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got resolved path %q", tt.userPath, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.userPath, err)
			}
			rel, errRel := filepath.Rel(workdir, got)
			if errRel != nil || rel == ".." || len(rel) >= 2 && rel[:2] == ".." {
				t.Fatalf("resolved path %q is outside workdir", got)
			}
		})
	}
}
