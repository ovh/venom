package assertions

import (
	"testing"
	"time"
)

func TestShouldEqual(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `a`,
				expected: []interface{}{`a`},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   1,
				expected: []interface{}{1},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   1.0,
				expected: []interface{}{1.0},
			},
		},
		{
			name: "different types",
			args: args{
				actual:   42,
				expected: []interface{}{"42"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldEqual(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldEqual() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotEqual(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `a`,
				expected: []interface{}{`b`},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   1,
				expected: []interface{}{2},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   1.0,
				expected: []interface{}{2.0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotEqual(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotEqual() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldAlmostEqual(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `a`,
				expected: []interface{}{`b`},
			},
			wantErr: true,
		},
		{
			name: "with int",
			args: args{
				actual:   10,
				expected: []interface{}{9, 2},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   1.1,
				expected: []interface{}{1.2, 0.1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldAlmostEqual(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("TestShouldAlmostEqual() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotAlmostEqual(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `a`,
				expected: []interface{}{`b`},
			},
			wantErr: true,
		},
		{
			name: "with int",
			args: args{
				actual:   10,
				expected: []interface{}{5, 2},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   1.1,
				expected: []interface{}{1.5, 0.1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotAlmostEqual(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotAlmostEqual() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeTrue(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual: `a`,
			},
			wantErr: true,
		},
		{
			name: "with args",
			args: args{
				actual:   1,
				expected: []interface{}{1},
			},
			wantErr: true,
		},
		{
			name: "with bool",
			args: args{
				actual: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeTrue(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeTrue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeFalse(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual: `a`,
			},
			wantErr: true,
		},
		{
			name: "with args",
			args: args{
				actual:   1,
				expected: []interface{}{1},
			},
			wantErr: true,
		},
		{
			name: "with bool",
			args: args{
				actual: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeFalse(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeFalse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeNil(t *testing.T) {
	var m map[string]string
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual: `a`,
			},
			wantErr: true,
		},
		{
			name: "with int",
			args: args{
				actual: 1,
			},
			wantErr: true,
		},
		{
			name: "with nothing",
		},
		{
			name: "with a nil map",
			args: args{
				actual: m,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeNil(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeNil() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotBeNil(t *testing.T) {
	var m map[string]string
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual: `a`,
			},
		},
		{
			name: "with int",
			args: args{
				actual: 1,
			},
		},
		{
			name:    "with nothing",
			wantErr: true,
		},
		{
			name: "with a nil map",
			args: args{
				actual: m,
			},
			wantErr: true,
		},
		{
			name: "with an empty slice",
			args: args{
				actual: []string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotBeNil(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotBeNil() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeZeroValue(t *testing.T) {
	var m map[string]string
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual: ``,
			},
		},
		{
			name: "with int",
			args: args{
				actual: 0,
			},
		},
		{
			name: "with nothing",
		},
		{
			name: "with a nil map",
			args: args{
				actual: m,
			},
		},
		{
			name: "with an empty slice",
			args: args{
				actual: []string{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeZeroValue(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeZeroValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeGreaterThan(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `b`,
				expected: []interface{}{"a"},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   2,
				expected: []interface{}{1},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   2.0,
				expected: []interface{}{1.0},
			},
		},
		{
			name: "with wrong types",
			args: args{
				actual:   2.0,
				expected: []interface{}{"a"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeGreaterThan(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeGreaterThan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeGreaterThanOrEqualTo(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `a`,
				expected: []interface{}{"a"},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   2,
				expected: []interface{}{2},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   2.0,
				expected: []interface{}{2.0},
			},
		},
		{
			name: "with string",
			args: args{
				actual:   `b`,
				expected: []interface{}{"a"},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   2,
				expected: []interface{}{1},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   2.0,
				expected: []interface{}{1.0},
			},
		},
		{
			name: "with wrong types",
			args: args{
				actual:   2.0,
				expected: []interface{}{"a"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeGreaterThanOrEqualTo(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeGreaterThanOrEqualTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeBetween(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `b`,
				expected: []interface{}{"a", "c"},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   2,
				expected: []interface{}{1, 3},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   2.0,
				expected: []interface{}{1.0, 3.0},
			},
		},
		{
			name: "with wrong types",
			args: args{
				actual:   2.0,
				expected: []interface{}{"a", 3},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeBetween(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeBetween() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotBeBetween(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   `a`,
				expected: []interface{}{"b", "c"},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   1,
				expected: []interface{}{2, 3},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   1.0,
				expected: []interface{}{2.0, 3.0},
			},
		},
		{
			name: "with wrong types",
			args: args{
				actual:   2.0,
				expected: []interface{}{"a", 3},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotBeBetween(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotBeBetween() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldContain(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   []interface{}{"a", "c"},
				expected: []interface{}{`a`},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   []interface{}{1, 2},
				expected: []interface{}{1},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   []interface{}{1.0, 2.0},
				expected: []interface{}{1.0},
			},
		},
		{
			name: "raise error",
			args: args{
				actual:   []interface{}{1.0, 2.0},
				expected: []interface{}{3.0},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldContain(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldContain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotContain(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   []interface{}{"a", "c"},
				expected: []interface{}{`b`},
			},
		},
		{
			name: "with int",
			args: args{
				actual:   []interface{}{1, 2},
				expected: []interface{}{3},
			},
		},
		{
			name: "with float",
			args: args{
				actual:   []interface{}{1.0, 2.0},
				expected: []interface{}{1.1},
			},
		},
		{
			name: "raise error",
			args: args{
				actual:   []interface{}{1.0, 2.0},
				expected: []interface{}{1.0},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotContain(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotContain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldContainKey(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   map[string]interface{}{"a": "", "c": ""},
				expected: []interface{}{`a`},
			},
		},
		{
			name: "raise error",
			args: args{
				actual:   map[string]interface{}{"a": "", "c": ""},
				expected: []interface{}{`b`},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldContainKey(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldContainKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotContainKey(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with string",
			args: args{
				actual:   map[string]interface{}{"a": "", "c": ""},
				expected: []interface{}{`b`},
			},
		},
		{
			name: "raise error",
			args: args{
				actual:   map[string]interface{}{"a": "", "c": ""},
				expected: []interface{}{`a`},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotContainKey(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotContainKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldBeEmpty(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual: map[string]interface{}{},
			},
		},
		{
			name: "ko",
			args: args{
				actual: map[string]interface{}{"a": "", "c": ""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldBeEmpty(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldBeEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotBeEmpty(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ko",
			args: args{
				actual: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				actual: map[string]interface{}{"a": "", "c": ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotBeEmpty(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotBeEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHaveLength(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok with slice",
			args: args{
				actual:   []interface{}{"a"},
				expected: []interface{}{1},
			},
		},
		{
			name: "ok with map",
			args: args{
				actual:   map[string]interface{}{"a": "a"},
				expected: []interface{}{1},
			},
		},
		{
			name: "ok with string",
			args: args{
				actual:   "a",
				expected: []interface{}{1},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   []interface{}{"a"},
				expected: []interface{}{2},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldHaveLength(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldHaveLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldStartWith(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   "aaa",
				expected: []interface{}{"a"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "aaa",
				expected: []interface{}{"b"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldStartWith(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldStartWith() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotStartWith(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   "aaa",
				expected: []interface{}{"b"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "aaa",
				expected: []interface{}{"a"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotStartWith(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotStartWith() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldEndWith(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   "aaa-",
				expected: []interface{}{"a-"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "aaa-",
				expected: []interface{}{"b"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldEndWith(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldEndWith() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotEndWith(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   "aaa-",
				expected: []interface{}{"b"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "aaa-",
				expected: []interface{}{"a-"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotEndWith(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotEndWith() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldContainSubstring(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   "aaa-x",
				expected: []interface{}{"a-"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "aaa-x",
				expected: []interface{}{"b-"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldContainSubstring(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldContainSubstring() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldNotContainSubstring(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   "aaa-x",
				expected: []interface{}{"b-"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "aaa-x",
				expected: []interface{}{"a-"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldNotContainSubstring(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldNotContainSubstring() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldEqualTrimSpace(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok with string",
			args: args{
				actual:   ` a`,
				expected: []interface{}{`a`},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   ` ba`,
				expected: []interface{}{`a`},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldEqualTrimSpace(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldEqualTrimSpace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHappenBefore(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(1 * time.Second)},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(-1 * time.Second)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldHappenBefore(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldHappenBefore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHappenOnOrBefore(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(1 * time.Second)},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(-1 * time.Second)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldHappenOnOrBefore(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldHappenOnOrBefore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHappenAfter(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(-1 * time.Second)},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(1 * time.Second)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldHappenAfter(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldHappenAfter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHappenOnOrAfter(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(-1 * time.Second)},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(1 * time.Second)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldHappenOnOrAfter(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldHappenOnOrAfter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHappenBetween(t *testing.T) {
	type args struct {
		actual   interface{}
		expected []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(-1 * time.Second), time.Now().Add(1 * time.Second)},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   time.Now(),
				expected: []interface{}{time.Now().Add(1 * time.Second), time.Now().Add(2 * time.Second)},
			},
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				actual:   "2006-01-02T15:04:05+07:00",
				expected: []interface{}{"2006-01-02T15:04:00+07:00", "2006-01-02T15:04:10+07:00"},
			},
		},
		{
			name: "ko",
			args: args{
				actual:   "2006-01-02T15:04:00+07:00",
				expected: []interface{}{"2006-01-02T15:04:05+07:00", "2006-01-02T15:04:10+07:00"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShouldHappenBetween(tt.args.actual, tt.args.expected...); (err != nil) != tt.wantErr {
				t.Errorf("ShouldHappenBetween() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
