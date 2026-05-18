package controlplane

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// captureAudit runs f with auditWriter redirected to a buffer and returns the
// captured output.
func captureAudit(t *testing.T, f func()) string {
	t.Helper()
	var buf strings.Builder
	restore := setAuditWriter(&buf)
	defer restore()
	f()
	return buf.String()
}

// TestEmitAuditOneLinePerStatus verifies that emitAudit emits exactly one
// line per TargetStatus entry.
func TestEmitAuditOneLinePerStatus(t *testing.T) {
	statuses := []TargetStatus{
		{Env: "staging", Server: "stg-app", Capability: CapManage, ProbedAt: time.Now(), SSHReachable: true, KeyPresent: true, DockerOK: true},
		{Env: "prod", Server: "prod-app", Capability: CapReadOnly, ProbedAt: time.Now(), Reason: "SSH unreachable (timeout)"},
		{Env: "staging", Server: "stg-obs", Capability: CapHidden, ProbedAt: time.Now()},
	}

	output := captureAudit(t, func() { emitAudit(statuses) })

	lines := nonEmptyLines(output)
	if len(lines) != len(statuses) {
		t.Errorf("emitAudit: got %d lines, want %d", len(lines), len(statuses))
	}
}

// TestEmitAuditContainsRequiredFields verifies each audit line contains the
// required structured fields.
func TestEmitAuditContainsRequiredFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	statuses := []TargetStatus{
		{
			Env:          "prod",
			Server:       "prod-app",
			Capability:   CapManage,
			ProbedAt:     now,
			SSHReachable: true,
			KeyPresent:   true,
			DockerOK:     true,
			LatencyMS:    42,
		},
	}

	output := captureAudit(t, func() { emitAudit(statuses) })

	requiredFields := []string{
		"component=controlplane",
		"env=prod",
		"server=prod-app",
		"capability=manage",
		"ssh_reachable=true",
		"key_present=true",
		"docker_ok=true",
		"latency_ms=42",
	}
	for _, field := range requiredFields {
		if !strings.Contains(output, field) {
			t.Errorf("audit line missing field %q in output:\n%s", field, output)
		}
	}
}

// TestEmitAuditReasonIncludedWhenNonEmpty verifies the reason field appears
// in the audit line when TargetStatus.Reason is set.
func TestEmitAuditReasonIncludedWhenNonEmpty(t *testing.T) {
	statuses := []TargetStatus{
		{
			Env:        "prod",
			Server:     "prod-app",
			Capability: CapReadOnly,
			ProbedAt:   time.Now(),
			Reason:     "SSH unreachable (timeout)",
		},
	}

	output := captureAudit(t, func() { emitAudit(statuses) })

	if !strings.Contains(output, "reason=") {
		t.Errorf("expected reason= field in audit output; got:\n%s", output)
	}
}

// TestEmitAuditReasonOmittedWhenEmpty verifies the reason field is omitted
// when TargetStatus.Reason is empty (manage case).
func TestEmitAuditReasonOmittedWhenEmpty(t *testing.T) {
	statuses := []TargetStatus{
		{
			Env:          "local",
			Server:       "local-app",
			Capability:   CapManage,
			ProbedAt:     time.Now(),
			Reason:       "",
			SSHReachable: true,
		},
	}

	output := captureAudit(t, func() { emitAudit(statuses) })

	if strings.Contains(output, "reason=") {
		t.Errorf("reason= must not appear for empty Reason; got:\n%s", output)
	}
}

// TestEmitAuditRedactsKeyFilePaths verifies that filesystem paths in Reason
// are replaced with "<redacted>" in audit output.
func TestEmitAuditRedactsKeyFilePaths(t *testing.T) {
	statuses := []TargetStatus{
		{
			Env:        "prod",
			Server:     "prod-app",
			Capability: CapReadOnly,
			ProbedAt:   time.Now(),
			// Reason contains a filesystem path that must be redacted.
			Reason: "key /home/user/.ssh/id_ed25519 not found",
		},
	}

	output := captureAudit(t, func() { emitAudit(statuses) })

	if strings.Contains(output, "/home/user") {
		t.Errorf("filesystem path must be redacted; found in output:\n%s", output)
	}
	if !strings.Contains(output, "<redacted>") {
		t.Errorf("expected <redacted> in output; got:\n%s", output)
	}
}

// TestEmitAuditNoSecretOrKeyPathInAnyField verifies that no audit line contains
// a literal filesystem path in any field.
func TestEmitAuditNoSecretOrKeyPathInAnyField(t *testing.T) {
	statuses := []TargetStatus{
		{Env: "prod", Server: "app", Capability: CapManage, ProbedAt: time.Now(),
			Reason: "no SSH key for NSELF_SSH_KEY_PROD"}, // env-var name is OK
		{Env: "prod", Server: "obs", Capability: CapReadOnly, ProbedAt: time.Now(),
			Reason: "SSH unreachable (timeout)"}, // no path
	}

	output := captureAudit(t, func() { emitAudit(statuses) })

	// Env-var names (uppercase, underscores) must pass through.
	if !strings.Contains(output, "NSELF_SSH_KEY_PROD") {
		t.Errorf("env-var name must appear in audit output; got:\n%s", output)
	}

	// No output line should contain what looks like a filesystem path.
	for _, line := range nonEmptyLines(output) {
		if strings.Contains(line, "/home/") || strings.Contains(line, "/root/") {
			t.Errorf("filesystem path found in audit line: %q", line)
		}
	}
}

// TestRedactKeyPaths covers the redaction logic in isolation.
func TestRedactKeyPaths(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Env-var names must not be redacted.
		{"no SSH key for NSELF_SSH_KEY_PROD", "no SSH key for NSELF_SSH_KEY_PROD"},
		// Absolute paths must be redacted.
		{"key /home/user/.ssh/id_ed25519 missing", "key <redacted> missing"},
		// Tilde paths must be redacted.
		{"key ~/.ssh/id_rsa not found", "key <redacted> not found"},
		// Multiple paths in one string.
		{"/etc/nself/key and /tmp/key2", "<redacted> and <redacted>"},
		// Empty string.
		{"", ""},
		// Plain text with no paths.
		{"SSH unreachable (timeout)", "SSH unreachable (timeout)"},
	}

	for _, tc := range tests {
		got := redactKeyPaths(tc.input)
		if got != tc.want {
			t.Errorf("redactKeyPaths(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestToAuditLineConvertsCorrectly verifies the TargetStatus→auditLine mapping.
func TestToAuditLineConvertsCorrectly(t *testing.T) {
	now := time.Now()
	ts := TargetStatus{
		Env:          "staging",
		Server:       "stg-app",
		Capability:   CapManage,
		SSHReachable: true,
		KeyPresent:   true,
		DockerOK:     true,
		LatencyMS:    15,
		Reason:       "",
		ProbedAt:     now,
	}

	line := toAuditLine(ts)

	if line.Env != ts.Env {
		t.Errorf("Env: got %q, want %q", line.Env, ts.Env)
	}
	if line.Server != ts.Server {
		t.Errorf("Server: got %q, want %q", line.Server, ts.Server)
	}
	if line.Capability != string(ts.Capability) {
		t.Errorf("Capability: got %q, want %q", line.Capability, ts.Capability)
	}
	if line.LatencyMS != ts.LatencyMS {
		t.Errorf("LatencyMS: got %d, want %d", line.LatencyMS, ts.LatencyMS)
	}
}

// nonEmptyLines splits s by newlines and returns only non-empty lines.
func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// TestEmitAuditConcurrentNoPanic verifies that concurrent calls to emitAudit
// and setAuditWriter do not race or panic. Run with -race to detect data races.
// This exercises the auditMu mutex that guards the package-level auditWriter.
func TestEmitAuditConcurrentNoPanic(t *testing.T) {
	const goroutines = 20

	statuses := []TargetStatus{
		{Env: "prod", Server: "app", Capability: CapManage, ProbedAt: time.Now(), SSHReachable: true, KeyPresent: true, DockerOK: true},
		{Env: "prod", Server: "obs", Capability: CapReadOnly, ProbedAt: time.Now(), Reason: "SSH unreachable (timeout)"},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Alternate between emitting and replacing the writer to exercise
			// both code paths under concurrency.
			var buf strings.Builder
			restore := setAuditWriter(&buf)
			defer restore()
			emitAudit(statuses)
		}()
	}

	wg.Wait()
	// If we reach here without a panic or race detector hit, the mutex is working.
}
