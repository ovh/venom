package venom

import "testing"

func TestRedactURI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "postgres URI with password", in: "postgres://u:p@h/db", want: "postgres://u:***@h/db"},
		{name: "postgres URI with options", in: "postgres://u:p@h:5432/db?sslmode=disable", want: "postgres://u:***@h:5432/db?sslmode=disable"},
		{name: "mongodb URI", in: "mongodb://u:p@h:27017/?retryWrites=true", want: "mongodb://u:***@h:27017/?retryWrites=true"},
		{name: "redis URI no user", in: "redis://:secret@h:6379", want: "redis://:***@h:6379"},
		{name: "amqp URI", in: "amqp://u:p@h:5672/vhost", want: "amqp://u:***@h:5672/vhost"},
		{name: "URI without password", in: "postgres://u@h/db", want: "postgres://u@h/db"},
		{name: "URI without user", in: "https://example.com/foo", want: "https://example.com/foo"},
		{name: "MySQL DSN", in: "u:p@tcp(h:3306)/db", want: "u:***@tcp(h:3306)/db"},
		{name: "MySQL DSN with options", in: "user:s3cret@tcp(host:3306)/dbname?parseTime=true", want: "user:***@tcp(host:3306)/dbname?parseTime=true"},
		{name: "SQLite memory", in: ":memory:", want: ":memory:"},
		{name: "SQLite file", in: "/path/to/file.db", want: "/path/to/file.db"},
		{name: "SQLite relative", in: "file:test.db", want: "file:test.db"},
		{name: "plain string", in: "not-a-dsn", want: "not-a-dsn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactURI(tt.in)
			if got != tt.want {
				t.Fatalf("RedactURI(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
