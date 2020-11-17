package venom

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/ghodss/yaml"

	"github.com/ovh/cds/sdk/interpolate"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func getFilesPath(path []string) (filePaths []string, err error) {
	for _, p := range path {
		p = strings.TrimSpace(p)

		// no need to check err on os.stat.
		// if we put ./test/*.yml, it will fail and it's normal
		fileInfo, _ := os.Stat(p)
		if fileInfo != nil && fileInfo.IsDir() {
			p = p + string(os.PathSeparator) + "*.yml"
		}

		fpaths, err := filepath.Glob(p)
		if err != nil {
			log.Errorf("error reading files on path:%s :%s", path, err)
			return nil, errors.Wrapf(err, "error reading files on path %q", path)
		}

		for _, fp := range fpaths {
			switch ext := filepath.Ext(fp); ext {
			case ".hcl", ".yml", ".yaml":
				filePaths = append(filePaths, fp)
			}
		}
	}

	sort.Strings(filePaths)
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no yml file selected")
	}
	return filePaths, nil
}

type partialTestSuite struct {
	Name string `json:"name" yaml:"name"`
	Vars H      `yaml:"vars" json:"vars"`
}

func (v *Venom) readFiles(filesPath []string) (err error) {
	for _, f := range filesPath {
		log.Info("Reading ", f)
		btes, err := ioutil.ReadFile(f)
		if err != nil {
			return fmt.Errorf("unable to read file %s err:%v", f, err)
		}

		vars, err := dump.ToStringMap(v.variables)
		if err != nil {
			return fmt.Errorf("unable to parse variables :%v", err)
		}

		content, err := interpolate.Do(string(btes), vars)
		if err != nil {
			return err
		}

		var partialTs partialTestSuite
		if err := yaml.Unmarshal([]byte(content), &partialTs); err != nil {
			Error(context.Background(), "file content: %s", content)
			return fmt.Errorf("error while unmarshal file %s err: %v", f, err)
		}

		var ts TestSuite
		if err := yaml.Unmarshal([]byte(content), &ts); err != nil {
			Error(context.Background(), "file content: %s", content)
			return fmt.Errorf("error while unmarshal file %s err: %v", f, err)
		}

		ts.Package = f
		ts.ShortName = ts.Name
		ts.Filename = f
		ts.Vars = partialTs.Vars.Clone()

		// Default workdir is testsuite directory
		if ts.Version == "" || !strings.HasPrefix(ts.Version, "1") {
			ts.WorkDir, err = filepath.Abs(filepath.Dir(f))
			if err != nil {
				return fmt.Errorf("Unable to get testsuite's working directory err:%s", err)
			}
		} else {
			ts.WorkDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("Unable to get current working directory err:%s", err)
			}
		}

		nSteps := 0
		for _, tc := range ts.TestCases {
			nSteps += len(tc.testSteps)
			if len(tc.Skipped) >= 1 {
				ts.Skipped += len(tc.Skipped)
			}
		}
		ts.Total = len(ts.TestCases)

		v.testsuites = append(v.testsuites, ts)
	}
	return nil
}
