package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/venom"
)

func Test_buildVariableSet(t *testing.T) {
	os.Setenv("VENOM_VAR_a", "value from A env")
	os.Setenv("VENOM_VAR_b", "value from B env")
	flags := []string{
		"b=value from B flag",
		"c=value from C flag",
	}
	varsInFile := venom.H{
		"c": "value from C file",
		"d": "value from D file",
	}
	varsBtes, _ := json.Marshal(varsInFile)

	f, err := ioutil.TempFile("", "*.json")
	assert.NoError(t, err)
	ioutil.WriteFile(f.Name(), varsBtes, os.FileMode(0644))
	f.Close()

	variables, err := buildVariableSet(flags, []string{f.Name()}, true)
	assert.NoError(t, err)

	assert.Equal(t, "value from A env", variables["a"])
	assert.Equal(t, "value from B flag", variables["b"])
	assert.Equal(t, "value from C flag", variables["c"])
	assert.Equal(t, "value from D file", variables["d"])
	assert.Len(t, variables, 4)
}
