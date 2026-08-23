package gdpr

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/database"
)

// DryRunResult lists what would be deleted/anonymized without making changes.
type DryRunResult struct {
	Plugin   string
	Table    string
	UserCol  string
	Strategy string
	RowCount int64
}

// DryRunUserDelete counts affected rows per table without deleting anything.
// Returns a list of results suitable for display before the user confirms.
func DryRunUserDelete(ctx context.Context, db *sql.DB, userID string) ([]DryRunResult, error) {
	strategies, err := RegistryTableStrategies(ctx, db, SubjectTypeUser)
	if err != nil {
		return nil, fmt.Errorf("gdpr dry-run: %w", err)
	}
	strategies["core"] = coreUserErasureTables()

	var results []DryRunResult
	for pluginName, tables := range strategies {
		for _, tbl := range tables {
			qtbl, qcol, err := validateTableStrategy(tbl)
			if err != nil {
				// Reject registry entries with invalid identifiers — do not silently skip.
				return nil, fmt.Errorf("gdpr dry-run: plugin %s: %w", pluginName, err)
			}
			q := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", qtbl, qcol)
			var n int64
			if err := db.QueryRowContext(ctx, q, userID).Scan(&n); err != nil {
				// Table may not exist in this deployment; skip gracefully.
				continue
			}
			results = append(results, DryRunResult{
				Plugin:   pluginName,
				Table:    tbl.Table,
				UserCol:  tbl.UserCol,
				Strategy: tbl.Strategy,
				RowCount: n,
			})
		}
	}
	return results, nil
}

// DeleteUserData executes the cascading delete/anonymization across all
// plugin-registered tables. Each table is processed in its own transaction
// to limit blast radius from individual failures.
//
// Returns the count of tables successfully processed and a slice of any
// per-table errors (non-fatal; logged to request notes).
func DeleteUserData(ctx context.Context, db *sql.DB, userID string) (processed int, errs []error) {
	strategies, err := RegistryTableStrategies(ctx, db, SubjectTypeUser)
	if err != nil {
		return 0, []error{fmt.Errorf("gdpr delete: registry: %w", err)}
	}
	strategies["core"] = coreUserErasureTables()

	for pluginName, tables := range strategies {
		for _, tbl := range tables {
			if _, _, err := validateTableStrategy(tbl); err != nil {
				errs = append(errs, fmt.Errorf("gdpr delete: plugin %s: %w", pluginName, err))
				continue
			}
			if err := applyErasure(ctx, db, tbl, userID); err != nil {
				errs = append(errs, err)
				continue
			}
			processed++
		}
	}
	return processed, errs
}

// validateTableStrategy validates that a TableStrategy's Table and UserCol
// fields are safe SQL identifiers before they are interpolated into queries.
// Table names registered by plugins come from the database (attacker-influenced
// if a malicious plugin inserts to np_gdpr_plugin_registry directly), so they
// MUST be validated against the identifier allowlist.
//
// Returns the double-quoted, injection-safe forms of the table and column, or
// an error if either value is not a valid SQL identifier.
//
// Inputs:  tbl — the TableStrategy read from the plugin registry
// Outputs: qtbl, qcol — double-quoted safe identifiers; err on invalid input
// Constraints: rejects empty string, SQL metacharacters, values >64 chars per
//
//	RFC PostgreSQL identifier limit, and any char outside [a-zA-Z0-9_]
//
// SPORT: MASTER-FEATURES.md — SQL injection hardening — cli
func validateTableStrategy(tbl TableStrategy) (qtbl, qcol string, err error) {
	qtbl, err = database.SanitizeIdentifier(tbl.Table)
	if err != nil {
		return "", "", fmt.Errorf("unsafe table identifier from registry: %w", err)
	}
	qcol, err = database.SanitizeIdentifier(tbl.UserCol)
	if err != nil {
		return "", "", fmt.Errorf("unsafe column identifier from registry: %w", err)
	}
	return qtbl, qcol, nil
}

// applyErasure executes a single table's delete or anonymize strategy.
// Callers MUST have already validated tbl via validateTableStrategy before
// calling this function. applyErasure uses the raw tbl values for identifier
// quoting but re-validates defensively via validateTableStrategy so the
// function is safe even if called directly in tests.
func applyErasure(ctx context.Context, db *sql.DB, tbl TableStrategy, userID string) error {
	// Defensive re-validation — identifiers were checked by callers but this
	// ensures applyErasure is safe even when called directly (e.g. from tests
	// or future callers that skip the outer loop validation).
	qtbl, qcol, err := validateTableStrategy(tbl)
	if err != nil {
		return fmt.Errorf("gdpr erase: %w", err)
	}

	tx, txErr := db.BeginTx(ctx, nil)
	if txErr != nil {
		return fmt.Errorf("gdpr erase %s: begin tx: %w", tbl.Table, txErr)
	}
	defer tx.Rollback() //nolint:errcheck

	switch strings.ToLower(tbl.Strategy) {
	case "delete":
		q := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", qtbl, qcol)
		if _, err := tx.ExecContext(ctx, q, userID); err != nil {
			return fmt.Errorf("gdpr erase %s (delete): %w", tbl.Table, err)
		}

	case "anonymize", "":
		// Replace PII columns with pseudonymous values rather than deleting
		// rows entirely. This preserves referential integrity for aggregate
		// analytics while removing personal data.
		anon := anonymizedUserID(userID)
		q := fmt.Sprintf(`
UPDATE %s SET
  %s         = $2,
  email      = CASE WHEN email    IS NOT NULL THEN 'deleted@gdpr.invalid' ELSE NULL END,
  name       = CASE WHEN name     IS NOT NULL THEN 'Deleted User' ELSE NULL END,
  first_name = CASE WHEN first_name IS NOT NULL THEN 'Deleted' ELSE NULL END,
  last_name  = CASE WHEN last_name  IS NOT NULL THEN 'User' ELSE NULL END
WHERE %s = $1
`, qtbl, qcol, qcol)
		if _, err := tx.ExecContext(ctx, q, userID, anon); err != nil {
			// Fallback: some tables may not have all four PII columns.
			// Try a minimal anonymization.
			qMin := fmt.Sprintf("UPDATE %s SET %s = $2 WHERE %s = $1", qtbl, qcol, qcol)
			if _, err2 := tx.ExecContext(ctx, qMin, userID, anon); err2 != nil {
				return fmt.Errorf("gdpr erase %s (anonymize): %w", tbl.Table, err2)
			}
		}

	default:
		return fmt.Errorf("gdpr erase %s: unknown strategy %q", tbl.Table, tbl.Strategy)
	}

	return tx.Commit()
}

// anonymizedUserID returns a deterministic pseudonym for audit-log continuity.
// It deliberately retains the "gdpr-erased-" prefix so rows can be identified.
func anonymizedUserID(originalID string) string {
	if len(originalID) > 8 {
		return "gdpr-erased-" + originalID[:8]
	}
	return "gdpr-erased-" + originalID
}

// coreUserErasureTables returns built-in nSelf tables covered by default
// erasure independent of the plugin registry.
func coreUserErasureTables() []TableStrategy {
	return []TableStrategy{
		{Table: "np_users", UserCol: "id", Strategy: "anonymize"},
		{Table: "np_user_sessions", UserCol: "user_id", Strategy: "delete"},
		{Table: "np_user_metadata", UserCol: "user_id", Strategy: "delete"},
	}
}

// CheckDeadlines scans np_gdpr_requests for pending requests approaching or
// past the 30-day GDPR deadline. Returns warning and breach lists.
func CheckDeadlines(ctx context.Context, db *sql.DB) (warnings []*GDPRRequest, breaches []*GDPRRequest, err error) {
	const q = `
SELECT id, tenant_id, request_type, subject_type, subject_id,
       requested_at, deadline, status, completed_at,
       artifact_url, artifact_expires, notes
FROM np_gdpr_requests
WHERE status NOT IN ('complete','failed')
  AND deadline <= NOW() + INTERVAL '7 days'
ORDER BY deadline ASC`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, nil, fmt.Errorf("gdpr deadlines: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		r := &GDPRRequest{}
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.RequestType, &r.SubjectType, &r.SubjectID,
			&r.RequestedAt, &r.Deadline, &r.Status, &r.CompletedAt,
			&r.ArtifactURL, &r.ArtifactExpires, &r.Notes,
		); err != nil {
			return nil, nil, fmt.Errorf("gdpr deadlines scan: %w", err)
		}
		if r.Deadline.Before(now) {
			breaches = append(breaches, r)
		} else {
			warnings = append(warnings, r)
		}
	}
	return warnings, breaches, rows.Err()
}
