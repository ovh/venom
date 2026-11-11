package venom

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWaitDuration(t *testing.T) {
	cases := []struct {
		in   interface{}
		want time.Duration
	}{
		{nil, 0},
		{0, 0},
		{2, 2 * time.Second},
		{int64(3), 3 * time.Second},
		{uint(4), 4 * time.Second},
		{0.2, 200 * time.Millisecond},
		{1.5, 1500 * time.Millisecond},
		{float32(0.5), 500 * time.Millisecond},
		{"", 0},
		{"7", 7 * time.Second},
		{"0.25", 250 * time.Millisecond},
		{"200ms", 200 * time.Millisecond},
		{"1m", time.Minute},
		{" 1.5s ", 1500 * time.Millisecond},
	}
	for _, c := range cases {
		got, err := ParseWaitDuration(c.in)
		require.NoError(t, err, "%#v", c.in)
		assert.Equal(t, c.want, got, "%#v", c.in)
	}
	for _, bad := range []interface{}{"soon", "10 parsecs", []string{"1"}} {
		_, err := ParseWaitDuration(bad)
		assert.Error(t, err, "%#v", bad)
	}
}

func TestExecutorRunnerWaitFields(t *testing.T) {
	v := New()
	cases := []struct {
		step                   TestStep
		delay, timeout         time.Duration
		delaySecs, timeoutSecs int
	}{
		{TestStep{"delay": 2, "timeout": 10}, 2 * time.Second, 10 * time.Second, 2, 10},
		{TestStep{"delay": 0.2, "timeout": "1500ms"}, 200 * time.Millisecond, 1500 * time.Millisecond, 0, 1},
		{TestStep{}, 0, 0, 0, 0},
	}
	for _, c := range cases {
		_, e, err := v.GetExecutorRunner(context.Background(), c.step, H{})
		require.NoError(t, err)
		assert.Equal(t, c.delay, e.DelayDuration())
		assert.Equal(t, c.timeout, e.TimeoutDuration())
		assert.Equal(t, c.delaySecs, e.Delay())
		assert.Equal(t, c.timeoutSecs, e.Timeout())
	}
	_, _, err := v.GetExecutorRunner(context.Background(), TestStep{"delay": "later"}, H{})
	assert.Error(t, err)
}

func TestFormatWait(t *testing.T) {
	assert.Equal(t, "3 second(s)", formatWait(3*time.Second))
	assert.Equal(t, "200ms", formatWait(200*time.Millisecond))
}
