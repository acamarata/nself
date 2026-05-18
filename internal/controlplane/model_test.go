package controlplane

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// fixtureYAML is a minimal but representative control-plane.yaml document.
const fixtureYAML = `
schema_version: 1
project: testproject
environments:
  local:
    name: local
    kind: local
    servers:
      - name: local-app
        role: app
        primary: true
  staging:
    name: staging
    kind: remote
    servers:
      - name: stg-obs
        role: observability
        host: ubuntu@stg-obs.example.com
        ssh_key_ref: NSELF_SSH_KEY_STAGING
        remote_path: /opt/nself
      - name: stg-app
        role: app
        host: ubuntu@stg-app.example.com
        ssh_key_ref: NSELF_SSH_KEY_STAGING
        remote_path: /opt/nself
        primary: true
      - name: stg-lb
        role: lb
        host: ubuntu@stg-lb.example.com
        ssh_key_ref: NSELF_SSH_KEY_STAGING
        remote_path: /opt/nself
        upstreams:
          - stg-app
`

// TestModelRoundTrip verifies that an Inventory can be marshalled to YAML and
// unmarshalled back to an identical value.
func TestModelRoundTrip(t *testing.T) {
	var inv Inventory
	if err := yaml.Unmarshal([]byte(fixtureYAML), &inv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Re-marshal and re-unmarshal to verify stability.
	b, err := yaml.Marshal(&inv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var inv2 Inventory
	if err := yaml.Unmarshal(b, &inv2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}

	if inv2.SchemaVersion != inv.SchemaVersion {
		t.Errorf("SchemaVersion: got %d, want %d", inv2.SchemaVersion, inv.SchemaVersion)
	}
	if inv2.Project != inv.Project {
		t.Errorf("Project: got %q, want %q", inv2.Project, inv.Project)
	}
	if len(inv2.Environments) != len(inv.Environments) {
		t.Errorf("Environments len: got %d, want %d", len(inv2.Environments), len(inv.Environments))
	}
}

// TestInventoryFields verifies the parsed fields match fixture expectations.
func TestInventoryFields(t *testing.T) {
	var inv Inventory
	if err := yaml.Unmarshal([]byte(fixtureYAML), &inv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if inv.SchemaVersion != 1 {
		t.Errorf("SchemaVersion: got %d, want 1", inv.SchemaVersion)
	}
	if inv.Project != "testproject" {
		t.Errorf("Project: got %q, want testproject", inv.Project)
	}

	local, ok := inv.Environments["local"]
	if !ok {
		t.Fatal("missing 'local' environment")
	}
	if local.Kind != "local" {
		t.Errorf("local.Kind: got %q, want local", local.Kind)
	}
	if len(local.Servers) != 1 {
		t.Fatalf("local.Servers len: got %d, want 1", len(local.Servers))
	}
	ls := local.Servers[0]
	if ls.Role != RoleApp {
		t.Errorf("local server role: got %q, want app", ls.Role)
	}
	if !ls.Primary {
		t.Errorf("local server primary: got false, want true")
	}
	// Local server must have no inline secrets.
	if ls.Host != "" {
		t.Errorf("local server Host should be empty (local env), got %q", ls.Host)
	}
	if ls.SSHKeyRef != "" {
		t.Errorf("local server SSHKeyRef should be empty (local env), got %q", ls.SSHKeyRef)
	}

	stg, ok := inv.Environments["staging"]
	if !ok {
		t.Fatal("missing 'staging' environment")
	}
	if stg.Kind != "remote" {
		t.Errorf("staging.Kind: got %q, want remote", stg.Kind)
	}
	if len(stg.Servers) != 3 {
		t.Fatalf("staging.Servers len: got %d, want 3", len(stg.Servers))
	}

	lb := stg.Servers[2]
	if lb.Role != RoleLB {
		t.Errorf("lb server role: got %q, want lb", lb.Role)
	}
	if len(lb.Upstreams) != 1 || lb.Upstreams[0] != "stg-app" {
		t.Errorf("lb upstreams: got %v, want [stg-app]", lb.Upstreams)
	}
	// SSHKeyRef must hold an env-var name, never a literal path.
	if lb.SSHKeyRef != "NSELF_SSH_KEY_STAGING" {
		t.Errorf("lb SSHKeyRef: got %q, want NSELF_SSH_KEY_STAGING", lb.SSHKeyRef)
	}
}

// TestCapabilityConstants verifies the three Capability string values.
func TestCapabilityConstants(t *testing.T) {
	tests := []struct {
		cap  Capability
		want string
	}{
		{CapManage, "manage"},
		{CapReadOnly, "read-only"},
		{CapHidden, "hidden"},
	}
	for _, tc := range tests {
		if string(tc.cap) != tc.want {
			t.Errorf("Capability %q: got %q", tc.want, string(tc.cap))
		}
	}
}

// TestServerRoleConstants verifies all ServerRole string values.
func TestServerRoleConstants(t *testing.T) {
	tests := []struct {
		role ServerRole
		want string
	}{
		{RoleApp, "app"},
		{RoleLB, "lb"},
		{RoleObservability, "observability"},
		{RoleDB, "db"},
		{RoleWorker, "worker"},
	}
	for _, tc := range tests {
		if string(tc.role) != tc.want {
			t.Errorf("ServerRole %q: got %q", tc.want, string(tc.role))
		}
	}
}

// TestTargetStatusZeroValue verifies TargetStatus can be created with zero
// values without panicking and that time.Time is usable.
func TestTargetStatusZeroValue(t *testing.T) {
	ts := TargetStatus{
		Env:        "staging",
		Server:     "stg-app",
		Capability: CapManage,
		ProbedAt:   time.Now(),
	}
	if ts.Env != "staging" {
		t.Errorf("Env: got %q", ts.Env)
	}
	if ts.Capability != CapManage {
		t.Errorf("Capability: got %q", ts.Capability)
	}
}

// TestNoInlineSecrets verifies that Server fields contain no path-like values
// when parsed from the fixture (only env-var names in SSHKeyRef).
func TestNoInlineSecrets(t *testing.T) {
	var inv Inventory
	if err := yaml.Unmarshal([]byte(fixtureYAML), &inv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for envName, env := range inv.Environments {
		for _, s := range env.Servers {
			// SSHKeyRef must not look like a filesystem path.
			if len(s.SSHKeyRef) > 0 && s.SSHKeyRef[0] == '/' {
				t.Errorf("env %q server %q: SSHKeyRef %q looks like a path, must be env-var name",
					envName, s.Name, s.SSHKeyRef)
			}
		}
	}
}
