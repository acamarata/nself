package doctor

// hardening_check_nginx_fronted_test.go — table-driven coverage for the
// NGINX_FRONTED_BY handling in checkHardeningNginxRateZones.
//
// Purpose: prove SEC-HARDENING-06 audits the directory nginx actually
// serves on a fronted deployment (staging: nginx mounts the parent
// directory of the project dir, which is "backend"), instead of the
// project's own <projectDir>/nginx/** — a directory the running nginx
// never reads there. Before this fix the check silently scanned the wrong
// directory, so it could pass while the live config had no rate limiting
// (or fail while it did) without ever saying so.
// Inputs: synthetic project trees under t.TempDir(), each with a .env
// declaring NGINX_FRONTED_BY where the case is fronted.
// Outputs: pass/fail/warn/skip assertions plus a check that every result's
// Message names the directory that was actually audited (falsifiability).
// Constraints: no real machine paths — fixtures are built fresh per test.
// SPORT: cli/internal/doctor — closes the fronted-topology gap in
// SEC-HARDENING-06 (cli#380/cli#371 did not actually close this on
// staging).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nginxRateLimitFixture is a minimal conf.d file that satisfies both the
// auth and API rate-limit signals via the literal-path fallback signal
// (see scanNginxContentForRateZones).
const nginxRateLimitFixture = `
limit_req_zone $binary_remote_addr zone=auth_login:10m rate=5r/m;
location /auth/login { limit_req zone=auth_login; }
limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s;
location /api/ { limit_req zone=api; }
`

// messageNamesPath reports whether msg names path as evidence, tolerating
// both a literal embed (fmt %s, used by the pass/fail/warn messages) and a
// Go-quoted embed (fmt %q, used by the skip message). %q escapes backslash
// as a literal two-character sequence, so on Windows — where path itself
// contains single backslashes — a raw strings.Contains(msg, path) can never
// match a %q-embedded copy of the same path; only its escaped form appears
// in msg. This does not depend on which verb the source code happens to
// use today, so it stays correct if that changes.
func messageNamesPath(msg, path string) bool {
	if strings.Contains(msg, path) {
		return true
	}
	quoted := strings.Trim(fmt.Sprintf("%q", path), `"`)
	return strings.Contains(msg, quoted)
}

// writeNginxConf writes a single conf.d file containing content under
// <dir>/nginx/conf.d/ratelimit.conf, creating directories as needed.
func writeNginxConf(t *testing.T, dir, content string) {
	t.Helper()
	confDir := filepath.Join(dir, "nginx", "conf.d")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", confDir, err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "ratelimit.conf"), []byte(content), 0o644); err != nil {
		t.Fatalf("write ratelimit.conf: %v", err)
	}
}

func TestCheckHardeningNginxRateZones_FrontedTopology(t *testing.T) {
	tests := []struct {
		name string
		// setup builds the fixture tree and returns the projectDir to pass
		// to checkHardeningNginxRateZones, plus the directory-path
		// substring the result's Message must contain — the falsifiability
		// proof that the result names what it actually looked at.
		setup          func(t *testing.T) string
		wantStatus     string
		wantMsgSubstrs func(projectDir string) []string
	}{
		{
			// (a) non-fronted, rate limits present at <projectDir>/nginx -> pass.
			name: "non-fronted present passes",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNginxConf(t, dir, nginxRateLimitFixture)
				return dir
			},
			wantStatus: "pass",
			wantMsgSubstrs: func(projectDir string) []string {
				return []string{filepath.Join(projectDir, "nginx")}
			},
		},
		{
			// (b) non-fronted, rate limits absent -> fail (no Docker fallback
			// available in this test environment).
			name: "non-fronted absent fails",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeNginxConf(t, dir, "# empty\n")
				return dir
			},
			wantStatus: "fail",
			wantMsgSubstrs: func(projectDir string) []string {
				return []string{filepath.Join(projectDir, "nginx")}
			},
		},
		{
			// (c) THE REGRESSION THIS FIXES: fronted, parent dir (named
			// after the fronting stack) has the rate limits, backend/nginx
			// (the project dir) does not. This is the staging shape —
			// projectDir=".../nself-web/backend", nginx served from
			// ".../nself-web/nginx". Must PASS: the check must read the
			// parent, not the project's own (unserved) nginx dir.
			name: "fronted parent has limits backend does not passes",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				parent := filepath.Join(base, "nself-web")
				projectDir := filepath.Join(parent, "backend")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// t.Setenv (not a .env file): config.Load's godotenv.Overload
				// mutates process-wide env for keys present in a file, and
				// that mutation outlives the subtest (Go tests in one
				// package share a process) — it leaked NGINX_FRONTED_BY
				// into every later test in this file the first time this
				// used writeEnvFile. t.Setenv restores it automatically.
				t.Setenv("NGINX_FRONTED_BY", "nself-web")
				writeNginxConf(t, parent, nginxRateLimitFixture)     // served dir: has limits
				writeNginxConf(t, projectDir, "# empty, unserved\n") // backend/nginx: none
				return projectDir
			},
			wantStatus: "pass",
			wantMsgSubstrs: func(projectDir string) []string {
				// Must name the PARENT's nginx dir, not backend's.
				return []string{filepath.Join(filepath.Dir(projectDir), "nginx")}
			},
		},
		{
			// (d) PROVES IT STOPPED READING THE WRONG DIR: fronted, parent
			// lacks the limits, backend/nginx (unserved) has them. Must
			// FAIL — if the check were still reading backend/nginx this
			// would incorrectly pass.
			name: "fronted parent lacks limits backend has them fails",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				parent := filepath.Join(base, "nself-web")
				projectDir := filepath.Join(parent, "backend")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// t.Setenv (not a .env file): config.Load's godotenv.Overload
				// mutates process-wide env for keys present in a file, and
				// that mutation outlives the subtest (Go tests in one
				// package share a process) — it leaked NGINX_FRONTED_BY
				// into every later test in this file the first time this
				// used writeEnvFile. t.Setenv restores it automatically.
				t.Setenv("NGINX_FRONTED_BY", "nself-web")
				writeNginxConf(t, parent, "# empty, served dir has nothing\n")
				writeNginxConf(t, projectDir, nginxRateLimitFixture) // backend/nginx: has limits, but unserved
				return projectDir
			},
			wantStatus: "fail",
			wantMsgSubstrs: func(projectDir string) []string {
				return []string{filepath.Join(filepath.Dir(projectDir), "nginx")}
			},
		},
		{
			// (e) fronted but the directory cannot be resolved (parent dir
			// name does not match FrontedBy) -> skip/unknown, never a
			// silent pass on the wrong directory.
			name: "fronted unresolvable directory skips",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				parent := filepath.Join(base, "some-other-name")
				projectDir := filepath.Join(parent, "backend")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// t.Setenv (not a .env file): config.Load's godotenv.Overload
				// mutates process-wide env for keys present in a file, and
				// that mutation outlives the subtest (Go tests in one
				// package share a process) — it leaked NGINX_FRONTED_BY
				// into every later test in this file the first time this
				// used writeEnvFile. t.Setenv restores it automatically.
				t.Setenv("NGINX_FRONTED_BY", "nself-web")
				// Even if limits exist somewhere, they must not be trusted
				// since the directory cannot be confirmed.
				writeNginxConf(t, parent, nginxRateLimitFixture)
				return projectDir
			},
			wantStatus: "skip",
			wantMsgSubstrs: func(projectDir string) []string {
				return []string{"nself-web", projectDir}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := tt.setup(t)
			ctx := context.Background()
			res := checkHardeningNginxRateZones(ctx, projectDir)
			if res.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q; msg=%s", res.Status, tt.wantStatus, res.Message)
			}
			// Falsifiability: every result must name the directory it
			// actually audited (or, for skip, why it audited none) — a
			// check that can't produce this evidence is how the wrong-
			// directory bug stayed hidden through cli#380 and cli#371.
			for _, substr := range tt.wantMsgSubstrs(projectDir) {
				if !messageNamesPath(res.Message, substr) {
					t.Errorf("message does not name expected evidence %q: %s", substr, res.Message)
				}
			}
		})
	}
}
