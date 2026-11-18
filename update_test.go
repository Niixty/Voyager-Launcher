//go:build windows

package main

import "testing"

func TestNewerVersion(t *testing.T) {
	tests := []struct {
		remote, local string
		want          bool
	}{
		{"v0.8.1", "0.8.0", true},
		{"v0.8.20", "v0.8.9", true},
		{"0.9.0", "0.8.700", true},
		{"v0.7.99", "v0.8.0", false},
		{"v0.8.1", "v0.8.1", false},
		{"v0.8.beta", "v0.8.0", false},
	}
	for _, tc := range tests {
		if got := newerVersion(tc.remote, tc.local); got != tc.want {
			t.Errorf("newerVersion(%q,%q) = %t, expected %t", tc.remote, tc.local, got, tc.want)
		}
	}
}
