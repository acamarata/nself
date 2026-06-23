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

// ── E5-W1-S01-T01 — SSH injection fix tests ──────────────────────────────────

// TestSSHKeyPathValidation verifies that remoteDeployPush rejects SSH key paths
// containing shell metacharacters that could enable injection via the -e flag.
func TestSSHKeyPathValidation(t *testing.T) {
	cases := []struct {
		keyPath string
		valid   bool
		desc    string
	}{
		// Valid paths
		{"/home/user/.ssh/id_ed25519", true, "standard key path"},
		{"/root/.ssh/id_rsa", true, "root ssh dir"},
		{"id_ed25519", true, "relative path (filename)"},
		{"/usr/local/etc/ssh/key-2024", true, "key with hyphen"},
		{"/opt/keys/id_ed25519_prod", true, "key with underscore"},

		// Invalid paths with shell metacharacters
		{"/home/user/.ssh/id`date`", false, "backtick injection"},
		{"/home/user/.ssh/id;rm -rf /", false, "semicolon command separator"},
		{"/home/user/.ssh/id && whoami", false, "shell and operator"},
		{"/home/user/.ssh/id | cat /etc/passwd", false, "pipe operator"},
		{"/home/user/.ssh/id$(whoami)", false, "command substitution"},
		{"/home/user/.ssh/id${HOME}", false, "variable expansion"},
		{"/home/user/.ssh/id'", false, "quote character"},
		{"/home/user/.ssh/id\"", false, "double quote"},
		{"/home/user/.ssh/id*", false, "glob character"},
		{"/home/user/.ssh/id?", false, "glob question mark"},
		{"/home/user/.ssh/id[a]", false, "glob bracket"},
		{"/home/user/.ssh/id>out", false, "redirection"},
		{"/home/user/.ssh/id<in", false, "redirection input"},
	}

	for _, c := range cases {
		match := sshKeyPathRe.MatchString(c.keyPath)
		if match != c.valid {
			t.Errorf("sshKeyPathRe for %q (%s): got %v, want %v", c.keyPath, c.desc, match, c.valid)
		}
	}
}

// TestServiceNameValidation verifies that service names used in SSH commands are
// validated to prevent injection via docker compose service arguments.
func TestServiceNameValidation(t *testing.T) {
	cases := []struct {
		svcName string
		valid   bool
		desc    string
	}{
		// Valid service names
		{"postgres", true, "standard service"},
		{"hasura", true, "single word"},
		{"auth-service", true, "with hyphen"},
		{"auth_service", true, "with underscore"},
		{"auth123", true, "with numbers"},
		{"service-1_test", true, "mixed separators"},

		// Invalid service names with shell metacharacters
		{"postgres;rm", false, "semicolon command separator"},
		{"auth && whoami", false, "shell and operator"},
		{"service | cat", false, "pipe operator"},
		{"auth$(whoami)", false, "command substitution"},
		{"auth`date`", false, "backtick injection"},
		{"auth${VAR}", false, "variable expansion"},
		{"service/path", false, "path separator"},
		{"auth.test", false, "dot character"},
		{"service>file", false, "redirection"},
		{"auth<input", false, "redirection input"},
		{"auth`id`up", false, "backtick in middle"},
		{"auth;id;", false, "multiple commands"},
	}

	for _, c := range cases {
		match := svcNameRe.MatchString(c.svcName)
		if match != c.valid {
			t.Errorf("svcNameRe for %q (%s): got %v, want %v", c.svcName, c.desc, match, c.valid)
		}
	}
}
