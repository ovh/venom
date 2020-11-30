package readfile

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-zglob"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/venom"
)

// Name for test readfile
const Name = "readfile"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// Result represents a step result
type Result struct {
	Content     string            `json:"content,omitempty" yaml:"content,omitempty"`
	ContentJSON interface{}       `json:"contentjson,omitempty" yaml:"contentjson,omitempty"`
	Err         string            `json:"error" yaml:"error"`
	TimeSeconds float64           `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	Md5sum      map[string]string `json:"md5sum,omitempty" yaml:"md5sum,omitempty"`
	Size        map[string]int64  `json:"size,omitempty" yaml:"size,omitempty"`
	ModTime     map[string]int64  `json:"modtime,omitempty" yaml:"modtime,omitempty"`
	Mod         map[string]string `json:"mod,omitempty" yaml:"mod,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	r := Result{
		Md5sum:  make(map[string]string),
		Size:    make(map[string]int64),
		ModTime: make(map[string]int64),
		Mod:     make(map[string]string),
	}
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	if e.Path == "" {
		return nil, fmt.Errorf("Invalid path")
	}

	start := time.Now()

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")
	result, errr := e.readfile(workdir)
	if errr != nil {
		result.Err = errr.Error()
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func (e *Executor) readfile(workdir string) (Result, error) {
	result := Result{}

	absPath := filepath.Join(workdir, e.Path)

	fileInfo, _ := os.Stat(absPath) // nolint
	if fileInfo != nil && fileInfo.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	filesPath, errg := zglob.Glob(absPath)
	if errg != nil {
		return result, fmt.Errorf("Error reading files on path:%s :%s", absPath, errg)
	}

	if len(filesPath) == 0 {
		return result, fmt.Errorf("Invalid path '%s' or file not found", absPath)
	}

	var content string
	md5sum := make(map[string]string)
	size := make(map[string]int64)
	modtime := make(map[string]int64)
	mod := make(map[string]string)

	for _, f := range filesPath {
		f, erro := os.Open(f)
		if erro != nil {
			return result, fmt.Errorf("Error while opening file: %s", erro)
		}
		defer f.Close()

		relativeName, err := filepath.Rel(workdir, f.Name())
		if err != nil {
			return result, fmt.Errorf("Error cannot evaluate relative path to file at %s: %s", f.Name(), err)
		}

		h := md5.New()
		tee := io.TeeReader(f, h)

		b, errr := ioutil.ReadAll(tee)
		if errr != nil {
			return result, fmt.Errorf("Error while reading file: %s", errr)
		}
		content += string(b)

		md5sum[relativeName] = hex.EncodeToString(h.Sum(nil))

		stat, errs := f.Stat()
		if errs != nil {
			return result, fmt.Errorf("Error while compute file size: %s", errs)
		}

		size[relativeName] = stat.Size()
		modtime[relativeName] = stat.ModTime().Unix()
		mod[relativeName] = stat.Mode().String()
	}

	result.Content = content

	var m interface{}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.UseNumber()
	if err := decoder.Decode(&m); err == nil {
		result.ContentJSON = m
	}

	result.Md5sum = md5sum
	result.Size = size
	result.ModTime = modtime
	result.Mod = mod

	return result, nil
}
