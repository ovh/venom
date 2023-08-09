package venom

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestProcessVariableAssignments(t *testing.T) {
	InitTestLogger(t)
	assign := AssignStep{}
	assign.Assignments = make(map[string]Assignment)
	assign.Assignments["assignVar"] = Assignment{
		From: "here.some.value",
	}
	assign.Assignments["assignVarWithRegex"] = Assignment{
		From:  "here.some.value",
		Regex: `this is (?s:(.*))`,
	}

	b, _ := yaml.Marshal(assign)
	t.Log("\n" + string(b))

	tcVars := H{"here.some.value": "this is the \nvalue"}

	result, is, err := processVariableAssignments(context.TODO(), "", &tcVars, b)
	assert.True(t, is)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	t.Log(result)
	assert.Equal(t, "map[assignVar:this is the \nvalue assignVarWithRegex:the \nvalue]", fmt.Sprint(result))

	var wrongStepIn TestStep
	b = []byte(`type: exec
script: echo 'foo'
`)
	assert.NoError(t, yaml.Unmarshal(b, &wrongStepIn))
	result, is, err = processVariableAssignments(context.TODO(), "", &tcVars, b)
	assert.False(t, is)
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.Empty(t, result)
}

func TestProcessJsonBlobWithObject(t *testing.T) {
	InitTestLogger(t)
	items, err := processJsonBlob("test", "{\"key\":123,\"another\":\"one\"}")
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Contains(t, items, "testjson.key")
	assert.Contains(t, items, "testjson.another")
	assert.Contains(t, items, "testjson")
	assert.Contains(t, items, "__Type__")
	assert.Contains(t, items, "__Len__")
}

func TestProcessJsonBlobWithArray(t *testing.T) {
	InitTestLogger(t)
	items, err := processJsonBlob("test", "{\"key\":123,\"anArray\":[\"one\",\"two\"]}")
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Contains(t, items, "testjson.key")
	assert.Contains(t, items, "testjson.anArray")
	assert.Contains(t, items, "testjson")
	assert.Equal(t, items["anArray.__Type__"], "Array")
	assert.Equal(t, items["anArray.__Len__"], "2")
	assert.Contains(t, items, "testjson.anArray.anArray0")
	assert.Equal(t, items["testjson.anArray.anArray0"], "one")
	assert.Contains(t, items, "testjson.anArray.anArray1")
	assert.Equal(t, items["testjson.anArray.anArray1"], "two")
	assert.Contains(t, items, "__Type__")
	assert.Equal(t, items["__Type__"], "Map")
	assert.Contains(t, items, "__Len__")
}

func TestGetKeyForLookup(t *testing.T) {
	InitTestLogger(t)
	assert.Equal(t, "test", getKeyForLookup("testjson.key"))
	assert.Equal(t, "test", getKeyForLookup("testjson.anArray.anArray0"))
}

func TestProcessRange(t *testing.T) {
	InitTestLogger(t)
	rawStep := []byte("{\"account_id\":\"{{.value}}\",\"name\":\"Account validation\",\"range\":\"{{.account_ids}}\",\"type\":\"account_validation\"}")
	vars := H{}
	vars.Add("account_ids", []string{`1`, `2`})
	ranged, err := parseRanged(context.Background(), rawStep, vars)
	assert.NoError(t, err)
	assert.NotNil(t, ranged)
	assert.NotNil(t, ranged.Items)
	assert.Nil(t, ranged.RawContent)
	assert.True(t, ranged.Enabled)
	assert.Equal(t, 2, len(ranged.Items))
}
