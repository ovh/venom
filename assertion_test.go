package venom

import (
	"context"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_splitAssertion(t *testing.T) {
	for _, tt := range []struct {
		Assertion string
		Args      []string
	}{
		{Assertion: `cmd arg`, Args: []string{"cmd", "arg"}},
		{Assertion: `cmd arg1 "arg 2"`, Args: []string{"cmd", "arg1", "arg 2"}},
		{Assertion: `cmd 'arg 1' "arg 2"`, Args: []string{"cmd", "arg 1", "arg 2"}},
		{Assertion: `cmd 'arg 1' "'arg' 2"`, Args: []string{"cmd", "arg 1", "'arg' 2"}},
		{Assertion: `cmd '"arg 1"' "'arg' 2"`, Args: []string{"cmd", "\"arg 1\"", "'arg' 2"}},
	} {
		args := splitAssertion(tt.Assertion)
		if !reflect.DeepEqual(args, tt.Args) {
			t.Errorf("expected args to be equal to %#v, got %#v", tt.Args, args)
		}
	}
}

func TestCheckBranchWithOR(t *testing.T) {
	tc := TestCase{}
	vars := map[string]interface{}{}
	vars["result.statuscode"] = 501
	vars["is_feature_supported"] = "false"
	branch := map[string]interface{}{}

	firstSetOfAssertions := []interface{}{`is_feature_supported ShouldEqual true`, `result.statuscode ShouldEqual 200`}
	secondSetOfAssertions := []interface{}{`is_feature_supported ShouldEqual false`, `result.statuscode ShouldEqual 501`}

	branch["or"] = []interface{}{
		map[string]interface{}{"and": firstSetOfAssertions},
		map[string]interface{}{"and": secondSetOfAssertions},
	}

	failure := checkBranch(context.Background(), tc, 0, 0, branch, vars)
	assert.Nil(t, failure)
}
func TestCheckBranchWithORFailing(t *testing.T) {
	tc := TestCase{}
	vars := map[string]interface{}{}
	vars["result.statuscode"] = 400
	vars["is_feature_supported"] = "false"
	branch := map[string]interface{}{}

	firstSetOfAssertions := []interface{}{`result.statuscode ShouldEqual 200`}
	secondSetOfAssertions := []interface{}{`result.statuscode ShouldEqual 501`}

	branch["or"] = []interface{}{
		map[string]interface{}{"and": firstSetOfAssertions},
		map[string]interface{}{"and": secondSetOfAssertions},
	}

	failure := checkBranch(context.Background(), tc, 0, 0, branch, vars)
	assert.NotNil(t, failure)
}
