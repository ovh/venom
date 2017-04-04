package main

import (
	"testing"

	"github.com/runabove/venom/lib"
)

func TestVenomTestCase(t *testing.T) {
	v := venom.NewTestCase(t, "TestVenomTestCase", venom.V{})
	venom.RunTest(v, venom.H{
		"type":   "exec",
		"script": "echo foo",
	})
}
