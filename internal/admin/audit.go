// Package admin provides admin dashboard support: audit logging, ACL,
// multi-project management, and remote connect helpers.
package admin

// MigrationAdminAudit returns the SQL to create the np_admin_audit table.
// Every admin write mutation creates one row. Read-only reads are sampled
// at 1 % to keep volume sane.
func MigrationAdminAudit() string {
	return `-- np_admin_audit: immutable admin action audit trail
CREATE SCHEMA IF NOT EXISTS nself_ops;

CREATE TABLE IF NOT EXISTS nself_ops.np_admin_audit (
    id           BIGSERIAL    PRIMARY KEY,
    event_time   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    actor_email  TEXT         NOT NULL,
    actor_ip     INET         NOT NULL,
    method       TEXT         NOT NULL,
    path         TEXT         NOT NULL,
    body_hash    TEXT,
    result_code  INT          NOT NULL,
    duration_ms  INT          NOT NULL,
    session_id   UUID         NOT NULL
);

CREATE INDEX idx_np_admin_audit_event_time ON nself_ops.np_admin_audit (event_time);
CREATE INDEX idx_np_admin_audit_actor      ON nself_ops.np_admin_audit (actor_email);
CREATE INDEX idx_np_admin_audit_session    ON nself_ops.np_admin_audit (session_id);

-- INSERT-only: revoke UPDATE and DELETE from all roles
REVOKE UPDATE, DELETE ON nself_ops.np_admin_audit FROM PUBLIC;

COMMENT ON TABLE nself_ops.np_admin_audit IS 'Immutable admin audit trail. INSERT-only, no UPDATE or DELETE.';`
}

// AuditEntry represents a single row in np_admin_audit, used by the
// middleware and query helpers.
type AuditEntry struct {
	ActorEmail string
	ActorIP    string
	Method     string
	Path       string
	BodyHash   string
	ResultCode int
	DurationMs int
	SessionID  string
}

// InsertAuditSQL returns the parameterised INSERT statement.
func InsertAuditSQL() string {
	return `INSERT INTO nself_ops.np_admin_audit
    (actor_email, actor_ip, method, path, body_hash, result_code, duration_ms, session_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::uuid)`
}
