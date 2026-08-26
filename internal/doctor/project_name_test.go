package doctor

// project_name_test.go — regression coverage for resolveProjectName and the
// container names derived from it.
//
// Before this fix, DockerDeepChecks' Postgres/nginx siblings and the
// SEC-HARDENING-01/06/07 checks hardcoded "nself_postgres"/"nself_nginx" and
// could never pass on a project with a custom PROJECT_NAME (confirmed on
// nself-org/web, PROJECT_NAME=nself-web — see nself-org/web#127). These tests
// pin down the three cases that matter: a custom PROJECT_NAME resolves to its
// own container names, an unconfigured project still resolves to config's
// real default ("myproject", not the old "nself" literal — see
// internal/config/defaults.go applyDefaultsCore), and a project whose config
// cannot be loaded at all falls back to the pre-fix "nself" literal so
// setups that only ever worked because of that hardcoding are undisturbed.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/health"
)

func TestResolveProjectName_ContainerNames(t *testing.T) {
	cases := []struct {
		name         string
		setup        func(t *testing.T, dir string) // writes (or omits) project config
		wantProject  string
		wantPostgres string
		wantNginx    string
	}{
		{
			name: "custom project name resolves to its own containers",
			setup: func(t *testing.T, dir string) {
				writeTestEnv(t, dir, "PROJECT_NAME=myproj\n")
			},
			wantProject:  "myproj",
			wantPostgres: "myproj_postgres",
			wantNginx:    "myproj_nginx",
		},
		{
			name:         "unset project name resolves to config's real default, not the old literal",
			setup:        func(t *testing.T, dir string) {}, // no .env at all
			wantProject:  "myproject",
			wantPostgres: "myproject_postgres",
			wantNginx:    "myproject_nginx",
		},
		{
			name: "unloadable config falls back to the legacy hardcoded name",
			setup: func(t *testing.T, dir string) {
				// A NUL byte makes config.Load reject the file outright
				// (internal/config/loader.go's maxEnvFileSize/null-byte guard),
				// simulating "project name cannot be resolved at all".
				content := []byte("PROJECT_NAME=x\x00")
				if err := os.WriteFile(filepath.Join(dir, ".env"), content, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantProject:  legacyProjectName,
			wantPostgres: "nself_postgres",
			wantNginx:    "nself_nginx",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Isolate from whatever PROJECT_NAME the host shell/CI runner may
			// already export; t.Setenv restores the prior value on cleanup.
			t.Setenv("PROJECT_NAME", "")

			dir := t.TempDir()
			tc.setup(t, dir)

			got := resolveProjectName(dir)
			if got != tc.wantProject {
				t.Errorf("resolveProjectName(%q) = %q, want %q", dir, got, tc.wantProject)
			}
			if pg := health.ContainerName(got, "postgres"); pg != tc.wantPostgres {
				t.Errorf("postgres container name = %q, want %q", pg, tc.wantPostgres)
			}
			if ng := health.ContainerName(got, "nginx"); ng != tc.wantNginx {
				t.Errorf("nginx container name = %q, want %q", ng, tc.wantNginx)
			}
		})
	}
}

// writeTestEnv writes content to a .env file in dir, failing the test on error.
func writeTestEnv(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
