package database

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// TestHasuraMetadataApplyCmd verifies that hasuraMetadataApplyCmd constructs
// the hasura-cli command without exposing the admin secret in argv (CWE-214).
// The hasura-cli path uses cmd.Env injection (HASURA_GRAPHQL_ADMIN_SECRET),
// not docker exec with a bare -e flag, so assertions (a)-(d) reflect that.
func TestHasuraMetadataApplyCmd(t *testing.T) {
	cases := []struct {
		name       string
		secret     string
		hasuraDir  string
		port       int
	}{
		{
			name:      "standard secret",
			secret:    "s3cur3_admin_secret_32charslong!!",
			hasuraDir: "/opt/nself/hasura",
			port:      8080,
		},
		{
			name:      "secret with special chars",
			secret:    "p@$$w0rd=&secret",
			hasuraDir: "/tmp/hasura-project",
			port:      9090,
		},
		{
			name:      "zero port falls back to 8080",
			secret:    "anothersecret_value_here_1234567",
			hasuraDir: "/hasura",
			port:      0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Hasura.AdminSecret = tc.secret
			cfg.Hasura.Port = tc.port

			cmd := hasuraMetadataApplyCmd(t.Context(), cfg, "hasura", tc.hasuraDir)

			// (a) Secret value must NOT appear as any argv element.
			for i, arg := range cmd.Args {
				if arg == tc.secret {
					t.Errorf("admin secret appears as plain argv element at index %d — CWE-214 violation", i)
				}
			}

			// (b) --admin-secret flag must not be in argv.
			for i, arg := range cmd.Args {
				if arg == "--admin-secret" || strings.HasPrefix(arg, "--admin-secret=") {
					t.Errorf("--admin-secret flag found at argv[%d]; secret must use env injection, got %q", i, arg)
				}
			}

			// (c) HASURA_GRAPHQL_ADMIN_SECRET=<value> must NOT appear in argv
			// (confirming no leakage through the argument list for the hasura-cli path).
			wantEnvKV := "HASURA_GRAPHQL_ADMIN_SECRET=" + tc.secret
			for i, arg := range cmd.Args {
				if arg == wantEnvKV {
					t.Errorf("env key=value appears in argv at index %d — secret must be in cmd.Env only", i)
				}
			}

			// (d) cmd.Env must contain "HASURA_GRAPHQL_ADMIN_SECRET=<secret>".
			envFound := false
			for _, e := range cmd.Env {
				if e == wantEnvKV {
					envFound = true
					break
				}
			}
			if !envFound {
				t.Errorf("cmd.Env does not contain %q", wantEnvKV)
			}
		})
	}
}
