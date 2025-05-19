#!/bin/bash

# nself-seed.sh - Handles the creation of seeds/initdb.sql for initializing the Postgres database

set -e

# ----------------------------
# Variables
# ----------------------------
SEEDS_DIR="$PWD/seeds"  # Ensure seeds/initdb.sql is created in the project directory
INITDB_FILE="$SEEDS_DIR/initdb.sql"
EXPECTED_CONTENT=$(cat <<'EOF'
-- Create Roles
DO $$
BEGIN
   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nhost_hasura') THEN
      CREATE ROLE nhost_hasura WITH LOGIN PASSWORD 'nhost_hasura_password';
   END IF;

   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nhost_auth_admin') THEN
      CREATE ROLE nhost_auth_admin WITH LOGIN PASSWORD 'nhost_auth_admin_password';
   END IF;

   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nhost_storage_admin') THEN
      CREATE ROLE nhost_storage_admin WITH LOGIN PASSWORD 'nhost_storage_admin_password';
   END IF;
END
$$;

-- Create Schema
DO $$
BEGIN
   IF NOT EXISTS (SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'hdb_catalog') THEN
      CREATE SCHEMA hdb_catalog;
   END IF;
END
$$;

-- Enable TimescaleDB
DO $$
BEGIN
   IF NOT EXISTS (SELECT * FROM pg_available_extensions WHERE name = 'timescaledb') THEN
      CREATE EXTENSION timescaledb CASCADE;
   END IF;
END
$$;

-- Grant Permissions
GRANT postgres TO nhost_hasura;

GRANT USAGE ON SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT CREATE ON SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT ALL ON ALL TABLES IN SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT ALL ON ALL SEQUENCES IN SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA hdb_catalog TO nhost_auth_admin;

GRANT USAGE ON SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT CREATE ON SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT ALL ON ALL TABLES IN SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT ALL ON ALL SEQUENCES IN SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA hdb_catalog TO nhost_storage_admin;

-- Default Privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON TABLES TO nhost_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON SEQUENCES TO nhost_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON FUNCTIONS TO nhost_auth_admin;

ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON TABLES TO nhost_storage_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON SEQUENCES TO nhost_storage_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON FUNCTIONS TO nhost_storage_admin;
EOF
)

# ----------------------------
# Helper Functions
# ----------------------------

# Function to print informational messages
echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

# Function to print error messages
echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

# Function to create seeds/initdb.sql
create_initdb_sql() {
  # Create seeds directory if it doesn't exist
  if [ ! -d "$SEEDS_DIR" ]; then
    echo_info "Creating seeds directory at $SEEDS_DIR..."
    mkdir -p "$SEEDS_DIR"
  fi

  # Check if initdb.sql exists
  if [ -f "$INITDB_FILE" ]; then
    echo_info "initdb.sql already exists in $SEEDS_DIR. Verifying content..."

    # Compare existing content with expected content
    EXISTING_CONTENT=$(cat "$INITDB_FILE")
    if [ "$EXISTING_CONTENT" = "$EXPECTED_CONTENT" ]; then
      echo_info "initdb.sql exists and matches the expected content. No action needed."
    else
      echo_error "initdb.sql exists but does not match the expected content."
      echo_info "Please review the existing initdb.sql or delete it to allow automatic recreation."
      exit 1
    fi
  else
    #echo_info "initdb.sql does not exist. Creating it now..."
    cat <<'EOF' > "$INITDB_FILE"
-- Create Roles
DO $$
BEGIN
   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nhost_hasura') THEN
      CREATE ROLE nhost_hasura WITH LOGIN PASSWORD 'nhost_hasura_password';
   END IF;

   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nhost_auth_admin') THEN
      CREATE ROLE nhost_auth_admin WITH LOGIN PASSWORD 'nhost_auth_admin_password';
   END IF;

   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nhost_storage_admin') THEN
      CREATE ROLE nhost_storage_admin WITH LOGIN PASSWORD 'nhost_storage_admin_password';
   END IF;
END
$$;

-- Create Schema
DO $$
BEGIN
   IF NOT EXISTS (SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'hdb_catalog') THEN
      CREATE SCHEMA hdb_catalog;
   END IF;
END
$$;

-- Enable TimescaleDB
DO $$
BEGIN
   IF NOT EXISTS (SELECT * FROM pg_available_extensions WHERE name = 'timescaledb') THEN
      CREATE EXTENSION timescaledb CASCADE;
   END IF;
END
$$;

-- Grant Permissions
GRANT postgres TO nhost_hasura;

GRANT USAGE ON SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT CREATE ON SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT ALL ON ALL TABLES IN SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT ALL ON ALL SEQUENCES IN SCHEMA hdb_catalog TO nhost_auth_admin;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA hdb_catalog TO nhost_auth_admin;

GRANT USAGE ON SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT CREATE ON SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT ALL ON ALL TABLES IN SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT ALL ON ALL SEQUENCES IN SCHEMA hdb_catalog TO nhost_storage_admin;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA hdb_catalog TO nhost_storage_admin;

-- Default Privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON TABLES TO nhost_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON SEQUENCES TO nhost_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON FUNCTIONS TO nhost_auth_admin;

ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON TABLES TO nhost_storage_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON SEQUENCES TO nhost_storage_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA hdb_catalog GRANT ALL ON FUNCTIONS TO nhost_storage_admin;
EOF
    echo_info "initdb.sql created successfully in /seeds"
  fi
}

# ----------------------------
# Main Execution
# ----------------------------

create_initdb_sql
