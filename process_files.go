package venom

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/yaml.v2"
)

func getFilesPath(path []string) []string {
	var filesPath []string
	for _, p := range path {
		fileInfo, _ := os.Stat(p)
		if fileInfo != nil && fileInfo.IsDir() {
			p = filepath.Dir(p) + "/*.yml"
			log.Debugf("path computed:%s", path)
		}
		fpaths, errg := filepath.Glob(p)
		if errg != nil {
			log.Fatalf("Error reading files on path:%s :%s", path, errg)
		}
		for _, fp := range fpaths {
			if strings.HasSuffix(fp, ".yml") || strings.HasSuffix(fp, ".yaml") {
				filesPath = append(filesPath, fp)
			} else {
				log.Debugf("%s is skipped (not yaml extension)", fp)
			}
		}
	}

	sort.Strings(filesPath)
	return filesPath
}

func readFiles(detailsLevel string, filesPath []string, chanToRun chan<- TestSuite) map[string]*pb.ProgressBar {
	bars := make(map[string]*pb.ProgressBar)
	for _, f := range filesPath {
		log.Debugf("read %s", f)
		dat, errr := ioutil.ReadFile(f)
		if errr != nil {
			log.WithError(errr).Errorf("Error while reading file")
			continue
		}

		ts := TestSuite{}
		ts.Package = f
		log.Debugf("Unmarshal %s", f)
		if err := yaml.Unmarshal(dat, &ts); err != nil {
			log.WithError(err).Errorf("Error while unmarshal file")
			continue
		}
		ts.Name += " [" + f + "]"

		nSteps := 0
		for _, tc := range ts.TestCases {
			nSteps += len(tc.TestSteps)
			if tc.Skipped == 1 {
				ts.Skipped++
			}
		}
		ts.Total = len(ts.TestCases)

		b := pb.New(nSteps).Prefix(rightPad("⚙ "+ts.Package, " ", 47))
		b.ShowCounters = false
		if detailsLevel == DetailsLow {
			b.ShowBar = false
			b.ShowFinalTime = false
			b.ShowPercent = false
			b.ShowSpeed = false
			b.ShowTimeLeft = false
		}

		if detailsLevel != DetailsLow {
			bars[ts.Package] = b
		}

		chanToRun <- ts
	}
	return bars
}
