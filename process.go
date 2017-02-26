package venom

import (
	"strings"
	"sync"
)

// Process runs tests suite and return a Tests result
func Process(path []string, alias []string, exclude []string, parallel int, detailsLevel string) (Tests, error) {

	chanEnd := make(chan TestSuite, 1)
	parallels := make(chan TestSuite, parallel)
	wg := sync.WaitGroup{}
	testsResult := Tests{}

	aliases := computeAliases(alias)

	filesPath := getFilesPath(path, exclude)
	wg.Add(len(filesPath))
	chanToRun := make(chan TestSuite, len(filesPath)+1)

	go computeStats(&testsResult, chanEnd, &wg)

	bars := readFiles(detailsLevel, filesPath, chanToRun)

	pool := initBars(detailsLevel, bars)

	go func() {
		for ts := range chanToRun {
			parallels <- ts
			go func(ts TestSuite) {
				runTestSuite(&ts, bars, detailsLevel, aliases)
				chanEnd <- ts
				<-parallels
			}(ts)
		}
	}()

	wg.Wait()

	endBars(detailsLevel, pool)

	return testsResult, nil
}

func computeAliases(alias []string) map[string]string {
	aliases := make(map[string]string)
	for _, a := range alias {
		t := strings.Split(a, ":")
		if len(t) < 2 {
			continue
		}
		aliases[t[0]] = strings.Join(t[1:], "")
	}
	return aliases
}

func computeStats(testsResult *Tests, chanEnd <-chan TestSuite, wg *sync.WaitGroup) {
	for t := range chanEnd {
		testsResult.TestSuites = append(testsResult.TestSuites, t)
		if t.Failures > 0 {
			testsResult.TotalKO += t.Failures
		} else {
			testsResult.TotalOK += len(t.TestCases) - t.Failures
		}
		if t.Skipped > 0 {
			testsResult.TotalSkipped += t.Skipped
		}

		testsResult.Total = testsResult.TotalKO + testsResult.TotalOK + testsResult.TotalSkipped
		wg.Done()
	}
}

func rightPad(s string, padStr string, pLen int) string {
	o := s + strings.Repeat(padStr, pLen)
	return o[0:pLen]
}
