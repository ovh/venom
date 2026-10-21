// Copyright 2026 Venom contributors.
// Licensed under the Apache License, Version 2.0 (see LICENSE).

package ovhapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/venom"
)

func TestExecutor_Resolve(t *testing.T) {
	for _, tt := range []struct{ name, network, address, destination string }{
		{"IPv4", "tcp4", "127.0.0.1:0", "127.0.0.1"},
		{"hostname", "tcp4", "127.0.0.1:0", "localhost"},
		{"bracketed IPv4", "tcp4", "127.0.0.1:0", "[127.0.0.1]"},
		{"bracketed hostname", "tcp4", "127.0.0.1:0", "[localhost]"},
		{"IPv6", "tcp6", "[::1]:0", "::1"},
		{"bracketed IPv6", "tcp6", "[::1]:0", "[::1]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := net.Listen(tt.network, tt.address)
			if err != nil && tt.network == "tcp6" {
				t.Skipf("IPv6 loopback unavailable: %v", err)
			}
			require.NoError(t, err)
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]string{"host": r.Host})
			}))
			require.NoError(t, srv.Listener.Close())
			srv.Listener = listener
			srv.Start()
			t.Cleanup(srv.Close)

			_, port, err := net.SplitHostPort(listener.Addr().String())
			require.NoError(t, err)
			host := net.JoinHostPort("localhost", port)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res, err := (Executor{}).Run(ctx, venom.TestStep{
				"endpoint":          "http://" + host,
				"applicationKey":    "unused",
				"applicationSecret": "unused",
				"path":              "/healthz",
				"noAuth":            true,
				"resolve":           []string{host + ":" + tt.destination},
			})
			require.NoError(t, err)
			result := res.(Result)
			require.Equal(t, http.StatusOK, result.StatusCode)
			require.Equal(t, host, result.BodyJSON.(map[string]interface{})["host"])
		})
	}
}
