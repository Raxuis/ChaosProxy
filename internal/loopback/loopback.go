// Package loopback identifies hosts that only resolve to the local machine.
package loopback

import (
	"net"
	"strings"
)

// IsHost reports whether host is localhost, a *.localhost name, or a loopback IP.
func IsHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.Trim(host, "[]")), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
