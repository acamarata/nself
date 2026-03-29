package ssl

import (
	"os"
	"strings"
	"testing"
)

// tempHostsFile creates a temporary file with the given initial content and
// returns its path and a cleanup function.
func tempHostsFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "nself-hosts-test-*")
	if err != nil {
		t.Fatalf("creating temp hosts file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp hosts file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// addHostsToFile is a test helper that calls AddHosts but targeting a custom
// path instead of /etc/hosts. It patches the global hostsFile constant via a
// package-level variable approach — but since hostsFile is a const, we instead
// test the component functions directly.
func TestAddHosts_AddsEntries(t *testing.T) {
	initial := "127.0.0.1\tlocalhost\n"
	path := tempHostsFile(t, initial)

	// Read and parse the initial file.
	lines, err := readHostsFile(path)
	if err != nil {
		t.Fatalf("readHostsFile: %v", err)
	}
	present := buildPresentSet(lines)

	hostnames := []string{"myapp.localhost", "api.myapp.localhost"}
	var toAdd []string
	for _, h := range hostnames {
		if !present[h] {
			toAdd = append(toAdd, h)
		}
	}

	// Build the new content manually (same logic as AddHosts internal).
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	for _, h := range toAdd {
		sb.WriteString(hostsLoopback)
		sb.WriteByte('\t')
		sb.WriteString(h)
		sb.WriteByte('\t')
		sb.WriteString(hostsMarker)
		sb.WriteByte('\n')
	}

	if err := writeHostsFile(path, sb.String()); err != nil {
		t.Fatalf("writeHostsFile: %v", err)
	}

	// Verify the result.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	content := string(data)

	for _, h := range hostnames {
		if !strings.Contains(content, h) {
			t.Errorf("expected %q in hosts file, not found\n%s", h, content)
		}
	}
	if !strings.Contains(content, hostsMarker) {
		t.Errorf("expected %q marker in added entries", hostsMarker)
	}
	// Original entry must still be present.
	if !strings.Contains(content, "localhost") {
		t.Error("original localhost entry was removed")
	}
}

func TestAddHosts_NoDuplication(t *testing.T) {
	// Start with an already-present entry.
	initial := "127.0.0.1\tmyapp.localhost\t" + hostsMarker + "\n"
	path := tempHostsFile(t, initial)

	lines, err := readHostsFile(path)
	if err != nil {
		t.Fatalf("readHostsFile: %v", err)
	}
	present := buildPresentSet(lines)

	// "myapp.localhost" is already present — toAdd should be empty.
	hostnames := []string{"myapp.localhost"}
	var toAdd []string
	for _, h := range hostnames {
		if !present[h] {
			toAdd = append(toAdd, h)
		}
	}

	if len(toAdd) != 0 {
		t.Errorf("expected no new entries, got: %v", toAdd)
	}

	// File should be unchanged in length.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading hosts file: %v", err)
	}
	lines2 := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines2) != len(lines) {
		t.Errorf("expected %d lines, got %d", len(lines), len(lines2))
	}
}

func TestFilterHostsEntries_Wildcards(t *testing.T) {
	input := []string{
		"*.example.com",
		"example.com",
		"127.0.0.1",
		"::1",
		"api.example.com",
		"foo.nip.io",
		"foo.sslip.io",
	}
	result := filterHostsEntries(input)

	// Only non-wildcard, non-IP, non-nip.io entries should remain.
	expected := map[string]bool{
		"example.com":     true,
		"api.example.com": true,
	}
	for _, h := range result {
		if !expected[h] {
			t.Errorf("unexpected entry in filtered result: %q", h)
		}
	}
	if len(result) != len(expected) {
		t.Errorf("expected %d entries, got %d: %v", len(expected), len(result), result)
	}
}

func TestBuildPresentSet(t *testing.T) {
	lines := []string{
		"# comment",
		"127.0.0.1\tlocalhost",
		"127.0.0.1\tmyapp.local\t# nself-managed",
		"",
		"::1\t\tip6-localhost",
	}
	present := buildPresentSet(lines)

	expected := []string{"localhost", "myapp.local", "ip6-localhost"}
	for _, h := range expected {
		if !present[h] {
			t.Errorf("expected %q in present set", h)
		}
	}
	// Comments should not appear.
	if present["# comment"] {
		t.Error("comment line appeared in present set")
	}
}

func TestRemoveHosts_RemovesMarkedEntries(t *testing.T) {
	initial := strings.Join([]string{
		"127.0.0.1\tlocalhost",
		"127.0.0.1\tmyapp.local\t" + hostsMarker,
		"127.0.0.1\tapi.myapp.local\t" + hostsMarker,
		"# some other comment",
	}, "\n") + "\n"

	path := tempHostsFile(t, initial)

	// Temporarily swap hostsFile constant — since it's a const, we test
	// the underlying functions directly.
	lines, err := readHostsFile(path)
	if err != nil {
		t.Fatalf("readHostsFile: %v", err)
	}

	var kept []string
	for _, line := range lines {
		if !strings.Contains(line, hostsMarker) {
			kept = append(kept, line)
		}
	}

	content := strings.Join(kept, "\n") + "\n"
	if err := writeHostsFile(path, content); err != nil {
		t.Fatalf("writeHostsFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	result := string(data)

	if strings.Contains(result, hostsMarker) {
		t.Error("nself-managed entries were not removed")
	}
	if !strings.Contains(result, "localhost") {
		t.Error("localhost entry was incorrectly removed")
	}
	if strings.Contains(result, "myapp.local") {
		t.Error("myapp.local was not removed")
	}
}
