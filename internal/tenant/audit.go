package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
)

// AuditOptions holds flags for querying the audit log.
type AuditOptions struct {
	TenantID string
	Since    string // PostgreSQL interval, e.g. "7d" -> "7 days"
	Format   string // table or json
}

// Audit queries nself_ops.audit_log for a specific tenant.
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

	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	sinceClause := ""
	if opts.Since != "" {
		sinceClause = fmt.Sprintf(" AND created_at >= now() - interval '%s'", sanitize(parseDuration(opts.Since)))
	}

	sql := fmt.Sprintf(
		"SELECT json_agg(row_to_json(t)) FROM ("+
			"SELECT id, tenant_id, user_id, action, resource_type, resource_id, "+
			"ip::text, user_agent, payload_sha256, reason, created_at "+
			"FROM nself_ops.audit_log WHERE tenant_id = '%s'%s "+
			"ORDER BY created_at DESC LIMIT 1000) t;",
		sanitize(opts.TenantID), sinceClause,
	)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying audit log: %w", err)
	}

	raw := trimOutput(out)
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var entries []AuditEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("parsing audit log: %w", err)
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
