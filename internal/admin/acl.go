package admin

// MigrationAdminACL returns the SQL to create the admin_acl table.
// Roles: owner, operator, viewer. A wildcard project ("*") grants
// access to every project.
func MigrationAdminACL() string {
	return `-- admin_acl: per-project access control for admin UI
CREATE SCHEMA IF NOT EXISTS nself_ops;

CREATE TABLE IF NOT EXISTS nself_ops.admin_acl (
    user_email TEXT NOT NULL,
    project    TEXT NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'operator', 'viewer')),
    PRIMARY KEY (user_email, project)
);

COMMENT ON TABLE nself_ops.admin_acl IS 'Per-project admin ACL. Wildcard project "*" grants access to all projects.';`
}

// ACLRole constants.
const (
	RoleOwner    = "owner"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// ACLEntry represents one row in admin_acl.
type ACLEntry struct {
	UserEmail string
	Project   string
	Role      string
}

// CheckACLSQL returns a query that resolves whether a user may access
// a given project. It accounts for the wildcard ("*") entry.
func CheckACLSQL() string {
	return `SELECT role FROM nself_ops.admin_acl
WHERE user_email = $1
  AND (project = $2 OR project = '*')
ORDER BY CASE WHEN project = '*' THEN 1 ELSE 0 END
LIMIT 1`
}

// ListProjectsForUserSQL returns all projects a user may access.
func ListProjectsForUserSQL() string {
	return `SELECT project, role FROM nself_ops.admin_acl
WHERE user_email = $1
ORDER BY project`
}

// SeedOwnerSQL returns the INSERT for the wildcard owner row.
func SeedOwnerSQL(email string) string {
	return "INSERT INTO nself_ops.admin_acl (user_email, project, role) VALUES ('" +
		email + "', '*', 'owner') ON CONFLICT DO NOTHING"
}

// SeedOperatorSQL returns the INSERT for a single-project operator.
func SeedOperatorSQL(email, project string) string {
	return "INSERT INTO nself_ops.admin_acl (user_email, project, role) VALUES ('" +
		email + "', '" + project + "', 'operator') ON CONFLICT DO NOTHING"
}
