package doctor

// hardening_check.go — entry point for all SEC-HARDENING-*, SEC-CORS-01,
// SEC-OFFLINE-01, SEC-DEVMODE-01, and SEC-METRICS-01 security checks.
//
// Purpose: Aggregate entry point — HardeningChecks() returns the full slice of
//          12 security check results by delegating to focused sub-files.
// Inputs:  ctx context.Context for Docker/psql exec; projectDir string (nSelf
//          working directory, same value passed to DeepChecks).
// Outputs: []CheckResult — one result per check ID (SEC-HARDENING-01..08,
//          SEC-CORS-01, SEC-OFFLINE-01, SEC-DEVMODE-01, SEC-METRICS-01).
// Constraints: All checks run as part of `nself doctor --deep` (no license
//              required). Each returns Pass | Warn | Fail + remediation hint.
//              Sub-checks live in hardening_check_db.go (RLS, audit),
//              hardening_check_auth_net.go (rate-limit, MFA, SSRF, JWT, nginx),
//              hardening_check_infra.go (encryption, CORS, devmode, license,
//              metrics), and hardening_check_helpers.go (env file readers).
// SPORT:   cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import "context"

const hardeningSection = "security"

// HardeningChecks runs all SEC-HARDENING-*, SEC-CORS-01, SEC-OFFLINE-01,
// SEC-DEVMODE-01, and SEC-METRICS-01 checks.
// projectDir is the nSelf working directory (same value passed to DeepChecks).
func HardeningChecks(ctx context.Context, projectDir string) []CheckResult {
	return []CheckResult{
		checkHardeningRLS(ctx, projectDir),
		checkHardeningRateLimit(projectDir),
		checkHardeningMFAThrottle(projectDir),
		checkHardeningSSRFImport(projectDir),
		checkHardeningJWTPublicKeys(projectDir),
		checkHardeningNginxRateZones(ctx, projectDir),
		checkHardeningAuditLog(ctx, projectDir),
		checkHardeningEncryptionAtRest(projectDir),
		checkCORSDomain(projectDir),
		checkLicenseOffline(),
		checkHasuraDevMode(projectDir),
		checkMetricsPortBinding(projectDir),
	}
}
