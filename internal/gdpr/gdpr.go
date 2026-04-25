// Package gdpr provides GDPR data portability (Art. 20) and right-to-erasure
// (Art. 17) helpers used by the `nself gdpr` CLI commands.
//
// All operations are executed against the nSelf Postgres instance identified
// by the DATABASE_URL environment variable. Requests are logged to
// np_gdpr_requests for mandatory audit-trail purposes; that table is
// append-only (RLS blocks DELETE).
package gdpr

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// RequestType is the kind of GDPR operation being performed.
type RequestType string

const (
	RequestTypeExport   RequestType = "export"
	RequestTypeDelete   RequestType = "delete"
	RequestTypeRestrict RequestType = "restrict"
)

// SubjectType identifies whether the request targets a user or a whole tenant.
type SubjectType string

const (
	SubjectTypeUser   SubjectType = "user"
	SubjectTypeTenant SubjectType = "tenant"
)

// RequestStatus reflects the lifecycle state of a GDPR request.
type RequestStatus string

const (
	StatusPending    RequestStatus = "pending"
	StatusProcessing RequestStatus = "processing"
	StatusComplete   RequestStatus = "complete"
	StatusFailed     RequestStatus = "failed"
)

// GDPRRequest is a single row from np_gdpr_requests.
type GDPRRequest struct {
	ID              string        `json:"id"`
	TenantID        *string       `json:"tenant_id,omitempty"`
	RequestType     RequestType   `json:"request_type"`
	SubjectType     SubjectType   `json:"subject_type"`
	SubjectID       string        `json:"subject_id"`
	RequestedAt     time.Time     `json:"requested_at"`
	Deadline        time.Time     `json:"deadline"`
	Status          RequestStatus `json:"status"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	ArtifactURL     *string       `json:"artifact_url,omitempty"`
	ArtifactExpires *time.Time    `json:"artifact_expires,omitempty"`
	Notes           *string       `json:"notes,omitempty"`
}

// RegistryEntry is a plugin-registered table in np_gdpr_plugin_registry.
type RegistryEntry struct {
	ID           string          `json:"id"`
	PluginName   string          `json:"plugin_name"`
	UserTables   json.RawMessage `json:"user_tables"`
	TenantTables json.RawMessage `json:"tenant_tables"`
	RegisteredAt time.Time       `json:"registered_at"`
}

// TableStrategy describes what to do with a table row during erasure.
type TableStrategy struct {
	Table    string `json:"table"`
	UserCol  string `json:"user_col"`
	Strategy string `json:"strategy"` // "anonymize" or "delete"
}

// GDPRProvider is the interface third-party plugins must implement to
// participate in the cascade export and delete flows.
type GDPRProvider interface {
	UserTables() []TableStrategy
	AnonymizeUser(ctx context.Context, db *sql.DB, userID string) error
	ExportUser(ctx context.Context, db *sql.DB, userID string) ([]byte, error)
}

// DeadlineDays is the GDPR-mandated response window (Art. 17/20).
const DeadlineDays = 30

// CreateRequest inserts a new entry into np_gdpr_requests and returns the
// generated request ID. The caller is responsible for opening the connection.
func CreateRequest(ctx context.Context, db *sql.DB, rt RequestType, st SubjectType, subjectID string, tenantID *string) (string, error) {
	const q = `
INSERT INTO np_gdpr_requests
  (request_type, subject_type, subject_id, tenant_id, deadline)
VALUES
  ($1, $2, $3, $4, NOW() + INTERVAL '30 days')
RETURNING id`

	var id string
	err := db.QueryRowContext(ctx, q, string(rt), string(st), subjectID, tenantID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("gdpr: create request: %w", err)
	}
	return id, nil
}

// UpdateRequestStatus transitions a request to a new status and optionally
// sets artifact metadata (for export completions).
func UpdateRequestStatus(ctx context.Context, db *sql.DB, requestID string, status RequestStatus, artifactURL *string, artifactExpires *time.Time, notes *string) error {
	const q = `
UPDATE np_gdpr_requests
SET
  status           = $2,
  completed_at     = CASE WHEN $2 IN ('complete','failed') THEN NOW() ELSE completed_at END,
  artifact_url     = COALESCE($3, artifact_url),
  artifact_expires = COALESCE($4, artifact_expires),
  notes            = COALESCE($5, notes)
WHERE id = $1`

	_, err := db.ExecContext(ctx, q, requestID, string(status), artifactURL, artifactExpires, notes)
	if err != nil {
		return fmt.Errorf("gdpr: update status: %w", err)
	}
	return nil
}

// GetRequest returns the current state of a single GDPR request.
func GetRequest(ctx context.Context, db *sql.DB, requestID string) (*GDPRRequest, error) {
	const q = `
SELECT id, tenant_id, request_type, subject_type, subject_id,
       requested_at, deadline, status, completed_at,
       artifact_url, artifact_expires, notes
FROM np_gdpr_requests
WHERE id = $1`

	r := &GDPRRequest{}
	err := db.QueryRowContext(ctx, q, requestID).Scan(
		&r.ID, &r.TenantID, &r.RequestType, &r.SubjectType, &r.SubjectID,
		&r.RequestedAt, &r.Deadline, &r.Status, &r.CompletedAt,
		&r.ArtifactURL, &r.ArtifactExpires, &r.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("gdpr: get request: %w", err)
	}
	return r, nil
}

// ListRequests returns all GDPR requests, optionally filtered by status.
// Pass an empty string to return all statuses.
func ListRequests(ctx context.Context, db *sql.DB, statusFilter string) ([]*GDPRRequest, error) {
	q := `
SELECT id, tenant_id, request_type, subject_type, subject_id,
       requested_at, deadline, status, completed_at,
       artifact_url, artifact_expires, notes
FROM np_gdpr_requests`
	args := []interface{}{}
	if statusFilter != "" {
		q += " WHERE status = $1"
		args = append(args, statusFilter)
	}
	q += " ORDER BY requested_at DESC"

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("gdpr: list requests: %w", err)
	}
	defer rows.Close()

	var out []*GDPRRequest
	for rows.Next() {
		r := &GDPRRequest{}
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.RequestType, &r.SubjectType, &r.SubjectID,
			&r.RequestedAt, &r.Deadline, &r.Status, &r.CompletedAt,
			&r.ArtifactURL, &r.ArtifactExpires, &r.Notes,
		); err != nil {
			return nil, fmt.Errorf("gdpr: scan request: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RegistryTableStrategies returns the per-plugin table strategies for a given
// subject type ("user" or "tenant") from np_gdpr_plugin_registry.
func RegistryTableStrategies(ctx context.Context, db *sql.DB, subjectType SubjectType) (map[string][]TableStrategy, error) {
	col := "user_tables"
	if subjectType == SubjectTypeTenant {
		col = "tenant_tables"
	}

	q := fmt.Sprintf(`SELECT plugin_name, %s FROM np_gdpr_plugin_registry`, col)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("gdpr: registry query: %w", err)
	}
	defer rows.Close()

	out := map[string][]TableStrategy{}
	for rows.Next() {
		var pluginName string
		var raw json.RawMessage
		if err := rows.Scan(&pluginName, &raw); err != nil {
			return nil, fmt.Errorf("gdpr: scan registry: %w", err)
		}
		var tables []TableStrategy
		if err := json.Unmarshal(raw, &tables); err != nil {
			return nil, fmt.Errorf("gdpr: decode tables for %s: %w", pluginName, err)
		}
		out[pluginName] = tables
	}
	return out, rows.Err()
}
