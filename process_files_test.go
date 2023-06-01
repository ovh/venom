package venom

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func tempDir(t *testing.T) (string, error) {
	dir := os.TempDir()
	name := path.Join(dir, randomString(5))
	if err := os.MkdirAll(name, os.FileMode(0744)); err != nil {
		return "", err
	}
	t.Logf("Creating directory %s", name)
	return name, nil
}

func randomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func Test_getFilesPath(t *testing.T) {
	InitTestLogger(t)
	rand.Seed(time.Now().UnixNano())

	tests := []struct {
		init    func(t *testing.T) ([]string, error)
		name    string
		want    []string
		wantErr bool
	}{
		{
			name: "Check an empty directory",
			init: func(t *testing.T) ([]string, error) {
				dir, err := tempDir(t)
				return []string{dir}, err
			},
			wantErr: true,
		},
		{
			name: "Check an directory with one yaml file",
			init: func(t *testing.T) ([]string, error) {
				dir, err := tempDir(t)
				if err != nil {
					return nil, err
				}

				d1 := []byte("hello")
				err = os.WriteFile(path.Join(dir, "d1.yml"), d1, 0644)
				return []string{dir}, err
			},
			want:    []string{"d1.yml"},
			wantErr: false,
		},
		{
			name: "Check an directory with one yaml file and a subdirectory with another file",
			init: func(t *testing.T) ([]string, error) {
				dir1, err := tempDir(t)
				if err != nil {
					return nil, err
				}

				d1 := []byte("hello")
				if err = os.WriteFile(path.Join(dir1, "d1.yml"), d1, 0644); err != nil {
					return nil, err
				}

				dir2 := path.Join(dir1, randomString(10))
				t.Logf("Creating directory %s", dir2)

				if err := os.Mkdir(dir2, 0744); err != nil {
					return nil, err
				}

				d2 := []byte("hello")
				if err = os.WriteFile(path.Join(dir2, "d2.yml"), d2, 0644); err != nil {
					return nil, err
				}

				return []string{dir1, dir2}, err
			},
			want:    []string{"d1.yml", "d2.yml"},
			wantErr: false,
		},
		{
			name: "Check globstars",
			init: func(t *testing.T) ([]string, error) {
				dir1, err := tempDir(t)
				if err != nil {
					return nil, err
				}

				d1 := []byte("hello")
				if err = os.WriteFile(path.Join(dir1, "d1.yml"), d1, 0644); err != nil {
					return nil, err
				}

				dir2 := path.Join(dir1, randomString(10))
				t.Logf("Creating directory %s", dir2)

				if err := os.Mkdir(dir2, 0744); err != nil {
					return nil, err
				}

				d2 := []byte("hello")
				if err = os.WriteFile(path.Join(dir2, "d2.yml"), d2, 0644); err != nil {
					return nil, err
				}

				dir3 := path.Join(dir2, randomString(10))
				t.Logf("Creating directory %s", dir3)

				if err := os.Mkdir(dir3, 0744); err != nil {
					return nil, err
				}

				d3 := []byte("hello")
				if err = os.WriteFile(path.Join(dir2, "d3.yml"), d3, 0644); err != nil {
					return nil, err
				}

				dir4 := path.Join(dir3, randomString(10))
				t.Logf("Creating directory %s", dir3)

				if err := os.Mkdir(dir4, 0744); err != nil {
					return nil, err
				}

				d4 := []byte("hello")
				if err = os.WriteFile(path.Join(dir4, "d4.yml"), d4, 0644); err != nil {
					return nil, err
				}

				return []string{fmt.Sprintf("%s/**/*.yml", dir1)}, err
			},
			want:    []string{"d1.yml", "d2.yml", "d3.yml", "d4.yml"},
			wantErr: false,
		},
		{
			name: "Check globstars with duplicate files",
			init: func(t *testing.T) ([]string, error) {
				dir1, err := tempDir(t)
				if err != nil {
					return nil, err
				}

				d1 := []byte("hello")
				if err = os.WriteFile(path.Join(dir1, "d1.yml"), d1, 0644); err != nil {
					return nil, err
				}

				dir2 := path.Join(dir1, randomString(10))
				t.Logf("Creating directory %s", dir2)

				if err := os.Mkdir(dir2, 0744); err != nil {
					return nil, err
				}

				d2 := []byte("hello")
				if err = os.WriteFile(path.Join(dir2, "d2.yml"), d2, 0644); err != nil {
					return nil, err
				}

				dir3 := path.Join(dir2, randomString(10))
				t.Logf("Creating directory %s", dir3)

				if err := os.Mkdir(dir3, 0744); err != nil {
					return nil, err
				}

				d3 := []byte("hello")
				if err = os.WriteFile(path.Join(dir2, "d3.yml"), d3, 0644); err != nil {
					return nil, err
				}

				dir4 := path.Join(dir3, randomString(10))
				t.Logf("Creating directory %s", dir3)

				if err := os.Mkdir(dir4, 0744); err != nil {
					return nil, err
				}

				d4 := []byte("hello")
				if err = os.WriteFile(path.Join(dir4, "d4.yml"), d4, 0644); err != nil {
					return nil, err
				}

				return []string{dir2, dir3, fmt.Sprintf("%s/**/*.yml", dir1)}, err
			},
			want:    []string{"d1.yml", "d2.yml", "d3.yml", "d4.yml"},
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.init(t)
			if err != nil {
				t.Fatal(err)
			}

			got, err := getFilesPath(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFilesPath() name:%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			for _, f := range tt.want {
				var found bool
				for _, g := range got {
					if strings.HasSuffix(g, f) {
						found = true
					}
				}
				if !found {
					t.Errorf("getFilesPath() error want %v got %v", f, got)
				}
			}
		})
	}
}

func Test_getFilesPath_files_order(t *testing.T) {
	dir1, _ := tempDir(t)

	d1 := []byte("hello")
	os.WriteFile(path.Join(dir1, "a.yml"), d1, 0644)

	d2 := []byte("hello")
	os.WriteFile(path.Join(dir1, "A.yml"), d2, 0644)

	input := []string{dir1 + "/a.yml", dir1 + "/A.yml"}

	output, err := getFilesPath(input)
	require.NoError(t, err)
	require.Len(t, output, 2)
	t.Log(output)
	require.True(t, strings.HasSuffix(output[0], "a.yml"))
	require.True(t, strings.HasSuffix(output[1], "A.yml"))
}
