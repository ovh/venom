// Copyright 2026 Venom contributors.
// Licensed under the Apache License, Version 2.0 (see LICENSE).

package resolve_test

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/ovh/venom/internal/resolve"
)

func TestAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		mappings []string
		want     string
	}{
		{"IPv4", "example.com:443", []string{"example.com:443:127.0.0.1"}, "127.0.0.1:443"},
		{"hostname", "example.com:443", []string{"example.com:443:localhost"}, "localhost:443"},
		{"bracketed IPv4", "example.com:443", []string{"example.com:443:[127.0.0.1]"}, "127.0.0.1:443"},
		{"bracketed hostname", "example.com:443", []string{"example.com:443:[localhost]"}, "localhost:443"},
		{"IPv6 loopback", "example.com:443", []string{"example.com:443:::1"}, "[::1]:443"},
		{"IPv6 compressed", "example.com:443", []string{"example.com:443:2001:db8::1"}, "[2001:db8::1]:443"},
		{"IPv6 full", "example.com:443", []string{"example.com:443:2001:db8:0:0:0:0:0:1"}, "[2001:db8:0:0:0:0:0:1]:443"},
		{"bracketed IPv6", "example.com:443", []string{"example.com:443:[::1]"}, "[::1]:443"},
		{"IPv6 zone", "example.com:443", []string{"example.com:443:fe80::1%eth0"}, "[fe80::1%eth0]:443"},
		{"bracketed IPv6 zone", "example.com:443", []string{"example.com:443:[fe80::1%eth0]"}, "[fe80::1%eth0]:443"},
		{"IPv4 mapped IPv6", "example.com:443", []string{"example.com:443:::ffff:127.0.0.1"}, "[::ffff:127.0.0.1]:443"},
		{"unmatched hostname", "other.com:443", []string{"example.com:443:::1"}, "other.com:443"},
		{"unmatched port", "example.com:80", []string{"example.com:443:::1"}, "example.com:80"},
		{"exact hostname", "EXAMPLE.com:443", []string{"example.com:443:::1"}, "EXAMPLE.com:443"},
		{"exact port", "example.com:0443", []string{"example.com:443:::1"}, "example.com:0443"},
		{"original IPv6", "[::1]:443", []string{"example.com:443:::1"}, "[::1]:443"},
		{"original IPv6 zone", "[fe80::1%eth0]:443", []string{"example.com:443:::1"}, "[fe80::1%eth0]:443"},
		{"IPv4 source", "127.0.0.1:443", []string{"127.0.0.1:443:::1"}, "[::1]:443"},
		{"multiple entries", "example.com:443", []string{"other.com:443:localhost", "example.com:443:::1"}, "[::1]:443"},
		{"ordered chaining", "example.com:443", []string{"example.com:443:localhost", "localhost:443:::1"}, "[::1]:443"},
		{"duplicate source", "example.com:443", []string{"example.com:443:localhost", "example.com:443:::1"}, "localhost:443"},
		{"service port", "example.com:https", []string{"example.com:https:::1"}, "[::1]:https"},
		{"port validation deferred", "example.com:65536", []string{"example.com:65536:127.0.0.1"}, "127.0.0.1:65536"},
		{"hostname validation deferred", "example.com:443", []string{"example.com:443:backend_service"}, "backend_service:443"},
		{"IPv4 validation deferred", "example.com:443", []string{"example.com:443:999.0.0.1"}, "999.0.0.1:443"},
		{"no mappings", "example.com:443", nil, "example.com:443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolve.Address(tt.addr, tt.mappings)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("Address() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddressInvalidMapping(t *testing.T) {
	for _, mapping := range []string{
		"", "example.com", "example.com:443", ":443:::1", "example.com::::1", "example.com:443:",
		"example.com:443:[::1", "example.com:443:::1]", "example.com:443:[[::1]]",
		"example.com:443:[]",
		"example.com:443:2001:db8::g", "example.com:443:localhost:80", "example.com:443:127.0.0.1:80",
		"example.com:443:[::1]:80", "example.com:443:fe80::1%", "[::1]:443:127.0.0.1",
		"example.com:443:[fe80::1%eth0]]", "example.com:443:fe80::1%eth0]", "example.com:443:[fe80::1%[eth0]",
	} {
		t.Run(mapping, func(t *testing.T) {
			// Invalid entries must report an error even after a previous match.
			_, err := resolve.Address("example.com:443", []string{"example.com:443:localhost", mapping})
			if err == nil || !strings.Contains(err.Error(), "invalid value for resolve attribute:") || !strings.Contains(err.Error(), mapping) {
				t.Fatalf("Address() error = %v, want an error identifying mapping %q", err, mapping)
			}
		})
	}
}

func TestAddressInvalidPort(t *testing.T) {
	for _, port := range []string{"-1", "65536", "venom-invalid-service"} {
		t.Run(port, func(t *testing.T) {
			addr, err := resolve.Address(net.JoinHostPort("example.com", port), []string{"example.com:" + port + ":127.0.0.1"})
			if err != nil {
				t.Fatal(err)
			}
			var dialer net.Dialer
			conn, err := dialer.DialContext(context.Background(), "tcp", addr)
			if err == nil {
				conn.Close()
				t.Fatal("expected the dialer to reject the invalid port")
			}
		})
	}
}
