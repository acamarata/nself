package doctor

// hardening_check_db.go — database-layer security checks for `nself doctor`.
//
// Purpose: SEC-HARDENING-01 (RLS enabled + FORCE RLS on all np_* tables) and
//          SEC-HARDENING-07 (audit log enabled with append-only trigger).
// Inputs:  ctx context.Context for docker exec / psql; projectDir string for
//          the nself-audit plugin list check.
// Outputs: CheckResult for each check — pass/warn/fail with remediation hint.
// Constraints: Both checks invoke docker exec against the project's Postgres
//              container (name resolved from PROJECT_NAME — see project_name.go)
//              for DB access.
//              checkAuditTrigger returns "pass"/"fail" (not CheckResult) so it
//              can be reused from two call sites in checkHardeningAuditLog.
//              auditTableName and auditTriggerName are exported as package-level
//              consts so tests can reference them directly.
// SPORT:   cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/health"
)

// auditTableName is the canonical table created by the nself-audit plugin
// migration (plugins-pro/paid/nself-audit/migrations/001_init.sql).
const auditTableName = "np_audit_events"

// auditTriggerName is the append-only trigger installed by 001_init.sql that
// blocks UPDATE and DELETE on np_audit_events to ensure tamper-evidence.
const auditTriggerName = "trg_audit_immutable"

// ─── SEC-HARDENING-01: RLS enabled + FORCE RLS on every np_* table ──────────

func checkHardeningRLS(ctx context.Context, projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-01"

	// Query pg_class for np_* tables that are missing RLS enable or force.
	query := `SELECT relname,
	       relrowsecurity::text   AS rls_enabled,
	       relforcerowsecurity::text AS rls_forced
	  FROM pg_class
	 WHERE relname LIKE 'np_%'
	   AND relkind = 'r'
	   AND (NOT relrowsecurity OR NOT relforcerowsecurity)
	 ORDER BY relname;`

	pgContainer := health.ContainerName(resolveProjectName(projectDir), "postgres")
	cmd := exec.CommandContext(ctx,
		"docker", "exec", pgContainer,
		"psql", "-U", "postgres", "-t", "-c", query,
	)
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-01: cannot query pg_class (%v) — ensure Postgres is running", err),
			FixCmd:  "nself start",
		}
	}

	// Trim whitespace; empty output means every np_* table has RLS+FORCE.
	violations := strings.TrimSpace(string(out))
	if violations == "" {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-HARDENING-01: all np_* tables have ENABLE and FORCE RLS",
		}
	}

	// Count violating tables.
	count := len(strings.Split(violations, "\n"))
	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "fail",
		Message: fmt.Sprintf("SEC-HARDENING-01: %d np_* table(s) missing ENABLE or FORCE RLS — run nself doctor --fix to remediate", count),
		FixCmd:  "nself doctor --fix --check SEC-HARDENING-01",
	}
}

// ─── SEC-HARDENING-07: Audit log enabled ─────────────────────────────────────

func checkHardeningAuditLog(ctx context.Context, projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-07"
	pgContainer := health.ContainerName(resolveProjectName(projectDir), "postgres")

	// Check 1: nself-audit plugin listed as installed.
	listCmd := exec.CommandContext(ctx, "nself", "plugin", "list", "--installed")
	listOut, listErr := listCmd.Output()
	if listErr == nil && strings.Contains(string(listOut), "nself-audit") {
		// Plugin is installed; also verify the append-only trigger is present
		// to confirm the tamper-evident guarantee is actually in place.
		triggerCheck := checkAuditTrigger(ctx, pgContainer)
		if triggerCheck == "pass" {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("SEC-HARDENING-07: nself-audit plugin installed, %s present with %s append-only trigger", auditTableName, auditTriggerName),
			}
		}
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-07: nself-audit plugin installed but %s trigger not found — migration may not have run", auditTriggerName),
			FixCmd:  "nself plugin run nself-audit migrate",
		}
	}

	// Check 2: np_audit_events table exists and is writable (INSERT privilege check).
	// The table is INSERT+SELECT only — UPDATE and DELETE are blocked by trigger.
	query := fmt.Sprintf(`SELECT has_table_privilege(current_user, '%s', 'INSERT')::text;`, auditTableName)
	cmd := exec.CommandContext(ctx,
		"docker", "exec", pgContainer,
		"psql", "-U", "postgres", "-t", "-c", query,
	)
	out, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(out)) == "t" {
		// Table writable — also verify append-only trigger.
		triggerCheck := checkAuditTrigger(ctx, pgContainer)
		if triggerCheck == "pass" {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("SEC-HARDENING-07: %s present, writable, and append-only trigger active", auditTableName),
			}
		}
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-07: %s writable but %s trigger missing — audit events not tamper-evident", auditTableName, auditTriggerName),
			FixCmd:  "nself plugin run nself-audit migrate",
		}
	}

	// Check 3: table exists but INSERT privilege not confirmed.
	existsQuery := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='%s')::text;`, auditTableName)
	existsCmd := exec.CommandContext(ctx,
		"docker", "exec", pgContainer,
		"psql", "-U", "postgres", "-t", "-c", existsQuery,
	)
	existsOut, existsErr := existsCmd.Output()
	if existsErr == nil && strings.TrimSpace(string(existsOut)) == "t" {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-07: %s table exists but INSERT privilege not confirmed — verify write access", auditTableName),
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "warn",
		Message: fmt.Sprintf("SEC-HARDENING-07: audit log not detected — install nself-audit plugin (table: %s)", auditTableName),
		FixCmd:  "nself plugin install nself-audit",
	}
}

// checkAuditTrigger verifies that the append-only trigger trg_audit_immutable
// exists on np_audit_events. Returns "pass" if confirmed, "fail" otherwise.
// pgContainer is the already-resolved Postgres container name (see
// resolveProjectName + health.ContainerName in the two call sites above) so
// this does not need to reload config on every call.
func checkAuditTrigger(ctx context.Context, pgContainer string) string {
	triggerQuery := fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM information_schema.triggers WHERE trigger_name='%s' AND event_object_table='%s')::text;`,
		auditTriggerName, auditTableName,
	)
	triggerCmd := exec.CommandContext(ctx,
		"docker", "exec", pgContainer,
		"psql", "-U", "postgres", "-t", "-c", triggerQuery,
	)
	triggerOut, triggerErr := triggerCmd.Output()
	if triggerErr == nil && strings.TrimSpace(string(triggerOut)) == "t" {
		return "pass"
	}
	return "fail"
}
