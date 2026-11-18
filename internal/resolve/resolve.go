// Copyright 2026 Venom contributors.
// Licensed under the Apache License, Version 2.0 (see LICENSE).

// Package resolve applies custom hostname:port:destination mappings to dial addresses.
package resolve

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// Address applies mappings in order, preserving exact host and port matching.
// The source is a hostname or IPv4 address; the destination may also be an IPv6
// literal containing a zone identifier. Destinations may be bracketed.
// Hostname lookup and port validation are left to the network dialer.
func Address(addr string, mappings []string) (string, error) {
	for _, mapping := range mappings {
		tuple := strings.SplitN(mapping, ":", 3)
		if len(tuple) != 3 || tuple[0] == "" || tuple[1] == "" || tuple[2] == "" {
			return "", fmt.Errorf("invalid value for resolve attribute: %q: expected hostname:port:destination with non-empty fields", mapping)
		}

		destination := tuple[2]
		if strings.HasPrefix(destination, "[") && strings.HasSuffix(destination, "]") {
			destination = strings.TrimSuffix(strings.TrimPrefix(destination, "["), "]")
		}
		if destination == "" || strings.ContainsAny(destination, ":[]") {
			ip, err := netip.ParseAddr(destination)
			if err != nil || !ip.Is6() || strings.ContainsAny(destination, "[]") {
				return "", fmt.Errorf("invalid value for resolve attribute: %q: invalid IPv6 destination %q", mapping, tuple[2])
			}
		}

		if addr == net.JoinHostPort(tuple[0], tuple[1]) {
			addr = net.JoinHostPort(destination, tuple[1])
		}
	}
	return addr, nil
}
