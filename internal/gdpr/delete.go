package gdpr

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
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
			q := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", tbl.Table, tbl.UserCol)
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

	for _, tables := range strategies {
		for _, tbl := range tables {
			if err := applyErasure(ctx, db, tbl, userID); err != nil {
				errs = append(errs, err)
				continue
			}
			processed++
		}
	}
	return processed, errs
}

// applyErasure executes a single table's delete or anonymize strategy.
func applyErasure(ctx context.Context, db *sql.DB, tbl TableStrategy, userID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gdpr erase %s: begin tx: %w", tbl.Table, err)
	}
	defer tx.Rollback() //nolint:errcheck

	switch strings.ToLower(tbl.Strategy) {
	case "delete":
		q := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tbl.Table, tbl.UserCol)
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
`, tbl.Table, tbl.UserCol, tbl.UserCol)
		if _, err := tx.ExecContext(ctx, q, userID, anon); err != nil {
			// Fallback: some tables may not have all four PII columns.
			// Try a minimal anonymization.
			qMin := fmt.Sprintf("UPDATE %s SET %s = $2 WHERE %s = $1", tbl.Table, tbl.UserCol, tbl.UserCol)
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
