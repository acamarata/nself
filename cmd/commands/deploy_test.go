package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/controlplane"
)

// ── T04 — deploy environments ─────────────────────────────────────────────────

// TestEnvServerRowJSON verifies the stable Admin-contract JSON schema for one
// server entry: the "reason" field must be omitted when empty.
func TestEnvServerRowJSON(t *testing.T) {
	row := envServerRow{Name: "app1", Role: "app", Capability: "manage"}
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if strings.Contains(s, `"reason"`) {
		t.Errorf("reason field must be omitted when empty; got %s", s)
	}
	for _, field := range []string{`"name"`, `"role"`, `"capability"`} {
		if !strings.Contains(s, field) {
			t.Errorf("expected field %s in JSON; got %s", field, s)
		}
	}
}

// TestDeployEnvironmentsCmdRegistered verifies that 'environments' is registered
// as a subcommand of deployCmd.
func TestDeployEnvironmentsCmdRegistered(t *testing.T) {
	for _, sub := range deployCmd.Commands() {
		if sub.Use == "environments" {
			return
		}
	}
	t.Error("'environments' subcommand not registered under deployCmd")
}

// TestDeployEnvironmentsOutputSchema verifies that deployEnvironmentsOutput
// marshals to the expected top-level JSON key.
func TestDeployEnvironmentsOutputSchema(t *testing.T) {
	out := deployEnvironmentsOutput{
		Environments: []envEnvironmentRow{
			{Name: "local", Kind: "local", Servers: []envServerRow{
				{Name: "local-app", Role: "app", Capability: "manage"},
			}},
		},
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"environments"`) {
		t.Errorf("expected top-level 'environments' key; got %s", s)
	}
	if !strings.Contains(s, `"local"`) {
		t.Errorf("expected environment name 'local'; got %s", s)
	}
}

// ── T05 — filterInventoryByServer / totalServers ──────────────────────────────

func makeTestInventory() *controlplane.Inventory {
	return &controlplane.Inventory{
		SchemaVersion: 1,
		Project:       "test",
		Environments: map[string]controlplane.Environment{
			"staging": {
				Name: "staging",
				Kind: "remote",
				Servers: []controlplane.Server{
					{Name: "staging-app", Role: controlplane.RoleApp, Primary: true},
					{Name: "staging-lb", Role: controlplane.RoleLB},
				},
			},
			"prod": {
				Name: "prod",
				Kind: "remote",
				Servers: []controlplane.Server{
					{Name: "prod-app", Role: controlplane.RoleApp, Primary: true},
				},
			},
		},
	}
}

func TestTotalServers(t *testing.T) {
	inv := makeTestInventory()
	if got := totalServers(inv); got != 3 {
		t.Errorf("totalServers: got %d, want 3", got)
	}
	empty := &controlplane.Inventory{
		Environments: map[string]controlplane.Environment{},
	}
	if got := totalServers(empty); got != 0 {
		t.Errorf("totalServers(empty): got %d, want 0", got)
	}
}

func TestFilterInventoryByServer_Found(t *testing.T) {
	inv := makeTestInventory()
	filtered := filterInventoryByServer(inv, "staging-app")
	if totalServers(filtered) != 1 {
		t.Errorf("expected 1 server after filter, got %d", totalServers(filtered))
	}
	env, ok := filtered.Environments["staging"]
	if !ok {
		t.Fatal("expected 'staging' environment to be retained")
	}
	if env.Servers[0].Name != "staging-app" {
		t.Errorf("expected server 'staging-app', got %q", env.Servers[0].Name)
	}
	// prod env should be gone (no match)
	if _, ok := filtered.Environments["prod"]; ok {
		t.Error("'prod' environment should be removed when only staging-app matches")
	}
}

func TestFilterInventoryByServer_NotFound(t *testing.T) {
	inv := makeTestInventory()
	filtered := filterInventoryByServer(inv, "nonexistent-server")
	if totalServers(filtered) != 0 {
		t.Errorf("expected 0 servers for missing name, got %d", totalServers(filtered))
	}
}

func TestFilterInventoryByServer_PreservesMetadata(t *testing.T) {
	inv := makeTestInventory()
	filtered := filterInventoryByServer(inv, "prod-app")
	if filtered.SchemaVersion != inv.SchemaVersion {
		t.Errorf("SchemaVersion not preserved: got %d, want %d", filtered.SchemaVersion, inv.SchemaVersion)
	}
	if filtered.Project != inv.Project {
		t.Errorf("Project not preserved: got %q, want %q", filtered.Project, inv.Project)
	}
}

// ── T07 — check-access deprecation ───────────────────────────────────────────

// TestDeployCheckAccessCmdDeprecation verifies the Long description contains the
// deprecation notice and pointer to the replacement command.
func TestDeployCheckAccessCmdDeprecation(t *testing.T) {
	long := deployCheckAccessCmd.Long
	if !strings.Contains(long, "Deprecated") {
		t.Error("deployCheckAccessCmd.Long must contain 'Deprecated'")
	}
	if !strings.Contains(long, "nself env target probe") {
		t.Error("deployCheckAccessCmd.Long must reference 'nself env target probe'")
	}
}

func TestResolveTarget(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"local", "local", false},
		{"staging", "staging", false},
		{"prod", "prod", false},
		{"production", "prod", false},
		{"PRODUCTION", "prod", false},
		{"  staging  ", "staging", false},
		{"preview", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := resolveTarget(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("resolveTarget(%q): expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("resolveTarget(%q): unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("resolveTarget(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDeployStrategies(t *testing.T) {
	for _, s := range []string{"rolling", "blue-green", "canary", "preview"} {
		if !deployStrategies[s] {
			t.Errorf("expected strategy %q to be allowed", s)
		}
	}
	for _, s := range []string{"", "recreate", "green"} {
		if deployStrategies[s] {
			t.Errorf("did not expect strategy %q to be allowed", s)
		}
	}
}

func TestStepStatus(t *testing.T) {
	if got := stepStatus(true, "done"); got != "pending" {
		t.Errorf("stepStatus(dryRun=true): got %q, want pending", got)
	}
	if got := stepStatus(false, "done"); got != "done" {
		t.Errorf("stepStatus(dryRun=false): got %q, want done", got)
	}
}
