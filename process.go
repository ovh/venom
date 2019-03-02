package venom

import (
	"sync"

	"github.com/sirupsen/logrus"
)

func (v *Venom) init() error {
	v.testsuites = []TestSuite{}
	if v.Parallel == 0 {
		v.Parallel = 1
	}

	if v.logger == nil {
		v.logger = logrus.New()
	}

	switch v.LogLevel {
	case "debug":
		v.logger.SetLevel(logrus.DebugLevel)
	case "info":
		v.logger.SetLevel(logrus.InfoLevel)
	case "error":
		v.logger.SetLevel(logrus.ErrorLevel)
	default:
		v.logger.SetLevel(logrus.WarnLevel)
	}

	v.logger.SetOutput(v.LogOutput)

	return nil
}

// Process runs tests suite and return a Tests result
func (v *Venom) Process(path []string) (*Tests, error) {
	if err := v.init(); err != nil {
		return nil, err
	}

	filesPath, err := getFilesPath(path)
	if err != nil {
		return nil, err
	}

	if err := v.readFiles(filesPath); err != nil {
		return nil, err
	}

	chanEnd := make(chan *TestSuite, 1)
	parallels := make(chan *TestSuite, v.Parallel) //Run testsuite in parrallel
	wg := sync.WaitGroup{}
	testsResult := &Tests{}

	wg.Add(len(filesPath))
	chanToRun := make(chan *TestSuite, len(filesPath)+1)

	go v.computeStats(testsResult, chanEnd, &wg)
	go func() {
		for ts := range chanToRun {
			parallels <- ts
			go func(ts *TestSuite) {
				v.runTestSuite(ts)
				chanEnd <- ts
				<-parallels
			}(ts)
		}
	}()

	for i := range v.testsuites {
		chanToRun <- &v.testsuites[i]
	}

	wg.Wait()

	return testsResult, nil
}

func (v *Venom) computeStats(testsResult *Tests, chanEnd <-chan *TestSuite, wg *sync.WaitGroup) {
	for t := range chanEnd {
		testsResult.TestSuites = append(testsResult.TestSuites, *t)
		if t.Failures > 0 || t.Errors > 0 {
			testsResult.TotalKO += (t.Failures + t.Errors)
		} else {
			testsResult.TotalOK += len(t.TestCases) - (t.Failures + t.Errors)
		}
		if t.Skipped > 0 {
			testsResult.TotalSkipped += t.Skipped
		}

		testsResult.Total = testsResult.TotalKO + testsResult.TotalOK + testsResult.TotalSkipped
		wg.Done()
	}
}
