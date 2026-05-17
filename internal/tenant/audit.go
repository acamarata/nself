package tenant

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// AuditOptions holds flags for querying the audit log.
type AuditOptions struct {
	TenantID string
	Since    string // PostgreSQL interval, e.g. "7d" -> "7 days"
	Format   string // table or json
}

// Audit queries nself_ops.audit_log for a specific tenant.
// Uses a direct database/sql connection with $N parameterized queries (Path A).
func Audit(ctx context.Context, cfg *config.Config, opts AuditOptions) ([]AuditEntry, error) {
	if opts.TenantID == "" {
		return nil, fmt.Errorf("tenant-id is required")
	}
	if err := validateUUID(opts.TenantID); err != nil {
		return nil, err
	}
	if opts.Since != "" {
		if err := validateDuration(opts.Since); err != nil {
			return nil, err
		}
	}

	sqlDB, err := openTenantDB(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("querying audit log: %w", err)
	}
	defer sqlDB.Close()

	// Build query with $N placeholders. The since interval is passed as a
	// PostgreSQL interval string derived from parseDuration (digits+unit only,
	// validated by validateDuration — no user-controlled characters remain).
	// The interval value itself is also bound as $2 to avoid any interpolation.
	args := []interface{}{opts.TenantID}
	query := `SELECT id, tenant_id::text, user_id::text, action, resource_type,
		resource_id, ip::text, user_agent, payload_sha256, reason, created_at
		FROM nself_ops.audit_log WHERE tenant_id = $1`

	if opts.Since != "" {
		// parseDuration converts e.g. "7d" -> "7 days"; validateDuration ensures
		// the input is digits+unit only, so the resulting string is safe. It is
		// still passed as a bound parameter ($2) for defence-in-depth.
		args = append(args, parseDuration(opts.Since))
		query += fmt.Sprintf(" AND created_at >= now() - $%d::interval", len(args))
	}
	query += " ORDER BY created_at DESC LIMIT 1000"

	rows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var userID, resourceID, ipText, userAgent, payloadSHA, reason sql.NullString
		var createdAt time.Time
		if err := rows.Scan(
			&e.ID, &e.TenantID, &userID, &e.Action, &e.ResourceType,
			&resourceID, &ipText, &userAgent, &payloadSHA, &reason, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scanning audit row: %w", err)
		}
		e.UserID = userID.String
		e.ResourceID = resourceID.String
		e.IP = ipText.String
		e.UserAgent = userAgent.String
		e.PayloadSHA256 = payloadSHA.String
		e.Reason = reason.String
		e.CreatedAt = createdAt
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit rows: %w", err)
	}

	return entries, nil
}

// parseDuration converts shorthand like "7d" to PostgreSQL interval "7 days".
func parseDuration(s string) string {
	if len(s) == 0 {
		return "7 days"
	}
	last := s[len(s)-1]
	num := s[:len(s)-1]
	switch last {
	case 'd':
		return num + " days"
	case 'h':
		return num + " hours"
	case 'w':
		return num + " weeks"
	case 'm':
		return num + " months"
	default:
		return s + " days"
	}
}
