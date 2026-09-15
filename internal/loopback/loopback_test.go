package loopback_test

import (
	"testing"

	"github.com/Raxuis/chaosproxy/internal/loopback"
)

func TestIsHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		want bool
	}{
		{host: "localhost", want: true},
		{host: "LOCALHOST.", want: true},
		{host: "app.localhost", want: true},
		{host: "127.0.0.1", want: true},
		{host: "127.8.9.10", want: true},
		{host: "::1", want: true},
		{host: "[::1]", want: true},
		{host: "", want: false},
		{host: "0.0.0.0", want: false},
		{host: "192.168.1.20", want: false},
		{host: "localhost.evil.example", want: false},
		{host: "evillocalhost", want: false},
	}
	for _, test := range tests {
		if got := loopback.IsHost(test.host); got != test.want {
			t.Errorf("IsHost(%q) = %t, want %t", test.host, got, test.want)
		}
	}
}
