package config

// Purpose: Regression test for gap #1 — real-world env var names declared by
//          an actual app (ntask/backend/.env.example) must all be present in
//          knownEnvVars, or userDefinedPrefixes, so `nself build` does not
//          emit false "unknown env var" warnings for legitimate config.
//
// Inputs:  none (static list mirrors the real ntask .env.example var names,
//          captured at fix time — update this list if ntask's env schema
//          changes).
// Outputs: none (t.Error per missing var).
//
// Constraints: This is a snapshot regression guard, not living documentation
//              of ntask's env file — if ntask adds new vars, this test will
//              not catch it until someone updates realWorldAppEnvVars below.
//
// SPORT: cli/internal/config — gap #1 fix (env-schema parity ticket).

import "testing"

// realWorldAppEnvVars mirrors every var declared in
// nself-org/ntask backend/.env.example at the time of the gap #1 fix. Real
// app repos are the ground truth for "legitimate config surface" — this list
// exists so a future edit to knownEnvVars that accidentally drops one of
// these entries fails CI immediately instead of silently reintroducing false
// warnings for ntask (and any other app with a similar env surface).
var realWorldAppEnvVars = []string{
	"ACTION_HANDLER_URL",
	"APNS_KEY_ID", "APNS_KEY_PATH", "APNS_TEAM_ID",
	"AUTH_ACCESS_TOKEN_EXPIRES_IN", "AUTH_ANONYMOUS_USERS_ENABLED",
	"AUTH_CLIENT_URL", "AUTH_DISABLE_NEW_USERS",
	"AUTH_EMAIL_PASSWORDLESS_ENABLED", "AUTH_EMAIL_TEMPLATE_FETCH_URL",
	"AUTH_GRAVATAR_ENABLED", "AUTH_JWT_SECRET", "AUTH_LOCALE_DEFAULT",
	"AUTH_PORT", "AUTH_REFRESH_TOKEN_EXPIRES_IN",
	"AUTH_REQUIRE_EMAIL_VERIFICATION", "AUTH_SERVER_URL",
	"BACKUP_ACCESS_KEY", "BACKUP_RETENTION_DAYS", "BACKUP_S3_BUCKET",
	"BACKUP_S3_ENDPOINT", "BACKUP_S3_PREFIX", "BACKUP_SECRET_KEY",
	"DEMO_SEED", "DR_DATABASE_URL", "FCM_SERVER_KEY",
	"FUNCTIONS_ENABLED", "FUNCTIONS_PORT",
	"HASURA_ADMIN_SECRET", "HASURA_AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED",
	"HASURA_AUTH_SMTP_HOST", "HASURA_AUTH_SMTP_PASS", "HASURA_AUTH_SMTP_PORT",
	"HASURA_AUTH_SMTP_SECURE", "HASURA_AUTH_SMTP_SENDER", "HASURA_AUTH_SMTP_USER",
	"HASURA_AUTH_URL", "HASURA_GRAPHQL_BATCH_SIZE", "HASURA_GRAPHQL_DEPTH_LIMIT",
	"HASURA_GRAPHQL_DEV_MODE", "HASURA_GRAPHQL_ENABLE_ALLOWLIST",
	"HASURA_GRAPHQL_ENABLE_CONSOLE", "HASURA_GRAPHQL_ENABLE_TELEMETRY",
	"HASURA_GRAPHQL_LIVE_QUERIES_MULTIPLEXED_BATCH_SIZE",
	"HASURA_GRAPHQL_LIVE_QUERIES_MULTIPLEXED_REFETCH_INTERVAL",
	"HASURA_GRAPHQL_NODE_LIMIT", "HASURA_GRAPHQL_UNAUTHORIZED_ROLE",
	"HASURA_PORT",
	"MAILHOG_SMTP_PORT", "MAILHOG_UI_PORT",
	"MINIO_ACCESS_KEY", "MINIO_BUCKET", "MINIO_CONSOLE_PORT",
	"MINIO_ENDPOINT", "MINIO_ENDPOINT_PUBLIC", "MINIO_PORT", "MINIO_SECRET_KEY",
	"POSTGRES_DB", "POSTGRES_PASSWORD", "POSTGRES_PORT", "POSTGRES_USER",
	"RATE_LIMIT_AUTH_RPM", "RATE_LIMIT_GRAPHQL_RPM", "RATE_LIMIT_UPLOADS_RPM",
	"S3_ACCESS_KEY", "S3_BUCKET", "S3_ENDPOINT", "S3_SECRET_KEY",
	"SENTRY_DSN_BACKEND",
	"SMTP_HOST", "SMTP_PASS", "SMTP_PORT", "SMTP_SECURE", "SMTP_SENDER", "SMTP_USER",
	"STORAGE_PORT", "STORAGE_PUBLIC_URL",
	"DOMAIN", "ACME_EMAIL", "AUTH_MODE",
	"SHARED_AUTH_JWT_SECRET", "SHARED_AUTH_SERVER_URL",
}

// TestKnownEnvVars_CoversRealWorldAppSurface verifies that every var in
// realWorldAppEnvVars is present in knownEnvVars (or would be skipped by a
// userDefinedPrefix — none of these currently match one, but the check is
// included for completeness/future-proofing).
func TestKnownEnvVars_CoversRealWorldAppSurface(t *testing.T) {
	knownSet := make(map[string]bool, len(knownEnvVars))
	for _, k := range knownEnvVars {
		knownSet[k] = true
	}

	var missing []string
	for _, v := range realWorldAppEnvVars {
		if knownSet[v] {
			continue
		}
		skippedByPrefix := false
		for _, pfx := range userDefinedPrefixes {
			if len(v) >= len(pfx) && v[:len(pfx)] == pfx {
				skippedByPrefix = true
				break
			}
		}
		if !skippedByPrefix {
			missing = append(missing, v)
		}
	}

	if len(missing) > 0 {
		t.Errorf("knownEnvVars is missing %d real-world app env vars (would trigger false 'unknown env var' warnings): %v", len(missing), missing)
	}
}
