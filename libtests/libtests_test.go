package main

import (
	"testing"

	"github.com/runabove/venom/lib"
	"github.com/stretchr/testify/assert"
)

func TestVenomTestCase(t *testing.T) {
	v := venom.NewTestCase(t, "TestVenomTestCase", venom.V{})
	r := venom.RunTest(v, venom.H{
		"type":   "exec",
		"script": "echo foo",
	})
	assert.Equal(t, "foo", r["result.systemout"])
}
