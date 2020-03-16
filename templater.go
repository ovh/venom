package venom

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Define a type to venom function
type VenomFunction func() string

// Templater contains templating values on a testsuite
type Templater struct {
	Values    map[string]string
	Functions map[string]VenomFunction
}

func newTemplater(inputValues map[string]string) *Templater {
	// Copy map to be thread safe with parallel > 1
	values := make(map[string]string)
	for key, value := range inputValues {
		values[key] = value
	}
	return &Templater{Values: values}
}

// Add add data to templater
func (tmpl *Templater) Add(prefix string, values map[string]string) {
	if tmpl.Values == nil {
		tmpl.Values = make(map[string]string)
	}
	dot := ""
	if prefix != "" {
		dot = "."
	}
	for k, v := range values {
		tmpl.Values[prefix+dot+k] = v
	}
}

// Add a function to templater
func (tmpl *Templater) AddFunction(name string, function VenomFunction) {
	if tmpl.Functions == nil {
		tmpl.Functions = make(map[string]VenomFunction)
	}
	tmpl.Functions[name] = function
}

//ApplyOnStep executes the template on a test step
func (tmpl *Templater) ApplyOnStep(stepNumber int, step TestStep) (TestStep, error) {
	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := s
	// if the testTest use some variable, we run tmpl.apply on it
	if strings.Contains(string(s), "{{") {
		if stepNumber >= 0 {
			tmpl.Add("", map[string]string{"venom.teststep.number": fmt.Sprintf("%d", stepNumber)})
		}
		_, sb = tmpl.apply(s)
	}

	// Apply functions
	body := string(sb)
	for k, v := range tmpl.Functions {
		functionName := k + "()"
		body = strings.Replace(body, functionName, v(), -1)
	}
	sb = []byte(body)

	var t TestStep
	if err := yaml.Unmarshal([]byte(sb), &t); err != nil {
		return nil, fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}

	return t, nil
}

// ApplyOnMap executes the template on a context
// return true if there is an variable replaced
func (tmpl *Templater) ApplyOnMap(mapStringInterface map[string]interface{}) (bool, map[string]interface{}, error) {
	var t map[string]interface{}
	if len(mapStringInterface) == 0 {
		return false, t, nil
	}

	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(mapStringInterface)
	if err != nil {
		return false, nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := s
	var applied bool
	// if the mapStringInterface use some variable, we run tmpl.apply on it
	if strings.Contains(string(s), "{{") {
		applied, sb = tmpl.apply(s)
	}

	// Apply functions
	body := string(sb)
	for k, v := range tmpl.Functions {
		functionName := k + "()"
		body = strings.Replace(body, functionName, v(), -1)
	}
	sb = []byte(body)

	if err := yaml.Unmarshal([]byte(sb), &t); err != nil {
		return applied, nil, fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}

	return applied, t, nil
}

var expandEnvRegEx = regexp.MustCompile("{{expandEnv (.*)}}")

func (tmpl *Templater) apply(in []byte) (bool, []byte) {
	out := string(in)

	if expandEnvRegEx.MatchString(out) {
		capture := expandEnvRegEx.FindAllStringSubmatch(out, -1)[0]
		if len(capture) > 0 {
			filename := capture[1]
			fileStat, _ := os.Stat(filename)
			fileContent, _ := ioutil.ReadFile(filename)
			if len(fileContent) != 0 {
				newFileContent := os.ExpandEnv(string(fileContent))
				err := ioutil.WriteFile(filename, []byte(newFileContent), fileStat.Mode().Perm())
				if err == nil {
					out = strings.Replace(out, capture[0], filename, -1)
				}
			}
		}
	}

	tmpl.Add("", map[string]string{
		"venom.datetime":  time.Now().Format(time.RFC3339),
		"venom.timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	})
	var applied bool
	for k, v := range tmpl.Values {
		applied = true
		var buffer bytes.Buffer
		buffer.WriteString("{{.")
		buffer.WriteString(k)
		buffer.WriteString("}}")
		out = strings.Replace(out, buffer.String(), v, -1)
		// if no more variable to replace, exit
		if !strings.Contains(out, "{{") {
			return applied, []byte(out)
		}
	}
	return applied, []byte(out)
}
