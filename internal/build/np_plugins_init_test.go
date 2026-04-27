package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePluginManifest creates <pluginDir>/<name>/plugin.json with a minimal
// manifest. tierFlag controls whether we emit a paid manifest.
func writePluginManifest(t *testing.T, pluginDir, name string, body string) {
	t.Helper()
	dir := filepath.Join(pluginDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}
}

func TestGenerateNpPluginsSeed_EmptyDir(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := t.TempDir()

	out, err := GenerateNpPluginsSeed(workdir, pluginDir)
	if err != nil {
		t.Fatalf("GenerateNpPluginsSeed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "No plugins installed") {
		t.Errorf("expected empty-state comment in output; got:\n%s", body)
	}
	if strings.Contains(body, "INSERT INTO np_plugins") {
		t.Errorf("did not expect any INSERT for empty plugin dir; got:\n%s", body)
	}
}

func TestGenerateNpPluginsSeed_SeedsRows(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := t.TempDir()

	writePluginManifest(t, pluginDir, "auth", `{"name":"auth"}`)
	writePluginManifest(t, pluginDir, "ai", `{"name":"ai","isCommercial":true,"licenseType":"pro"}`)
	writePluginManifest(t, pluginDir, "internal-thing", `{"name":"internal-thing","visibility":"internal"}`)

	out, err := GenerateNpPluginsSeed(workdir, pluginDir)
	if err != nil {
		t.Fatalf("GenerateNpPluginsSeed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	body := string(data)

	wantLines := []string{
		"INSERT INTO np_plugins (name, tier) VALUES ('ai', 'pro') ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO np_plugins (name, tier) VALUES ('auth', 'free') ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO np_plugins (name, tier) VALUES ('internal-thing', 'internal') ON CONFLICT (name) DO NOTHING;",
	}
	for _, line := range wantLines {
		if !strings.Contains(body, line) {
			t.Errorf("missing expected SQL line:\n  %s\nfull body:\n%s", line, body)
		}
	}
}

// TestGenerateNpPluginsSeed_ReRunIsNoOp verifies the build is idempotent in
// the byte-equality sense: two runs back-to-back on the same plugin set
// produce the same SQL. This guards against ordering or timestamp leakage.
func TestGenerateNpPluginsSeed_ReRunIsNoOp(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := t.TempDir()

	writePluginManifest(t, pluginDir, "auth", `{"name":"auth"}`)
	writePluginManifest(t, pluginDir, "ai", `{"name":"ai","isCommercial":true}`)

	out1, err := GenerateNpPluginsSeed(workdir, pluginDir)
	if err != nil {
		t.Fatalf("first GenerateNpPluginsSeed: %v", err)
	}
	first, err := os.ReadFile(out1)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}

	out2, err := GenerateNpPluginsSeed(workdir, pluginDir)
	if err != nil {
		t.Fatalf("second GenerateNpPluginsSeed: %v", err)
	}
	if out2 != out1 {
		t.Errorf("output path changed across runs: %q vs %q", out1, out2)
	}
	second, err := os.ReadFile(out2)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("output differs between runs (idempotency broken)\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// TestGenerateNpPluginsSeed_SkipsDisabled verifies that plugins with a
// .disabled marker are excluded from the seed.
func TestGenerateNpPluginsSeed_SkipsDisabled(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := t.TempDir()

	writePluginManifest(t, pluginDir, "auth", `{"name":"auth"}`)
	writePluginManifest(t, pluginDir, "off", `{"name":"off"}`)
	if err := os.WriteFile(filepath.Join(pluginDir, "off", ".disabled"), []byte(""), 0o644); err != nil {
		t.Fatalf("write .disabled: %v", err)
	}

	out, err := GenerateNpPluginsSeed(workdir, pluginDir)
	if err != nil {
		t.Fatalf("GenerateNpPluginsSeed: %v", err)
	}
	data, _ := os.ReadFile(out)
	body := string(data)

	if !strings.Contains(body, "'auth'") {
		t.Errorf("expected 'auth' row to appear; got:\n%s", body)
	}
	if strings.Contains(body, "'off'") {
		t.Errorf("did not expect disabled plugin 'off' to be seeded; got:\n%s", body)
	}
}

func TestPluginTier_Classification(t *testing.T) {
	cases := []struct {
		name string
		in   *pluginSeedManifest
		want string
	}{
		{"nil-defaults-free", nil, "free"},
		{"plain-free", &pluginSeedManifest{Name: "x"}, "free"},
		{"commercial-true-is-pro", &pluginSeedManifest{Name: "x", IsCommercial: true}, "pro"},
		{"licenseType-pro-is-pro", &pluginSeedManifest{Name: "x", LicenseType: "pro"}, "pro"},
		{"visibility-internal-wins", &pluginSeedManifest{Name: "x", IsCommercial: true, Visibility: "internal"}, "internal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pluginTier(tc.in)
			if got != tc.want {
				t.Errorf("pluginTier: got %q want %q", got, tc.want)
			}
		})
	}
}
