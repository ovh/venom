package venom

import "testing"

func Test_readPartialYML(t *testing.T) {
	type args struct {
		btes      []byte
		attribute string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{
				btes: []byte(`
foo:
  - foo1
  - foo2

record:
  - val
  - to be recorded

bar:
  bar1: bar1v
  bar2: bar2v
				`),
				attribute: "record",
			},
			want: `record:
  - val
  - to be recorded
`,
		},
		{
			name: "simple",
			args: args{
				btes: []byte(`
foo:
- foo1
- foo2

record:
- val
- to be recorded

bar:
  bar1: bar1v
  bar2: bar2v
				`),
				attribute: "record",
			},
			want: `record:
- val
- to be recorded
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readPartialYML(tt.args.btes, tt.args.attribute); got != tt.want {
				t.Errorf("readPartialYML() = %v, want %v", got, tt.want)
			}
		})
	}
}
