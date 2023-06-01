package venom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-zglob"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/sdk/interpolate"
	"github.com/pkg/errors"
)

func getFilesPath(path []string) ([]string, error) {
	filePaths := make([]string, 0)

	for _, p := range path {
		p = strings.TrimSpace(p)

		// no need to check err on os.stat.
		// if we put ./test/*.yml, it will fail and it's normal
		fileInfo, _ := os.Stat(p)
		if fileInfo != nil && fileInfo.IsDir() {
			//check if *.yml or *.yaml files exists in the path
			p = p + string(os.PathSeparator) + "*.y*ml"
		}

		fpaths, err := zglob.Glob(p)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading files on path %q", path)
		}

		for _, fp := range fpaths {
			switch ext := filepath.Ext(fp); ext {
			case ".yml", ".yaml":
				filePaths = append(filePaths, fp)
			}
		}
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no YAML (*.yml or *.yaml) file found or defined")
	}
	return uniq(filePaths), nil
}

func uniq(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func (v *Venom) readFiles(ctx context.Context, filesPath []string) (err error) {
	for _, filePath := range filesPath {
		Info(ctx, "Reading %v", filePath)
		btes, err := os.ReadFile(filePath)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %q", filePath)
		}

		varCloned := v.variables.Clone()

		fromPartial, err := getVarFromPartialYML(ctx, btes)
		if err != nil {
			return errors.Wrapf(err, "unable to get vars from file %q", filePath)
		}

		var varsFromPartial map[string]string
		if len(fromPartial) > 0 {
			varsFromPartial, err = DumpStringPreserveCase(fromPartial)
			if err != nil {
				return errors.Wrapf(err, "unable to parse variables")
			}
		}

		// we take default vars from the testsuite, only if it's not already is global vars
		for k, value := range varsFromPartial {
			if k == "" {
				continue
			}
			if _, ok := varCloned[k]; !ok || (varCloned[k] == "{}" && varCloned["__Len__"] == "0") {
				// we interpolate the value of vars here, to do it only once per ts
				valueInterpolated, err := interpolate.Do(value, varsFromPartial)
				if err != nil {
					return errors.Wrapf(err, "unable to parse variable %q", k)
				}
				varCloned.Add(k, valueInterpolated)
			}
		}

		var vars map[string]string
		if len(varCloned) > 0 {
			vars, err = DumpStringPreserveCase(varCloned)
			if err != nil {
				return errors.Wrapf(err, "unable to parse variables")
			}
		}

		content, err := interpolate.Do(string(btes), vars)
		if err != nil {
			return err
		}

		var testSuiteInput TestSuiteInput
		if err := yaml.Unmarshal([]byte(content), &testSuiteInput); err != nil {
			Error(context.Background(), "file content: %s", content)
			return errors.Wrapf(err, "error while unmarshal file %q", filePath)
		}

		ts := TestSuite{
			Name:      testSuiteInput.Name,
			TestCases: make([]TestCase, len(testSuiteInput.TestCases)),
			Vars:      testSuiteInput.Vars,
		}
		for i := range testSuiteInput.TestCases {
			ts.TestCases[i] = TestCase{
				TestCaseInput: testSuiteInput.TestCases[i],
			}
		}

		// Default workdir is testsuite directory
		ts.WorkDir, err = filepath.Abs(filepath.Dir(filePath))
		if err != nil {
			return errors.Wrapf(err, "Unable to get testsuite's working directory")
		}

		// ../foo/a.yml
		ts.Filepath = filePath
		// a
		ts.ShortName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		// a.yml
		ts.Filename = filepath.Base(filePath)
		ts.Vars = varCloned

		ts.Vars.Add("venom.testsuite.workdir", ts.WorkDir)
		ts.Vars.Add("venom.testsuite.name", ts.Name)
		ts.Vars.Add("venom.testsuite.shortName", ts.ShortName)
		ts.Vars.Add("venom.testsuite.filename", ts.Filename)
		ts.Vars.Add("venom.testsuite.filepath", ts.Filepath)
		ts.Vars.Add("venom.datetime", time.Now().Format(time.RFC3339))
		ts.Vars.Add("venom.timestamp", fmt.Sprintf("%d", time.Now().Unix()))

		v.Tests.TestSuites = append(v.Tests.TestSuites, ts)
	}
	return nil
}
