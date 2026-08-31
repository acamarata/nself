//go:build darwin

// Package trust — coverage_g0t11_darwin_test.go: darwin-only follow-on coverage.
//
// These tests reference symbols defined only on darwin (findDnsmasqConfFunc,
// findDnsmasqConf, findDnsmasqConfReal, dnsmasqConfPaths, configureDnsmasqConf).
// They are gated with //go:build darwin so that go vet / go test on Linux do
// not attempt to resolve these symbols.
package trust

import (
	"os"
	"testing"
)

// TestConfigureDnsmasqConf_TempPath drives configureDnsmasqConf against a
// temp path by overriding the findDnsmasqConfFunc seam. Exercises the create-
// and-append branch end-to-end without requiring /opt/homebrew permissions.
func TestConfigureDnsmasqConf_TempPath(t *testing.T) {
	tmp := t.TempDir()
	confPath := tmp + "/dnsmasq.conf"

	orig := findDnsmasqConfFunc
	findDnsmasqConfFunc = func() string { return confPath }
	defer func() { findDnsmasqConfFunc = orig }()

	if err := configureDnsmasqConf(); err != nil {
		t.Fatalf("configureDnsmasqConf failed: %v", err)
	}

	// Second call should be idempotent — line is already present.
	if err := configureDnsmasqConf(); err != nil {
		t.Fatalf("second configureDnsmasqConf failed: %v", err)
	}

	// Verify the file contents.
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("reading temp dnsmasq.conf: %v", err)
	}
	s := string(data)
	if !containsStr(s, dnsmasqConfLine) {
		t.Errorf("conf missing magic line: %q", s)
	}
	// Line should appear exactly once (idempotent).
	count := countOccurrences(s, dnsmasqConfLine)
	if count != 1 {
		t.Errorf("expected 1 occurrence, got %d", count)
	}
}

// TestFindDnsmasqConfSeam_RoundTrip verifies the seam restoration behaviour.
func TestFindDnsmasqConfSeam_RoundTrip(t *testing.T) {
	orig := findDnsmasqConfFunc
	findDnsmasqConfFunc = func() string { return "/tmp/test-only-dnsmasq.conf" }
	got := findDnsmasqConf()
	if got != "/tmp/test-only-dnsmasq.conf" {
		t.Errorf("seam override not honoured: got %q", got)
	}
	findDnsmasqConfFunc = orig
	// After restore, must return one of the real candidates.
	got = findDnsmasqConf()
	candidates := dnsmasqConfPaths()
	found := false
	for _, c := range candidates {
		if got == c {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("after restore, findDnsmasqConf returned %q, not in candidates", got)
	}
}

// TestFindDnsmasqConfReal_FallbackPath verifies that when no candidate exists
// (CI environment), the function returns the documented Apple Silicon fallback.
func TestFindDnsmasqConfReal_FallbackPath(t *testing.T) {
	// findDnsmasqConfReal probes real paths. In CI both /opt/homebrew/etc/...
	// and /usr/local/etc/... usually don't exist — fallback returns
	// /opt/homebrew/etc/dnsmasq.conf.
	got := findDnsmasqConfReal()
	if got == "" {
		t.Error("findDnsmasqConfReal must return non-empty path")
	}
}

// countOccurrences returns the number of times needle appears in haystack.
func countOccurrences(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	count := 0
	pos := 0
	for {
		idx := indexOf(haystack[pos:], needle)
		if idx < 0 {
			break
		}
		count++
		pos += idx + len(needle)
	}
	return count
}

// indexOf returns the first index of needle in haystack, or -1.
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
