package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/venom"
)

func buildVariableSet(flags []string, files []string, fromEnv bool) (venom.H, error) {
	var variables = venom.H{}
	if fromEnv {
		variables.AddAll(buildVariableSetFromEnv())
	}
	v, err := buildVariableSetFromFiles(files)
	if err != nil {
		return nil, err
	}
	variables.AddAll(v)

	variables.AddAll(buildVariableSetFromFlags(flags))
	return variables, nil
}

func buildVariableSetFromFlags(flags []string) venom.H {
	var variables = venom.H{}
	for _, flag := range flags {
		tuple := strings.SplitN(flag, "=", 2)
		k := strings.TrimSpace(tuple[0])
		v := strings.TrimSpace(tuple[1])
		variables.Add(k, v)
	}
	return variables
}

func buildVariableSetFromFiles(files []string) (venom.H, error) {
	var variables = venom.H{}
	for _, f := range files {
		v, err := buildVariableSetFromFile(f)
		if err != nil {
			return nil, fmt.Errorf("unable to load variables from %s", f)
		}
		variables.AddAll(v)
	}
	return variables, nil
}

func buildVariableSetFromFile(file string) (venom.H, error) {
	var ext = filepath.Ext(file)
	btes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var variables = venom.H{}
	switch ext {
	case ".json":
		if err := json.Unmarshal(btes, &variables); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(btes, &variables); err != nil {
			return nil, err
		}
	case ".hcl":
		if err := hcl.Unmarshal(btes, &variables); err != nil {
			return nil, err
		}
	}
	return variables, nil
}

func buildVariableSetFromEnv() venom.H {
	var variables = venom.H{}
	environ := os.Environ()
	for _, env := range environ {
		tuple := strings.SplitN(env, "=", 2)
		k := strings.TrimSpace(tuple[0])
		v := strings.TrimSpace(tuple[1])
		if strings.HasPrefix(k, "VENOM_VAR_") {
			k = strings.TrimPrefix(k, "VENOM_VAR_")
			variables.Add(k, v)
		}
	}
	return variables
}
