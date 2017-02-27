package venom

import (
	"fmt"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
)

// applyExtracts try to run extract on step, return true if all extracts are OK, false otherwise
func applyExtracts(executorResult *ExecutorResult, step TestStep, l *log.Entry) (bool, []Failure, []Failure) {

	var se StepExtracts
	var errors []Failure
	var failures []Failure

	if err := mapstructure.Decode(step, &se); err != nil {
		return false, []Failure{{Value: fmt.Sprintf("error decoding extracts: %s", err)}}, failures
	}

	isOK := true
	for key, pattern := range se.Extracts {
		e := *executorResult
		if _, ok := e[key]; !ok {
			return false, []Failure{{Value: fmt.Sprintf("key %s in result is not found", key)}}, failures
		}
		errs, fails := checkExtracts(pattern, e[key], executorResult, l)
		if errs != nil {
			errors = append(errors, *errs)
			isOK = false
		}
		if fails != nil {
			failures = append(failures, *fails)
			isOK = false
		}
	}

	return isOK, errors, failures
}

func checkExtracts(pattern, instring string, executorResult *ExecutorResult, l *log.Entry) (*Failure, *Failure) {
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(instring)
	if match == nil {
		return &Failure{Value: fmt.Sprintf("Pattern '%s' does not match string '%s'", pattern, instring)}, nil
	}

	e := *executorResult
	found := true
	for i, name := range r.SubexpNames() {
		if i == 0 {
			continue
		}
		e[name] = match[i]
	}

	if !found {
		return nil, &Failure{Value: fmt.Sprintf("pattern '%s' match nothing in result '%s'", pattern, instring)}
	}
	return nil, nil
}
