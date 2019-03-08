package venom

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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
			logrus.Errorf("Error reading files on path:%s :%s", path, err)
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
	return filePaths, nil
}

func (v *Venom) readFiles(filesPath []string) (err error) {
	for _, f := range filesPath {
		v.logger.Infof("Reading %s", f)
		rawData, err := ioutil.ReadFile(f)
		if err != nil {
			return fmt.Errorf("error while reading file %s: %v", f, err)
		}

		interpolatedData, err := interpolate.Do(string(rawData), v.variables)
		if err != nil {
			return err
		}

		ts := TestSuite{}
		//		ts.Templater = newTemplater(v.variables)
		ts.Package = f

		// Apply templater unitl there is no more modifications
		// it permits to include testcase from env
		//		_, out := ts.Templater.apply(dat)
		//		for i := 0; i < 10; i++ {
		//			_, tmp := ts.Templater.apply(out)
		//			if string(tmp) == string(out) {
		//				break
		//			}
		//			out = tmp
		//		}

		switch ext := filepath.Ext(f); ext {
		case ".hcl":
			err = hcl.Unmarshal([]byte(interpolatedData), &ts)
		case ".yaml", ".yml":
			err = yaml.Unmarshal([]byte(interpolatedData), &ts)
		default:
			return fmt.Errorf("unsupported test suite file extension: %q", ext)
		}
		if err != nil {
			return fmt.Errorf("Error while unmarshal file %s err: %v", f, err)
		}

		ts.ShortName = ts.Name
		ts.Name += " [" + f + "]"
		ts.Filename = f

		if ts.Version != "" && !strings.HasPrefix(ts.Version, "1") {
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
			nSteps += len(tc.TestSteps)
			if len(tc.Skipped) >= 1 {
				ts.Skipped += len(tc.Skipped)
			}
		}
		ts.Total = len(ts.TestCases)

		v.testsuites = append(v.testsuites, ts)
	}
	return nil
}
