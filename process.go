package venom

import (
	"context"
	"fmt"
	"sync"
)

// Process runs tests suite and return a Tests result
func (v *Venom) Process(ctx context.Context, path []string) (*Tests, error) {
	if err := v.init(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go v.Display.Display(ctx)

	if len(path) == 0 {
		return nil, fmt.Errorf("nothing to do")
	}

	v.GetLogger().Debugf("Starting venom...")

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
				tsLogger := LoggerWithField(v.logger, "testsuite", ts.ShortName)
				v.runTestSuite(ctx, ts, tsLogger)
				chanEnd <- ts
				<-parallels
			}(ts)
		}
	}()

	for i := range v.testsuites {
		chanToRun <- &v.testsuites[i]
	}

	wg.Wait()
	fmt.Println()
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
