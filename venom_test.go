package venom

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_createUserExecutorFromFile(t *testing.T) {
	executorPointer, err := createUserExecutorFromFile("tests/my-executors/testing_executor.yml")
	if err != nil {
		t.Fail()
	}
	ex := *executorPointer
	assert.Equal(t, "my-test", ex.Executor)
	assert.Equal(t, json.RawMessage(`{"all":"{{.output}}"}`), ex.Output)
	entries := H{}
	entries.Add("script", "")
	assert.Equal(t, entries, ex.Input)
	assert.Equal(t, 1, len(ex.RawTestSteps))
	assert.ObjectsAreEqualValues(ex.RawTestSteps, []json.RawMessage{json.RawMessage(`{"type":"exec","script":"{{ .input.script | nindent 4 }}","assertions":["result.code ShouldEqual 0"],"vars":{"output":{"from":"result.systemout"}`)})
}
