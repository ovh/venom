package venom

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// ParseWaitDuration reads a per-step wait attribute (delay, timeout).
// An integer, or a string holding one, means whole seconds, as before.
// A fractional number means seconds (0.2 is 200ms).
// A Go duration string ("200ms", "1.5s", "1m") is taken as is.
// A missing or empty value is zero.
func ParseWaitDuration(value interface{}) (time.Duration, error) {
	switch x := value.(type) {
	case nil:
		return 0, nil
	case time.Duration:
		return x, nil
	case float32:
		return secondsToDuration(float64(x))
	case float64:
		return secondsToDuration(x)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, nil
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Duration(i) * time.Second, nil
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return secondsToDuration(f)
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("%q is neither a number of seconds nor a duration", x)
		}
		return d, nil
	}
	i, err := cast.ToInt64E(value)
	if err != nil {
		return 0, err
	}
	return time.Duration(i) * time.Second, nil
}

func secondsToDuration(f float64) (time.Duration, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%v is not a number of seconds", f)
	}
	return time.Duration(math.Round(f * float64(time.Second))), nil
}

// DurationValue reads a wait attribute of a step, see ParseWaitDuration.
func (t TestStep) DurationValue(name string) (time.Duration, error) {
	d, err := ParseWaitDuration(t[name])
	if err != nil {
		return 0, fmt.Errorf("attribute %q is not a number of seconds or a duration: %v", name, err)
	}
	return d, nil
}

// formatWait prints a whole number of seconds as before ("5 second(s)"), and as a duration otherwise.
func formatWait(d time.Duration) string {
	if d%time.Second == 0 {
		return fmt.Sprintf("%d second(s)", int64(d/time.Second))
	}
	return d.String()
}
