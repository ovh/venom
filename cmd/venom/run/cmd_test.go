package run

import (
	"io"
	"strings"
	"testing"

	"github.com/ovh/venom"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
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
			name: "from environment",
			args: args{
				env: []string{`VENOM_VAR_a=1`, `VENOM_VAR_b="B"`, `VENOM_VAR_c=[1,2,3]`},
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
