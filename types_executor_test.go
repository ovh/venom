/*
Copyright 2026 OVH SAS

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package venom

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// result is named so go-dump exposes the step output as result.systemout.
type result struct {
	Systemout string `json:"systemout"`
}

type captureExecutor struct{ out result }

func (e captureExecutor) Run(context.Context, TestStep) (interface{}, error) {
	return e.out, nil
}

func TestUserExecutorPreservesMultilineOutput(t *testing.T) {
	InitTestLogger(t)
	const systemout = "<hello>\n---\n"
	v := New()
	v.RegisterExecutorBuiltin("capture", captureExecutor{out: result{Systemout: systemout}})
	ux := UserExecutor{Executor: "captureOut", Raw: []byte(`
executor: captureOut
steps:
- type: capture
  vars:
    systemout:
      from: result.systemout
output:
  systemout: "{{ .systemout }}"
  upper: "{{ .systemout | upper }}"
  base64: "{{ .systemout | b64enc }}"
`)}
	runner := newExecutorRunner(ux, ux.Executor, "user", 0, nil, 0, 0, nil)
	tc := &TestCase{TestCaseInput: TestCaseInput{Name: "capture-case"}, TestSuiteVars: H{}}
	got, err := v.RunUserExecutor(context.Background(), runner, tc, &TestStepResult{}, TestStep{})
	require.NoError(t, err)
	flat := got.(map[string]interface{})
	require.Equal(t, systemout, flat["result.systemout"])
	require.Equal(t, strings.ToUpper(systemout), flat["result.upper"])
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte(systemout)), flat["result.base64"])
}
