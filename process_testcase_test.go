package venom

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestProcessVariableAssigments(t *testing.T) {
	InitTestLogger(t)
	assign := AssignStep{}
	assign.Assignments = make(map[string]Assignment)
	assign.Assignments["assignVar"] = Assignment{
		From: "here.some.value",
	}
	assign.Assignments["assignVarWithRegex"] = Assignment{
		From:    "here.some.value",
		Regex:   `this is (?s:(.*))`,
		Unalter: true,
	}

	b, _ := yaml.Marshal(assign)
	t.Log("\n" + string(b))

	tcVars := H{"here.some.value": "this is the \nvalue"}

	result, unalteredResult, is, err := processVariableAssigments(context.TODO(), "", tcVars, b)
	assert.True(t, is)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, unalteredResult)
	t.Log(result)
	assert.Equal(t, "map[assignVar:this is the \nvalue]", fmt.Sprint(result))
	assert.Equal(t, "map[assignVarWithRegex:the \nvalue]", fmt.Sprint(unalteredResult))

	var wrongStepIn TestStep
	b = []byte(`type: exec
script: echo 'foo'
`)
	assert.NoError(t, yaml.Unmarshal(b, &wrongStepIn))
	result, unalteredResult, is, err = processVariableAssigments(context.TODO(), "", tcVars, b)
	assert.False(t, is)
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.Empty(t, result)
	assert.Nil(t, unalteredResult)
	assert.Empty(t, unalteredResult)
}
