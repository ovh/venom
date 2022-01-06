package run

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/ovh/venom"
	"github.com/stretchr/testify/require"
)

func Test_readInitialVariables(t *testing.T) {
	venom.InitTestLogger(t)
	type args struct {
		argsVars     []string
		argVarsFiles []io.Reader
		env          []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "from args",
			args: args{
				argsVars: []string{`a=1`, `b="B"`, `c=[1,2,3]`},
			},
			want: map[string]interface{}{
				"a": 1.0,
				"b": "B",
				"c": []interface{}{1.0, 2.0, 3.0},
			},
		},
		{
			name: "from args",
			args: args{
				argsVars: []string{`db.dsn="user=test password=test dbname=yo host=localhost port=1234 sslmode=disable"`},
			},
			want: map[string]interface{}{
				"db.dsn": "user=test password=test dbname=yo host=localhost port=1234 sslmode=disable",
			},
		},
		{
			name: "from readers",
			args: args{
				argVarsFiles: []io.Reader{
					strings.NewReader(`
a: 1
b: B
c:
  - 1
  - 2
  - 3`),
				},
			},
			want: map[string]interface{}{
				"a": 1.0,
				"b": "B",
				"c": []interface{}{1.0, 2.0, 3.0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readInitialVariables(context.TODO(), tt.args.argsVars, tt.args.argVarsFiles, tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("readInitialVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.EqualValues(t, tt.want, got)
		})
	}
}

func Test_mergeVariables(t *testing.T) {
	ma := mergeVariables("aa=bb", []string{"cc=dd", "ee=ff"})
	require.Equal(t, 3, len(ma))

	mb := mergeVariables("aa=bb", []string{"aa=dd"})
	require.Equal(t, 1, len(mb))

	mc := mergeVariables("aa=bb=dd", []string{"aa=dd"})
	require.Equal(t, 1, len(mc))

	md := mergeVariables("aa=bb=dd", []string{"cc=dd"})
	require.Equal(t, 2, len(md))
}

func Test_initFromEnv(t *testing.T) {
	env := []string{`VENOM_VAR_a=1`, `VENOM_VAR_b="B"`, `VENOM_VAR_c=[1,2,3]`}
	err := initFromEnv(env)
	found := Options.Variables
	require.NoError(t, err)
	require.Equal(t, 3, len(found))
	var nb int
	for i := range found {
		if found[i] == "a=1" {
			nb++
		} else if found[i] == "b=B" {
			nb++
		} else if found[i] == "c=[1,2,3]" {
			nb++
		}
	}
}
