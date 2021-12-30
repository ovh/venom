package venom_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors/amqp"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/grpc"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/kafka"
	"github.com/ovh/venom/executors/mqtt"
	"github.com/ovh/venom/executors/ovhapi"
	"github.com/ovh/venom/executors/rabbitmq"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/redis"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/sql"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGherkinVenom(t *testing.T) *venom.GherkinVenom {
	venom.InitTestLogger(t)
	v := venom.NewGherkin()
	v.Verbose = 0
	v.RegisterExecutorBuiltin(amqp.Name, amqp.New())
	v.RegisterExecutorBuiltin(dbfixtures.Name, dbfixtures.New())
	v.RegisterExecutorBuiltin(exec.Name, exec.New())
	v.RegisterExecutorBuiltin(grpc.Name, grpc.New())
	v.RegisterExecutorBuiltin(http.Name, http.New())
	v.RegisterExecutorBuiltin(imap.Name, imap.New())
	v.RegisterExecutorBuiltin(kafka.Name, kafka.New())
	v.RegisterExecutorBuiltin(mqtt.Name, mqtt.New())
	v.RegisterExecutorBuiltin(ovhapi.Name, ovhapi.New())
	v.RegisterExecutorBuiltin(rabbitmq.Name, rabbitmq.New())
	v.RegisterExecutorBuiltin(readfile.Name, readfile.New())
	v.RegisterExecutorBuiltin(redis.Name, redis.New())
	v.RegisterExecutorBuiltin(smtp.Name, smtp.New())
	v.RegisterExecutorBuiltin(sql.Name, sql.New())
	v.RegisterExecutorBuiltin(ssh.Name, ssh.New())
	v.RegisterExecutorBuiltin(web.Name, web.New())

	return v
}

func TestTransformExecutorToMap(t *testing.T) {
	v := setupGherkinVenom(t)

	for name, executor := range v.Executors() {
		ge, is := executor.(venom.ExecutorWithGherkinSupport)
		if !is {
			t.Logf("%T is not a ExecutorWithGherkinSupport", executor)
			continue
		}
		m := v.TransformExecutorToMap(ge)
		t.Logf("%s => %+v", name, m)
	}
}

func TestFindSuitableExecutor_HTTP(t *testing.T) {
	var gStep = venom.GherkinStep{
		Keywork: "When",
		Text:    "HTTP Get https://eu.api.ovh.com/1.0/",
	}

	v := setupGherkinVenom(t)

	step, err := v.FindSuitableExecutor(gStep)
	require.NoError(t, err)
	assert.Len(t, step, 2)
	assert.Equal(t, "Get", step["method"])
	assert.Equal(t, "https://eu.api.ovh.com/1.0/", step["url"])
}

func TestFindSuitableExecutor_HTTP_Complex(t *testing.T) {
	var gStep = venom.GherkinStep{
		Keywork: "When",
		Text: `HTTP POST https://jsonplaceholder.typicode.com/posts
		With foo bar biz as body
		And {"Content-Type": "application/json"} as headers`,
	}

	v := setupGherkinVenom(t)

	step, err := v.FindSuitableExecutor(gStep)
	require.NoError(t, err)

	assert.Equal(t, "POST", step["method"])
	assert.Equal(t, "https://jsonplaceholder.typicode.com/posts", step["url"])
	assert.Equal(t, "foo bar biz", step["body"])
	assert.Equal(t, "map[Content-Type:application/json]", fmt.Sprint(step["headers"]))
}

func TestFindSuitableExecutor_HTTP_with_retry(t *testing.T) {
	var gStep = venom.GherkinStep{
		Keywork: "When",
		Text:    "Try 3 times every 10 seconds HTTP Get https://eu.api.ovh.com/1.0/",
	}

	v := setupGherkinVenom(t)

	step, err := v.FindSuitableExecutor(gStep)
	require.NoError(t, err)

	t.Logf("%+v", step)

	assert.Equal(t, "Get", step["method"])
	assert.Equal(t, "https://eu.api.ovh.com/1.0/", step["url"])
	assert.Equal(t, 3, step["retry"])
	assert.Equal(t, 10, step["delay"])
}

func TestTransformGherkinStepToAssertion(t *testing.T) {
	v := setupGherkinVenom(t)

	var gStep = venom.GherkinStep{
		Keywork: "Then",
		Text:    "result.statuscode MustEqual 200",
	}

	assert, err := v.TransformGherkinStepToAssertion(gStep)
	require.NoError(t, err)
	t.Logf("%+v", assert)
}

func TestHandleGherkinScenario(t *testing.T) {
	v := setupGherkinVenom(t)

	gScenario := venom.GherkinScenario{
		Text: "this is a scenario",
		Steps: []venom.GherkinStep{
			{
				Keywork: "When",
				Text:    "HTTP Get https://eu.api.ovh.com/1.0/",
			},
			{
				Keywork: "Then",
				Text:    "result.statuscode MustEqual 200",
			},
			{
				Keywork: "Then",
				Text:    "result.body ShouldNotBeEmpty",
			},
			{
				Keywork: "And",
				Text:    "HTTP Get https://eu.api.ovh.com/1.0/",
			},
			{
				Keywork: "Then",
				Text:    "result.statuscode MustEqual 200",
			},
		},
	}

	testcase, err := v.HandleGherkinScenario(gScenario)
	require.NoError(t, err)

	t.Logf("%+v", testcase)

	require.Len(t, testcase.RawTestSteps, 2)
}

func TestParseGherkin(t *testing.T) {
	v := setupGherkinVenom(t)

	featureContent := `
	Feature: HTTP Gherkin test suite

	Scenario: HTTP Get test
		When    HTTP Get https://eu.api.ovh.com/1.0/
		Then    result.body ShouldContainSubstring /dedicated/server
		And     result.body ShouldContainSubstring /ipLoadbalancing
		And     result.statuscode ShouldEqual 200
		And     result.bodyjson.apis.apis0.path ShouldEqual /allDom
	
	Scenario: HTTP Get test 2
		When    HTTP Get https://eu.api.ovh.com/1.0/
		Then    result.body ShouldContainSubstring /dedicated/server
		And     result.body ShouldContainSubstring /ipLoadbalancing
		And     result.statuscode ShouldEqual 200
		And     result.bodyjson.apis.apis0.path ShouldEqual /allDom
	`
	tmpdir := os.TempDir()
	tmpfile, err := os.CreateTemp(tmpdir, "*.feature")
	require.NoError(t, err)

	tmpfileName := tmpfile.Name()

	_, err = tmpfile.WriteString(featureContent)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	err = v.ParseGherkin(context.TODO(), []string{tmpfileName})
	require.NoError(t, err)

	t.Logf("\n" + v.GetFeaturesString())
	expectedFeaturesString := `Feature: HTTP Gherkin test suite
Scenario: HTTP Get test
	When HTTP Get https://eu.api.ovh.com/1.0/
	Then result.body ShouldContainSubstring /dedicated/server
	And result.body ShouldContainSubstring /ipLoadbalancing
	And result.statuscode ShouldEqual 200
	And result.bodyjson.apis.apis0.path ShouldEqual /allDom
Scenario: HTTP Get test 2
	When HTTP Get https://eu.api.ovh.com/1.0/
	Then result.body ShouldContainSubstring /dedicated/server
	And result.body ShouldContainSubstring /ipLoadbalancing
	And result.statuscode ShouldEqual 200
	And result.bodyjson.apis.apis0.path ShouldEqual /allDom`

	assert.Contains(t, v.GetFeaturesString(), expectedFeaturesString)

	t.Logf("\n" + v.GetTestSuitesString())
	expectedTestsuitesString := `name: HTTP Gherkin test suite
testcases:
- name: HTTP Get test
  steps:
  - assertions:
    - result.body ShouldContainSubstring /dedicated/server
    - result.body ShouldContainSubstring /ipLoadbalancing
    - result.statuscode ShouldEqual 200
    - result.bodyjson.apis.apis0.path ShouldEqual /allDom
    method: Get
    type: http
    url: https://eu.api.ovh.com/1.0/
- name: HTTP Get test 2
  steps:
  - assertions:
    - result.body ShouldContainSubstring /dedicated/server
    - result.body ShouldContainSubstring /ipLoadbalancing
    - result.statuscode ShouldEqual 200
    - result.bodyjson.apis.apis0.path ShouldEqual /allDom
    method: Get
    type: http
    url: https://eu.api.ovh.com/1.0/`

	assert.Contains(t, v.GetTestSuitesString(), expectedTestsuitesString)
}
