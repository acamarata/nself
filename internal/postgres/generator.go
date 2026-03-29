package postgres

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateInitScript creates the basic Postgres initialization SQL script
// containing the structural schemas required by Nhost containers.
func GenerateInitScript(workdir string) error {
	dir := filepath.Join(workdir, "postgres", "init")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating postgres init dir: %w", err)
	}

	content := `CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS storage;

-- Hasura internal schemas (required before Hasura starts)
CREATE SCHEMA IF NOT EXISTS hdb_catalog;
CREATE SCHEMA IF NOT EXISTS hdb_views;

-- Grant permissions on all schemas (supports non-superuser Hasura deployments)
DO $$ BEGIN
  EXECUTE format('GRANT ALL ON SCHEMA auth, storage, hdb_catalog, hdb_views, public TO %I', current_user);
  EXECUTE format('GRANT ALL ON ALL TABLES IN SCHEMA auth, storage, hdb_catalog, hdb_views, public TO %I', current_user);
  EXECUTE format('GRANT ALL ON ALL SEQUENCES IN SCHEMA auth, storage, hdb_catalog, hdb_views, public TO %I', current_user);
  EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA auth, storage, hdb_catalog, hdb_views, public GRANT ALL ON TABLES TO %I', current_user);
  EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA auth, storage, hdb_catalog, hdb_views, public GRANT ALL ON SEQUENCES TO %I', current_user);
END $$;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
`
	
	path := filepath.Join(dir, "01-init.sql")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing postgres init script: %w", err)
	}

	return nil
}
