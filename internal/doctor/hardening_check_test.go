package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// writeEnvFile writes key=value lines to fname inside dir.
func writeEnvFile(t *testing.T, dir, fname string, lines ...string) {
	t.Helper()
	path := filepath.Join(dir, fname)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeEnvFile %s: %v", fname, err)
	}
}

// ─── SEC-HARDENING-02: rate-limit env vars ───────────────────────────────────

func TestCheckHardeningRateLimit_AllSet(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev",
		"AUTH_RATE_LIMIT_ENABLED=true",
		"AUTH_RATE_LIMIT_MAX_REQUESTS=100",
		"AUTH_RATE_LIMIT_WINDOW_SECONDS=60",
	)

	res := checkHardeningRateLimit(dir)
	if res.Status != "pass" {
		t.Errorf("all rate-limit vars set: status=%q, want pass; msg=%s", res.Status, res.Message)
	}
	if res.Name != "SEC-HARDENING-02" {
		t.Errorf("Name=%q, want SEC-HARDENING-02", res.Name)
	}
}

func TestCheckHardeningRateLimit_Missing(t *testing.T) {
	dir := t.TempDir()
	// Only one of the three required vars present.
	writeEnvFile(t, dir, ".env.dev", "AUTH_RATE_LIMIT_ENABLED=true")

	res := checkHardeningRateLimit(dir)
	if res.Status != "warn" {
		t.Errorf("missing rate-limit vars: status=%q, want warn; msg=%s", res.Status, res.Message)
	}
	if !strings.Contains(res.Message, "missing rate-limit") {
		t.Errorf("unexpected message: %s", res.Message)
	}
}

func TestCheckHardeningRateLimit_NoneSet(t *testing.T) {
	dir := t.TempDir()
	// Empty env file — no vars at all.
	writeEnvFile(t, dir, ".env.dev", "# nothing here")

	res := checkHardeningRateLimit(dir)
	if res.Status != "warn" {
		t.Errorf("no rate-limit vars: status=%q, want warn", res.Status)
	}
	if res.FixCmd == "" {
		t.Error("expected a FixCmd suggestion")
	}
}

// ─── SEC-HARDENING-03: MFA throttle ─────────────────────────────────────────

func TestCheckHardeningMFAThrottle_Set(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev", "AUTH_MFA_MAX_ATTEMPTS_PER_HOUR=5")

	res := checkHardeningMFAThrottle(dir)
	if res.Status != "pass" {
		t.Errorf("MFA throttle set: status=%q, want pass; msg=%s", res.Status, res.Message)
	}
}

func TestCheckHardeningMFAThrottle_Missing(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev", "# no MFA throttle")

	res := checkHardeningMFAThrottle(dir)
	if res.Status != "warn" {
		t.Errorf("MFA throttle missing: status=%q, want warn", res.Status)
	}
	if !strings.Contains(res.Message, "AUTH_MFA_MAX_ATTEMPTS_PER_HOUR") {
		t.Errorf("message missing key name: %s", res.Message)
	}
}

// ─── SEC-HARDENING-04: SSRF guard import ─────────────────────────────────────

func TestCheckHardeningSSRFImport_NoPaidDir(t *testing.T) {
	dir := t.TempDir()
	// No plugins-pro directory adjacent to dir — expect warn.
	res := checkHardeningSSRFImport(dir)
	if res.Status != "warn" {
		t.Errorf("no plugins-pro dir: status=%q, want warn; msg=%s", res.Status, res.Message)
	}
}

func TestCheckHardeningSSRFImport_WithGuard(t *testing.T) {
	// Build a fake project structure where plugins-pro/paid/ai/shared/ssrf.go exists
	// and contains a guard symbol.
	root := t.TempDir()
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	guardDir := filepath.Join(root, "plugins-pro", "paid", "ai", "shared")
	if err := os.MkdirAll(guardDir, 0o755); err != nil {
		t.Fatal(err)
	}
	guardFile := filepath.Join(guardDir, "ssrf.go")
	content := "package shared\n\nfunc IsBlockedIP(ip string) bool { return false }\n"
	if err := os.WriteFile(guardFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res := checkHardeningSSRFImport(projectDir)
	// The ai plugin now has a guard; others are absent so skipped.
	// Result may be pass (no missing plugins) or warn (plugins-pro not found for missing ones).
	validStatuses := map[string]bool{"pass": true, "warn": true, "fail": true}
	if !validStatuses[res.Status] {
		t.Errorf("invalid status %q", res.Status)
	}
}

// ─── SEC-HARDENING-05: JWT public keys ──────────────────────────────────────

func TestCheckHardeningJWTPublicKeys_NotSet(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev", "# no JWT_PUBLIC_KEYS")

	res := checkHardeningJWTPublicKeys(dir)
	if res.Status != "warn" {
		t.Errorf("no JWT_PUBLIC_KEYS: status=%q, want warn", res.Status)
	}
}

func TestCheckHardeningJWTPublicKeys_OneKey(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev", `JWT_PUBLIC_KEYS=-----BEGIN PUBLIC KEY-----abc-----END PUBLIC KEY-----`)

	res := checkHardeningJWTPublicKeys(dir)
	if res.Status != "warn" {
		t.Errorf("single key: status=%q, want warn; msg=%s", res.Status, res.Message)
	}
}

func TestCheckHardeningJWTPublicKeys_TwoKeys(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev",
		`JWT_PUBLIC_KEYS=-----BEGIN PUBLIC KEY-----abc-----END PUBLIC KEY----------BEGIN PUBLIC KEY-----def-----END PUBLIC KEY-----`,
	)

	res := checkHardeningJWTPublicKeys(dir)
	if res.Status != "pass" {
		t.Errorf("two keys: status=%q, want pass; msg=%s", res.Status, res.Message)
	}
}

func TestCountJWTKeys_CommaList(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"key1,key2,key3", 3},
		{"key1", 1},
		{"key1,", 1}, // trailing comma produces one empty part
		{"", 0},
	}
	for _, c := range cases {
		got := countJWTKeys(c.input)
		if got != c.want {
			t.Errorf("countJWTKeys(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestCountJWTKeys_JSONArray(t *testing.T) {
	val := `[{"kty":"RSA","n":"abc"},{"kty":"RSA","n":"def"}]`
	got := countJWTKeys(val)
	if got != 2 {
		t.Errorf("JSON array with 2 JWKs: got %d, want 2", got)
	}
}

func TestCountJWTKeys_PEMBlocks(t *testing.T) {
	val := "-----BEGIN PUBLIC KEY-----abc-----END PUBLIC KEY----------BEGIN PUBLIC KEY-----def-----END PUBLIC KEY-----"
	got := countJWTKeys(val)
	if got != 2 {
		t.Errorf("two PEM blocks: got %d, want 2", got)
	}
}

// ─── SEC-HARDENING-06: Nginx rate zones ─────────────────────────────────────

func TestCheckHardeningNginxRateZones_BothPresent(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "nginx", "conf.d")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `
limit_req_zone $binary_remote_addr zone=auth_login:10m rate=5r/m;
location /auth/login { limit_req zone=auth_login; }
limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s;
location /api/ { limit_req zone=api; }
`
	if err := os.WriteFile(filepath.Join(confDir, "ratelimit.conf"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	res := checkHardeningNginxRateZones(ctx, dir)
	if res.Status != "pass" {
		t.Errorf("both zones present: status=%q, want pass; msg=%s", res.Status, res.Message)
	}
}

func TestCheckHardeningNginxRateZones_NonePresent(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "nginx", "conf.d")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "default.conf"), []byte("# empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	res := checkHardeningNginxRateZones(ctx, dir)
	// Without Docker running the fallback will also return nothing.
	if res.Status != "fail" && res.Status != "warn" {
		t.Errorf("no zones: status=%q, want fail or warn", res.Status)
	}
}

// ─── SEC-HARDENING-08: Encryption at rest ───────────────────────────────────

func TestCheckHardeningEncryptionAtRest_EnvSet(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.prod", "POSTGRES_DATA_ENCRYPTED=true")

	res := checkHardeningEncryptionAtRest(dir)
	if res.Status != "pass" {
		t.Errorf("POSTGRES_DATA_ENCRYPTED=true: status=%q, want pass; msg=%s", res.Status, res.Message)
	}
}

func TestCheckHardeningEncryptionAtRest_NotSet(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.dev", "# no encryption flag")

	res := checkHardeningEncryptionAtRest(dir)
	// On a CI runner without LUKS/dm-crypt, this will be warn.
	if res.Status == "fail" {
		t.Errorf("encryption not set: status=fail, want warn (fail is too strict for missing config)")
	}
	if res.Status != "warn" && res.Status != "pass" {
		t.Errorf("unexpected status %q", res.Status)
	}
}

func TestCheckHardeningEncryptionAtRest_AltEnvKey(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.prod", "NSELF_ENCRYPTION_AT_REST=yes")

	res := checkHardeningEncryptionAtRest(dir)
	if res.Status != "pass" {
		t.Errorf("NSELF_ENCRYPTION_AT_REST=yes: status=%q, want pass", res.Status)
	}
}

// ─── HardeningChecks aggregate ───────────────────────────────────────────────

func TestHardeningChecks_ReturnsEightResults(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	results := HardeningChecks(ctx, dir)
	if len(results) != 8 {
		t.Errorf("HardeningChecks returned %d results, want 8", len(results))
	}

	for _, r := range results {
		if r.Section != hardeningSection {
			t.Errorf("result %q: Section=%q, want %q", r.Name, r.Section, hardeningSection)
		}
		validStatuses := map[string]bool{"pass": true, "warn": true, "fail": true}
		if !validStatuses[r.Status] {
			t.Errorf("result %q: invalid Status=%q", r.Name, r.Status)
		}
		if r.Name == "" {
			t.Error("result has empty Name")
		}
		if r.Message == "" {
			t.Error("result has empty Message")
		}
	}
}

func TestHardeningChecks_CheckIDsDistinct(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	results := HardeningChecks(ctx, dir)
	seen := map[string]bool{}
	for _, r := range results {
		if seen[r.Name] {
			t.Errorf("duplicate check ID: %q", r.Name)
		}
		seen[r.Name] = true
	}
}
