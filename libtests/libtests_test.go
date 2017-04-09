package main

import (
	"testing"

	"github.com/runabove/venom/lib"
	"github.com/stretchr/testify/assert"
)

func TestVenomTestCase(t *testing.T) {
	v := venom.TestCase(t, "TestVenomTestCase", venom.V{
		"foo": "bar",
	})
	r := v.Do(venom.H{
		"type":   "exec",
		"script": "echo {{.foo}}",
	})
	assert.Equal(t, "bar", r["result.systemout"])
}
