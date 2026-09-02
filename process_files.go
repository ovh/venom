package venom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-zglob"
	"github.com/rockbears/yaml"
	yamlv3 "gopkg.in/yaml.v3"

	"github.com/ovh/venom/interpolate"
	"github.com/pkg/errors"
)

// testCaseLineInfo holds source line numbers for a single test case.
type testCaseLineInfo struct {
	TestCaseLine   int     // line of the testcase mapping node
	StepLines      []int   // line of each step mapping node
	AssertionLines [][]int // [stepIdx][assertIdx] line numbers
}

// extractLineNumbers parses raw YAML bytes using gopkg.in/yaml.v3 to extract
// source line numbers for testcases, steps, and assertions.
func extractLineNumbers(content []byte) []testCaseLineInfo {
	var doc yamlv3.Node
	if err := yamlv3.Unmarshal(content, &doc); err != nil {
		return nil
	}

	// doc is a DocumentNode containing the root MappingNode
	if doc.Kind != yamlv3.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yamlv3.MappingNode {
		return nil
	}

	// Find "testcases" key in root mapping
	var testcasesNode *yamlv3.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "testcases" {
			testcasesNode = root.Content[i+1]
			break
		}
	}
	if testcasesNode == nil || testcasesNode.Kind != yamlv3.SequenceNode {
		return nil
	}

	result := make([]testCaseLineInfo, 0, len(testcasesNode.Content))
	for _, tcNode := range testcasesNode.Content {
		if tcNode.Kind != yamlv3.MappingNode {
			result = append(result, testCaseLineInfo{})
			continue
		}

		info := testCaseLineInfo{
			TestCaseLine: tcNode.Line,
		}

		// Find "steps" in testcase mapping
		var stepsNode *yamlv3.Node
		for i := 0; i < len(tcNode.Content)-1; i += 2 {
			if tcNode.Content[i].Value == "steps" {
				stepsNode = tcNode.Content[i+1]
				break
			}
		}

		if stepsNode != nil && stepsNode.Kind == yamlv3.SequenceNode {
			info.StepLines = make([]int, len(stepsNode.Content))
			info.AssertionLines = make([][]int, len(stepsNode.Content))

			for stepIdx, stepNode := range stepsNode.Content {
				info.StepLines[stepIdx] = stepNode.Line

				if stepNode.Kind != yamlv3.MappingNode {
					continue
				}

				// Find "assertions" in step mapping
				for i := 0; i < len(stepNode.Content)-1; i += 2 {
					if stepNode.Content[i].Value == "assertions" {
						assertionsNode := stepNode.Content[i+1]
						if assertionsNode.Kind == yamlv3.SequenceNode {
							info.AssertionLines[stepIdx] = make([]int, len(assertionsNode.Content))
							for aIdx, aNode := range assertionsNode.Content {
								info.AssertionLines[stepIdx][aIdx] = aNode.Line
							}
						}
						break
					}
				}
			}
		}

		result = append(result, info)
	}
	return result
}

func getFilesPath(path []string) ([]string, error) {
	filePaths := make([]string, 0)

	for _, p := range path {
		p = strings.TrimSpace(p)

		// no need to check err on os.stat.
		// if we put ./test/*.yml, it will fail and it's normal
		fileInfo, _ := os.Stat(p)
		if fileInfo != nil && fileInfo.IsDir() {
			// check if *.yml or *.yaml files exists in the path
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
		Debug(ctx, "Reading %v", filePath)
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
			Name:        testSuiteInput.Name,
			Description: testSuiteInput.Description,
			TestCases:   make([]TestCase, len(testSuiteInput.TestCases)),
			Vars:        testSuiteInput.Vars,
			Secrets:     testSuiteInput.Secrets,
		}

		// Extract source line numbers from original YAML (before interpolation)
		lineInfos := extractLineNumbers(btes)

		for i := range testSuiteInput.TestCases {
			ts.TestCases[i] = TestCase{
				TestCaseInput: testSuiteInput.TestCases[i],
			}
			if i < len(lineInfos) {
				ts.TestCases[i].SourceLine = lineInfos[i].TestCaseLine
				ts.TestCases[i].StepSourceLines = lineInfos[i].StepLines
				ts.TestCases[i].AssertionSourceLines = lineInfos[i].AssertionLines
			}
		}
		Info(ctx, "Has %d Secrets", len(ts.Secrets))

		// Default workdir is testsuite directory
		ts.WorkDir, err = filepath.Abs(filepath.Dir(filePath))
		if runtime.GOOS == "windows" {
			// Replace backslashes with forward slashes for Windows
			ts.WorkDir = strings.ReplaceAll(ts.WorkDir, "\\", "/")
		}
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
