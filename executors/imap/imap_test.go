package imap

import (
	"testing"
)

func TestMail_containsFlag(t *testing.T) {
	type fields struct {
		UID     uint32
		From    string
		To      string
		Subject string
		Body    string
		Flags   []string
	}
	type args struct {
		flag string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Does not contain flag",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flag: "Flag2"},
			want: false,
		},
		{
			name: "Same flags",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flag: "Flag1"},
			want: true,
		},
		{
			name: "Contains the flag",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flag: "Flag1"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mail{
				UID:     tt.fields.UID,
				From:    tt.fields.From,
				To:      tt.fields.To,
				Subject: tt.fields.Subject,
				Body:    tt.fields.Body,
				Flags:   tt.fields.Flags,
			}
			if got := m.containsFlag(tt.args.flag); got != tt.want {
				t.Errorf("containsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMail_containsFlags(t *testing.T) {
	type fields struct {
		UID     uint32
		From    string
		To      string
		Subject string
		Body    string
		Flags   []string
	}
	type args struct {
		flags []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Does not contain flag",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flags: []string{"Flag2"}},
			want: false,
		},
		{
			name: "Same flags",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: true,
		},
		{
			name: "Contains the flag",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flags: []string{"Flag1"}},
			want: true,
		},
		{
			name: "Does not contain one of the flags",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mail{
				UID:     tt.fields.UID,
				From:    tt.fields.From,
				To:      tt.fields.To,
				Subject: tt.fields.Subject,
				Body:    tt.fields.Body,
				Flags:   tt.fields.Flags,
			}
			if got := m.containsFlags(tt.args.flags); got != tt.want {
				t.Errorf("containsFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMail_containsAnyButSeenOrRecent(t *testing.T) {
	type fields struct {
		UID     uint32
		From    string
		To      string
		Subject string
		Body    string
		Flags   []string
	}
	type args struct {
		flags []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Does not contain flag",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flags: []string{"Flag2"}},
			want: false,
		},
		{
			name: `\Seen should be ignored`,
			fields: fields{
				Flags: []string{"Flag1", `\Seen`},
			},
			args: args{flags: []string{`\Seen`}},
			want: false,
		},
		{
			name: `\Recent should be ignored`,
			fields: fields{
				Flags: []string{"Flag1", `\Recent`},
			},
			args: args{flags: []string{`\Recent`}},
			want: false,
		},
		{
			name: `\Seen and \Recent should be ignored`,
			fields: fields{
				Flags: []string{"Flag1", `\Seen`, `\Recent`},
			},
			args: args{flags: []string{`\Seen`, `\Recent`}},
			want: false,
		},
		{
			name: "Same flags",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: true,
		},
		{
			name: "Contains the flag",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flags: []string{"Flag1"}},
			want: true,
		},
		{
			name: "Contains one of the flags",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mail{
				UID:     tt.fields.UID,
				From:    tt.fields.From,
				To:      tt.fields.To,
				Subject: tt.fields.Subject,
				Body:    tt.fields.Body,
				Flags:   tt.fields.Flags,
			}
			if got := m.containsAnyButSeenOrRecent(tt.args.flags); got != tt.want {
				t.Errorf("containsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMail_hasSameFlags(t *testing.T) {
	type fields struct {
		UID     uint32
		From    string
		To      string
		Subject string
		Body    string
		Flags   []string
	}
	type args struct {
		flags []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{

		{
			name: "Does not contain flags",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flags: []string{"Flag2"}},
			want: false,
		},
		{
			name: "Same flags ordered",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: true,
		},
		{
			name: "Same flags unordered",
			fields: fields{
				Flags: []string{"Flag2", "Flag1"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: true,
		},
		{
			name: "Contains flag",
			fields: fields{
				Flags: []string{"Flag1", "Flag2"},
			},
			args: args{flags: []string{"Flag1"}},
			want: false,
		},
		{
			name: "Does not contain one of the flags",
			fields: fields{
				Flags: []string{"Flag1"},
			},
			args: args{flags: []string{"Flag1", "Flag2"}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mail{
				UID:     tt.fields.UID,
				From:    tt.fields.From,
				To:      tt.fields.To,
				Subject: tt.fields.Subject,
				Body:    tt.fields.Body,
				Flags:   tt.fields.Flags,
			}
			if got := m.hasSameFlags(tt.args.flags); got != tt.want {
				t.Errorf("hasSameFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}
