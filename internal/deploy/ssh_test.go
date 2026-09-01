package deploy

import (
	"context"
	"strings"
	"testing"
)

// TestDeployViaSsh_RejectsInjectedRemotePath (T31) is the defense-in-depth
// test for DeployViaSsh: even when an SSHConfig is constructed directly
// (bypassing the CLI-flag and inventory-file validation layers in
// cmd/commands and internal/controlplane), a Host carrying a RemotePath
// with shell metacharacters must be rejected before any rsync/ssh exec call
// is attempted -- so this test needs neither binary to be present on PATH.
func TestDeployViaSsh_RejectsInjectedRemotePath(t *testing.T) {
	payloads := []string{
		"ubuntu@example.com:/opt/x; id",
		"ubuntu@example.com:/opt/x$(id)",
		"ubuntu@example.com:/opt/x`id`",
	}

	for _, host := range payloads {
		cfg := SSHConfig{Host: host, KeyPath: "/tmp/does-not-matter"}
		err := DeployViaSsh(context.Background(), cfg, "/tmp/nself-compose.yml")
		if err == nil {
			t.Fatalf("DeployViaSsh(%q): expected rejection, got nil error", host)
		}
		if !strings.Contains(err.Error(), "unsafe characters") {
			t.Errorf("DeployViaSsh(%q): unexpected error: %v", host, err)
		}
	}
}

// TestDeployViaSsh_EmptyHost proves the pre-existing empty-Host guard is
// unaffected by the new RemotePath check (which runs after it).
func TestDeployViaSsh_EmptyHost(t *testing.T) {
	err := DeployViaSsh(context.Background(), SSHConfig{}, "/tmp/nself-compose.yml")
	if err == nil {
		t.Fatal("DeployViaSsh with empty Host: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Host is empty") {
		t.Errorf("unexpected error: %v", err)
	}
}
