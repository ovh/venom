package venom

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDump(t *testing.T) {
	v := map[string]interface{}{}
	v["result.systemoutjson"] = map[string]string{"FOO": "bar", "foo": "foo"}
	v["result.systemoutjson.FOO"] = "bar"
	v["result.systemoutjson.foo"] = "foo"
	got, err := Dump(v)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "foo", got["result.systemoutjson.foo"])
	require.Equal(t, "bar", got["result.systemoutjson.FOO"])
}
