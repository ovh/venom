package venom

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ovh/cds/sdk/interpolate"
)

func (v *Venom) runTestSuite(ctx context.Context, ts *TestSuite, log Logger) {
	if v.EnableProfiling {
		var filename, filenameCPU, filenameMem string
		if v.ReportDir != "" {
			filename = v.ReportDir + "/"
		}
		filenameCPU = filename + "pprof_cpu_profile_" + ts.Filename + ".prof"
		filenameMem = filename + "pprof_mem_profile_" + ts.Filename + ".prof"
		fCPU, errCPU := os.Create(filenameCPU)
		fMem, errMem := os.Create(filenameMem)
		if errCPU != nil || errMem != nil {
			log.Errorf("error while create profile file CPU:%v MEM:%v", errCPU, errMem)
		} else {
			pprof.StartCPUProfile(fCPU)
			p := pprof.Lookup("heap")
			defer p.WriteTo(fMem, 1)
			defer pprof.StopCPUProfile()
		}
	}

	start := time.Now()
	log.Infof("Starting test suite %s", ts.ShortName)
	defer func() {
		log.Infof("End (%.3f seconds)", time.Since(start).Seconds())
	}()

	// init variables on testsuite level
	if ts.Vars == nil {
		ts.Vars = H{}
	}
	ts.Vars.AddAll(v.variables)
	ts.Vars.Add("venom.testsuite", ts.ShortName)
	ts.Vars.Add("venom.testsuite.filename", ts.Filename)

	for k, val := range ts.Vars {
		log.Debugf("Interpolating variable '%s'='%s'", k, val)
		newval, err := interpolate.Do(val, ts.Vars)
		if err != nil {
			v.logger.Errorf("interpolation error on %s: %v", val, err)
			continue
		}
		ts.Vars[k] = newval
	}

	totalSteps := 0
	for _, tc := range ts.TestCases {
		totalSteps += len(tc.TestSteps)
	}

	v.runTestCases(ctx, ts, log)

	elapsed := time.Since(start)

	var output string
	var detailPerTestcase [][]string
	for _, tc := range ts.TestCases {
		symbolStatus := "✓"
		colorFunc := colorSuccess
		if tc.HasFailureOrError() {
			symbolStatus = "✗"
			colorFunc = colorFailure
		}

		var detail []string
		s := colorFunc(" "+symbolStatus+" "+tc.Name) + " [" + tc.ShortName + "]"
		detail = append(detail, s)
		for _, failure := range tc.Failures {
			s := colorFailure("   - " + failure.Value)
			detail = append(detail, s)
		}
		for _, err := range tc.Errors {
			s := colorFailure("   - " + err.Value)
			detail = append(detail, s)
		}
		detailPerTestcase = append(detailPerTestcase, detail)
	}

	var hasFailure bool
	if ts.Failures > 0 || ts.Errors > 0 {
		hasFailure = true
		output = fmt.Sprintf("%s %s", colorFailure("FAILURE"), rightPad(ts.Package, " ", 47))
	} else {
		output = fmt.Sprintf("%s %s", colorSuccess("SUCCESS"), rightPad(ts.Package, " ", 47))
	}

	output += fmt.Sprintf("%.3f seconds", elapsed.Seconds())
	fmt.Fprintln(v.Output, output)
	if hasFailure {
		for _, detail := range detailPerTestcase {
			fmt.Fprint(v.Output, strings.Join(detail, "\n"))
		}
	}
	log.Infof(output)
}

//Parse the suite to find unreplaced and extracted variables
//func (v *Venom) parseTestSuite(ts *TestSuite) ([]string, []string, error) {
//	d, err := dump.ToStringMap(ts.Vars)
//	if err != nil {
//		v.logger.Errorf("err:%s", err)
//	}
//	ts.Templater.Add("", d)
//
//	return v.parseTestCases(ts)
//}

//Parse the testscases to find unreplaced and extracted variables
//func (v *Venom) parseTestCases(ts *TestSuite) ([]string, []string, error) {
//	vars := []string{}
//	extractsVars := []string{}
//	for i := range ts.TestCases {
//		tc := &ts.TestCases[i]
//		if len(tc.Skipped) == 0 {
//			tvars, tExtractedVars, err := v.parseTestCase(ts, tc)
//			if err != nil {
//				return nil, nil, err
//			}
//			for _, k := range tvars {
//				var found bool
//				for i := 0; i < len(vars); i++ {
//					if vars[i] == k {
//						found = true
//						break
//					}
//				}
//				if !found {
//					vars = append(vars, k)
//				}
//			}
//			for _, k := range tExtractedVars {
//				var found bool
//				for i := 0; i < len(extractsVars); i++ {
//					if extractsVars[i] == k {
//						found = true
//						break
//					}
//				}
//				if !found {
//					extractsVars = append(extractsVars, k)
//				}
//			}
//		}
//	}
//
//	return vars, extractsVars, nil
//}
//
