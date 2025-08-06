# Database Tools Documentation

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Commands Reference](#commands-reference)
- [Workflow Guide](#workflow-guide)
- [Schema Management](#schema-management)
- [Migration System](#migration-system)
- [Database Seeding](#database-seeding)
- [Backup and Recovery](#backup-and-recovery)
- [Environment Configuration](#environment-configuration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Advanced Scenarios](#advanced-scenarios)

## Overview

The nself database tools provide a complete database management system built on top of PostgreSQL and Hasura. The system uses DBML (Database Markup Language) for schema definition, automatically generates SQL migrations, handles environment-specific seeding, and maintains automatic backups of all changes.

### Key Features

- **Schema-First Development**: Define your database structure in DBML format
- **Automatic Migration Generation**: Convert DBML changes to SQL migrations
- **Safe Team Workflow**: Lead developers create migrations, team members apply updates
- **Version Control**: Track all schema changes with Git-friendly text files
- **Automatic Backups**: Every change is backed up to `bin/dbsyncs/` with timestamps
- **Environment-Specific Seeding**: Different seed data for dev, staging, and production
- **Migration Safety**: No auto-migrations, explicit update commands
- **Rollback Support**: Revert to any previous state
- **dbdiagram.io Integration**: Optional visual schema design

## Quick Start

### 1. Initialize a New Project

```bash
# Create a new nself project
nself init
nself build

# This creates a sample schema.dbml file
```

### 2. Edit Your Schema

Edit `schema.dbml` to define your database structure:

```dbml
Table users {
  id uuid [pk, default: `gen_random_uuid()`]
  email varchar(255) [unique, not null]
  created_at timestamptz [default: `now()`]
}
```

### 3. Generate Migrations

```bash
# Analyze schema and generate migrations
nself db run

# This creates timestamped migration files in hasura/migrations/
```

### 4. Apply Migrations

```bash
# Start services
nself up

# Apply migrations to database
nself db migrate:up
```

## Commands Reference

### Primary Commands

| Command | Description |
|---------|-------------|
| `nself db` | Show help for all database commands |
| `nself db run` | Analyze schema.dbml and generate migrations (Lead Developers) |
| `nself db update` | Safely apply pending migrations (All Developers) |
| `nself db sync` | Pull schema from dbdiagram.io (manual export) |
| `nself db migrate:up` | Apply pending migrations |
| `nself db migrate:down` | Rollback last migration |
| `nself db seed` | Seed database with sample data |
| `nself db reset` | Reset database (drop and recreate) |
| `nself db revert` | Revert to previous backup |
| `nself db status` | Show current database status |
| `nself db sample` | Create sample schema.dbml |

### Command Details

#### `nself db run`

Analyzes your `schema.dbml` file and generates SQL migrations for any changes detected.

```bash
# Basic usage
nself db run

# Force regenerate (ignore hash check)
nself db run --force
```

**What it does:**
1. Backs up current state to `bin/dbsyncs/YYYY-MM-DD_HH-MM-SS/`
2. Compares schema.dbml with previous hash
3. If changes detected, generates new migration in `hasura/migrations/default/`
4. Creates both up.sql and down.sql files
5. Shows instructions to apply migrations

#### `nself db sync`

Synchronize schema from dbdiagram.io project.

```bash
# Requires DBDIAGRAM_URL in .env.local
nself db sync
```

**Process:**
1. Opens dbdiagram.io URL (configured in environment)
2. Prompts for manual DBML export
3. Updates local schema.dbml
4. Automatically runs `nself db run` to generate migrations

#### `nself db migrate:create <name>`

Create a manual migration for custom database changes.

```bash
nself db migrate:create add_user_preferences
```

Creates empty migration files for manual SQL editing.

#### `nself db update`

Safely apply pending migrations with confirmation prompt (recommended for all developers).

```bash
nself db update
```

**Features:**
- Checks for pending migrations
- Shows list of migrations to apply
- Asks for confirmation
- Creates automatic backup
- Applies migrations and seeds

#### `nself db migrate:up`

Directly apply all pending migrations (for lead developers).

```bash
nself db migrate:up
```

**Requirements:**
- PostgreSQL container must be running (`nself up`)
- Migrations are applied in chronological order

#### `nself db migrate:down [steps]`

Rollback migrations.

```bash
# Rollback last migration
nself db migrate:down

# Rollback last 3 migrations
nself db migrate:down 3
```

**Important**: Creates backup before rollback in `bin/dbsyncs/`

#### `nself db seed`

Seed database with sample data.

```bash
nself db seed
```

**Seed Structure:**
```
seeds/
├── common/           # Applied in all environments
│   ├── 01_users.sql
│   └── 02_settings.sql
├── development/      # Dev-only data
│   └── 01_test_data.sql
├── staging/          # Staging data
│   └── 01_demo_data.sql
└── production/       # Production essentials
    └── 01_admin.sql
```

#### `nself db reset`

Complete database reset.

```bash
nself db reset
# Type 'yes' to confirm
```

**Process:**
1. Creates backup
2. Drops database
3. Recreates database
4. Applies all migrations
5. Runs seeds

#### `nself db revert`

Revert to previous database state.

```bash
nself db revert
```

Restores from the latest backup in `bin/dbsyncs/`

#### `nself db status`

Display current database status.

```bash
nself db status
```

**Shows:**
- Schema file status and hash
- Migration count and latest migrations
- Backup count and latest backup
- PostgreSQL container status

## Workflow Guide

### Development Workflow for Lead Developers

1. **Edit Schema Locally**
   ```bash
   # Edit your schema
   nano schema.dbml
   ```

2. **Generate Migrations**
   ```bash
   # Analyze and create migrations
   nself db run
   ```

3. **Review Generated Files**
   ```bash
   # Check the generated SQL
   cat hasura/migrations/default/*/up.sql
   ```

4. **Test Migrations**
   ```bash
   # Apply to your local database
   nself db migrate:up
   ```

5. **Commit to Git**
   ```bash
   git add schema.dbml hasura/migrations/
   git commit -m "Add user preferences table"
   git push
   ```

### Workflow for All Developers

1. **Pull Latest Code**
   ```bash
   git pull
   ```

2. **Start Services**
   ```bash
   nself up
   # If you see "DATABASE MIGRATIONS PENDING" warning...
   ```

3. **Apply Updates**
   ```bash
   # Safe command with confirmation
   nself db update
   ```

### Team Collaboration Workflow

1. **Lead Developer Updates Schema**
   ```bash
   # Option A: Edit locally
   nano schema.dbml
   
   # Option B: Design in dbdiagram.io then sync
   nself db sync
   ```

2. **Generate and Test Migrations**
   ```bash
   # Generate migrations
   nself db run
   
   # Test locally
   nself db migrate:up
   ```

3. **Commit and Push**
   ```bash
   git add schema.dbml hasura/migrations/
   git commit -m "Add user preferences table"
   git push
   ```

4. **Team Members Pull and Apply**
   ```bash
   git pull
   nself db migrate:up
   ```

### Production Deployment Workflow

1. **Test in Staging**
   ```bash
   # Set environment
   export ENVIRONMENT=staging
   
   # Apply migrations
   nself db migrate:up
   
   # Run staging seeds
   nself db seed
   ```

2. **Backup Production**
   ```bash
   # Manual backup before deployment
   nself db status  # Check current state
   cp -r hasura/migrations/ migrations.backup/
   ```

3. **Deploy to Production**
   ```bash
   # Set environment
   export ENVIRONMENT=production
   export DBML_AUTO_MIGRATE=false  # Never auto-migrate in production
   
   # Apply migrations carefully
   nself db migrate:up
   ```

## Schema Management

### DBML Syntax

DBML (Database Markup Language) provides a simple, readable syntax for defining database schemas.

#### Basic Table Definition

```dbml
Table users {
  id uuid [pk, default: `gen_random_uuid()`]
  email varchar(255) [unique, not null]
  username varchar(100) [unique]
  created_at timestamptz [default: `now()`]
  
  Indexes {
    email
    username
  }
}
```

#### Relationships

```dbml
Table posts {
  id uuid [pk]
  author_id uuid [ref: > users.id, not null]  // many-to-one
  title varchar(500) [not null]
}

Table profiles {
  id uuid [pk]
  user_id uuid [ref: - users.id, not null]  // one-to-one
}
```

#### Enums

```dbml
enum user_role {
  admin
  moderator
  user
  guest
}

Table users {
  id uuid [pk]
  role user_role [default: 'user']
}
```

#### Complex Example

```dbml
Project MyApp {
  database_type: 'PostgreSQL'
  Note: 'Multi-tenant SaaS application'
}

Table organizations {
  id uuid [pk, default: `gen_random_uuid()`]
  name varchar(255) [not null]
  slug varchar(100) [unique, not null]
  plan varchar(50) [default: 'free']
  created_at timestamptz [default: `now()`]
  
  Indexes {
    slug
    plan
    created_at
  }
}

Table organization_members {
  org_id uuid [ref: > organizations.id]
  user_id uuid [ref: > users.id]
  role varchar(50) [default: 'member']
  joined_at timestamptz [default: `now()`]
  
  Indexes {
    (org_id, user_id) [pk]
  }
}
```

### Schema Evolution

#### Adding Tables

```dbml
// Add this to schema.dbml
Table notifications {
  id uuid [pk, default: `gen_random_uuid()`]
  user_id uuid [ref: > users.id, not null]
  message text [not null]
  read boolean [default: false]
  created_at timestamptz [default: `now()`]
}
```

```bash
# Generate migration
nself db run
```

#### Adding Columns

```dbml
Table users {
  // ... existing columns ...
  phone varchar(20)  // New column
  verified_at timestamptz  // New column
}
```

#### Modifying Columns

For column modifications that can't be expressed in DBML:

```bash
# Create manual migration
nself db migrate:create alter_users_email_length

# Edit the generated files
nano hasura/migrations/default/*/up.sql
```

## Migration System

### Migration Structure

```
hasura/migrations/default/
├── 20240101120000_initial_schema/
│   ├── up.sql      # Forward migration
│   └── down.sql    # Rollback migration
├── 20240102153000_add_notifications/
│   ├── up.sql
│   └── down.sql
```

### Auto-Generated Migrations

When you run `nself db run`, migrations are generated with:

- **Idempotent operations**: Uses `IF NOT EXISTS` clauses
- **Automatic rollback**: Generates DROP statements in down.sql
- **Timestamps**: Each migration has a unique timestamp
- **Descriptive names**: Clear indication of changes

Example generated up.sql:
```sql
-- Auto-generated migration from schema.dbml
-- Generated: 2024-01-15 10:30:00

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
```

Example generated down.sql:
```sql
-- Rollback migration
-- WARNING: Review and modify as needed before using in production

DROP TABLE IF EXISTS users CASCADE;
```

### Manual Migrations

For complex changes that can't be expressed in DBML:

```bash
# Create migration
nself db migrate:create add_user_search_function

# Edit up.sql
cat > hasura/migrations/default/*/up.sql << 'EOF'
CREATE OR REPLACE FUNCTION search_users(search_term TEXT)
RETURNS SETOF users AS $$
BEGIN
  RETURN QUERY
  SELECT * FROM users
  WHERE email ILIKE '%' || search_term || '%'
     OR full_name ILIKE '%' || search_term || '%';
END;
$$ LANGUAGE plpgsql;
EOF

# Edit down.sql
cat > hasura/migrations/default/*/down.sql << 'EOF'
DROP FUNCTION IF EXISTS search_users(TEXT);
EOF
```

## Database Seeding

### Seed File Organization

```
seeds/
├── common/              # All environments
│   ├── 01_settings.sql  # System settings
│   └── 02_roles.sql     # Default roles
├── development/         # Dev environment
│   ├── 01_test_users.sql
│   └── 02_sample_data.sql
├── staging/            # Staging environment
│   └── 01_demo_data.sql
└── production/         # Production environment
    └── 01_admin_user.sql
```

### Writing Seed Files

#### Development Seeds

```sql
-- seeds/development/01_test_users.sql
INSERT INTO users (email, username, full_name, role) VALUES
  ('admin@example.com', 'admin', 'Admin User', 'admin'),
  ('john@example.com', 'johndoe', 'John Doe', 'user'),
  ('jane@example.com', 'janedoe', 'Jane Doe', 'user')
ON CONFLICT (email) DO NOTHING;

-- Add test organizations
INSERT INTO organizations (name, slug, owner_id) 
SELECT 'Test Org', 'test-org', id FROM users WHERE email = 'admin@example.com'
ON CONFLICT (slug) DO NOTHING;
```

#### Production Seeds

```sql
-- seeds/production/01_admin_user.sql
-- Minimal production data
INSERT INTO users (email, username, role) VALUES
  ('admin@company.com', 'admin', 'admin')
ON CONFLICT (email) DO NOTHING;

-- System settings
INSERT INTO settings (key, value) VALUES
  ('maintenance_mode', 'false'),
  ('allow_registration', 'true')
ON CONFLICT (key) DO NOTHING;
```

### Environment-Specific Seeding

```bash
# Development (includes all test data)
ENVIRONMENT=development nself db seed

# Staging (includes demo data)
ENVIRONMENT=staging nself db seed

# Production (minimal required data)
ENVIRONMENT=production nself db seed
```

## Backup and Recovery

### Automatic Backups

Every database operation creates a timestamped backup:

```
bin/dbsyncs/
├── 2024-01-15_10-30-00/
│   ├── schema.dbml
│   ├── migrations/
│   ├── seeds/
│   └── metadata.json
├── 2024-01-15_14-45-30/
│   ├── schema.dbml
│   ├── migrations/
│   ├── seeds/
│   └── metadata.json
```

### Manual Backup

```bash
# Check current state
nself db status

# Manual backup (happens automatically on most operations)
cp -r hasura/migrations/ backup/
cp schema.dbml backup/
```

### Recovery Options

#### Revert to Previous State

```bash
# Revert to last backup
nself db revert
```

#### Restore Specific Backup

```bash
# List backups
ls -la bin/dbsyncs/

# Restore specific backup
cp -r bin/dbsyncs/2024-01-15_10-30-00/migrations hasura/
cp bin/dbsyncs/2024-01-15_10-30-00/schema.dbml .

# Apply restored state
nself db migrate:up
```

#### Disaster Recovery

```bash
# Complete reset and rebuild
nself db reset

# Or manually:
docker exec myproject-postgres-1 psql -U postgres -c "DROP DATABASE nhost;"
docker exec myproject-postgres-1 psql -U postgres -c "CREATE DATABASE nhost;"
nself db migrate:up
nself db seed
```

## Environment Configuration

### Configuration Variables

Add to `.env.local`:

```bash
# Database Configuration
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=nhost
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secretpassword

# Schema Management
LOCAL_SCHEMA_FILE=schema.dbml      # Path to schema file
DBDIAGRAM_URL=https://dbdiagram.io/d/your-project  # Optional

# Environment
ENVIRONMENT=development            # development, staging, production
PROJECT_NAME=myproject             # Used for container names

# IMPORTANT: Auto-migration has been removed for safety
# All migrations must be explicitly applied with 'nself db update'
```

### Environment-Specific Settings

#### Development

```bash
# .env.local (development)
ENVIRONMENT=development
LOCAL_SCHEMA_FILE=schema.dbml
```

#### Staging

```bash
# .env.staging
ENVIRONMENT=staging
LOCAL_SCHEMA_FILE=schema.dbml
```

#### Production

```bash
# .env (production)
ENVIRONMENT=production
LOCAL_SCHEMA_FILE=schema.dbml
# Always use 'nself db update' for migrations
```

## Best Practices

### Schema Design

1. **Use UUIDs for Primary Keys**
   ```dbml
   id uuid [pk, default: `gen_random_uuid()`]
   ```

2. **Always Add Timestamps**
   ```dbml
   created_at timestamptz [default: `now()`]
   updated_at timestamptz [default: `now()`]
   ```

3. **Create Appropriate Indexes**
   ```dbml
   Indexes {
     email              // Single column
     (org_id, user_id)  // Composite
     created_at         // For sorting
   }
   ```

4. **Use Enums for Fixed Values**
   ```dbml
   enum status {
     active
     inactive
     suspended
   }
   ```

5. **Document Relationships**
   ```dbml
   author_id uuid [ref: > users.id, not null]  // Clear foreign keys
   ```

### Migration Management

1. **Review Generated Migrations**
   - Always check up.sql and down.sql before applying
   - Test rollbacks in development

2. **Keep Migrations Small**
   - One logical change per migration
   - Easier to debug and rollback

3. **Use Idempotent Operations**
   ```sql
   CREATE TABLE IF NOT EXISTS ...
   ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...
   ```

4. **Test Migration Rollbacks**
   ```bash
   nself db migrate:up
   nself db migrate:down
   nself db migrate:up
   ```

5. **Never Edit Applied Migrations**
   - Create new migrations for fixes
   - Keep migration history intact

### Seeding Best Practices

1. **Use ON CONFLICT Clauses**
   ```sql
   INSERT INTO users (email, username)
   VALUES ('admin@example.com', 'admin')
   ON CONFLICT (email) DO NOTHING;
   ```

2. **Environment-Specific Data**
   - Development: Comprehensive test data
   - Staging: Realistic demo data
   - Production: Minimal required data

3. **Reference Data by Query**
   ```sql
   INSERT INTO posts (author_id, title)
   SELECT id, 'Welcome Post' FROM users WHERE email = 'admin@example.com';
   ```

### Version Control

1. **Commit Schema and Migrations Together**
   ```bash
   git add schema.dbml hasura/migrations/
   git commit -m "Add notifications table"
   ```

2. **Don't Commit Backups**
   ```gitignore
   bin/dbsyncs/
   *.backup
   ```

3. **Document Breaking Changes**
   ```bash
   git commit -m "BREAKING: Rename users.name to users.full_name"
   ```

## Troubleshooting

### Common Issues

#### Schema Not Syncing

```bash
# Check current status
nself db status

# Force regenerate
nself db run --force

# Check hash file
cat .nself/schema.hash
```

#### Migration Failures

```bash
# Check PostgreSQL logs
docker logs myproject-postgres-1

# Connect directly to database
docker exec -it myproject-postgres-1 psql -U postgres -d nhost

# Apply migrations manually
docker exec -i myproject-postgres-1 psql -U postgres -d nhost < hasura/migrations/default/*/up.sql
```

#### Container Not Running

```bash
# Check container status
docker ps -a | grep postgres

# Start services
nself up

# Check logs
docker logs myproject-postgres-1
```

#### Invalid DBML Syntax

```bash
# Install DBML CLI
npm install -g @dbml/cli

# Validate syntax
dbml2sql schema.dbml --postgres

# Check for errors
dbml-cli validate schema.dbml
```

#### Rollback Issues

```bash
# List available backups
ls -la bin/dbsyncs/

# Manual restore
cp bin/dbsyncs/latest/schema.dbml .
cp -r bin/dbsyncs/latest/migrations hasura/

# Force reset
nself db reset
```

### Debug Commands

```bash
# Enable debug output
set -x
nself db run

# Check generated SQL
cat /tmp/dbml_*.sql

# Test database connection
docker exec myproject-postgres-1 psql -U postgres -c "SELECT 1"

# List all tables
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "\dt"

# Show table structure
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "\d users"
```

## Advanced Scenarios

### Multi-Tenant Architecture

```dbml
Table tenants {
  id uuid [pk, default: `gen_random_uuid()`]
  name varchar(255) [not null]
  subdomain varchar(100) [unique, not null]
  plan varchar(50) [default: 'free']
  created_at timestamptz [default: `now()`]
}

Table users {
  id uuid [pk, default: `gen_random_uuid()`]
  tenant_id uuid [ref: > tenants.id, not null]
  email varchar(255) [not null]
  
  Indexes {
    tenant_id
    (tenant_id, email) [unique]
  }
}

// All tables include tenant_id for isolation
Table posts {
  id uuid [pk]
  tenant_id uuid [ref: > tenants.id, not null]
  author_id uuid [ref: > users.id]
  // ...
  
  Indexes {
    tenant_id
    (tenant_id, author_id)
  }
}
```

### Soft Deletes

```dbml
Table users {
  id uuid [pk]
  email varchar(255) [unique]
  deleted_at timestamptz  // Soft delete timestamp
  
  Indexes {
    email
    deleted_at  // For filtering active records
  }
}
```

Generate view for active records:

```bash
nself db migrate:create add_active_users_view

# In up.sql:
CREATE VIEW active_users AS
SELECT * FROM users WHERE deleted_at IS NULL;

# In down.sql:
DROP VIEW IF EXISTS active_users;
```

### Audit Logging

```dbml
Table audit_logs {
  id uuid [pk, default: `gen_random_uuid()`]
  user_id uuid [ref: > users.id]
  action varchar(100) [not null]
  table_name varchar(100) [not null]
  record_id uuid
  old_values jsonb
  new_values jsonb
  ip_address inet
  created_at timestamptz [default: `now()`]
  
  Indexes {
    user_id
    table_name
    record_id
    created_at
  }
}
```

### JSON/JSONB Columns

```dbml
Table users {
  id uuid [pk]
  email varchar(255) [unique, not null]
  metadata jsonb [default: '{}']  // Flexible metadata
  preferences jsonb [default: '{"theme": "light", "notifications": true}']
}

Table events {
  id uuid [pk]
  type varchar(100) [not null]
  payload jsonb [not null]  // Event data
  processed_at timestamptz
  
  Indexes {
    type
    processed_at
    (payload->>'user_id')  // Index on JSON field
  }
}
```

### Full-Text Search

```dbml
Table articles {
  id uuid [pk]
  title varchar(500) [not null]
  content text [not null]
  search_vector tsvector  // Full-text search vector
  
  Indexes {
    search_vector  // GIN index for FTS
  }
}
```

Manual migration for search trigger:

```sql
-- up.sql
CREATE OR REPLACE FUNCTION update_search_vector() RETURNS trigger AS $$
BEGIN
  NEW.search_vector := to_tsvector('english', NEW.title || ' ' || NEW.content);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_search_vector_trigger
BEFORE INSERT OR UPDATE ON articles
FOR EACH ROW EXECUTE FUNCTION update_search_vector();

-- down.sql
DROP TRIGGER IF EXISTS update_search_vector_trigger ON articles;
DROP FUNCTION IF EXISTS update_search_vector();
```

### Time-Series Data

```dbml
Table metrics {
  time timestamptz [not null]
  sensor_id uuid [not null]
  value numeric [not null]
  metadata jsonb
  
  Indexes {
    (sensor_id, time DESC)  // Composite index for queries
  }
}
```

With TimescaleDB:

```bash
# Enable in .env.local
POSTGRES_EXTENSIONS=timescaledb

# Create hypertable
nself db migrate:create create_metrics_hypertable

# up.sql
SELECT create_hypertable('metrics', 'time');
```

### Database Functions and Triggers

```bash
# Create migration for updated_at trigger
nself db migrate:create add_updated_at_trigger

# up.sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();

# down.sql
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at();
```

### Performance Optimization

```dbml
Table large_table {
  id uuid [pk]
  category varchar(50) [not null]
  status varchar(50) [not null]
  created_at timestamptz [not null]
  data jsonb
  
  Indexes {
    category                           // Single column
    status                            // Single column
    (category, status)                // Composite
    created_at                        // Range queries
    (status, created_at DESC)         // Filtered sorting
    (data->>'user_id')               // JSON field
  }
}
```

### Testing Migrations

```bash
# Test migration in isolated environment
docker run --rm -d --name test-db -e POSTGRES_PASSWORD=test postgres:14
docker exec test-db createdb -U postgres test_db

# Apply migrations
docker exec -i test-db psql -U postgres -d test_db < hasura/migrations/default/*/up.sql

# Run tests
# ...

# Cleanup
docker stop test-db
```

## Integration with Hasura

### Hasura Metadata

After applying migrations, update Hasura metadata:

```bash
# Track all tables
hasura metadata apply

# Or manually via console
# Open Hasura console and track tables/relationships
```

### Permissions

Define permissions in Hasura console or metadata:

```yaml
# hasura/metadata/databases/default/tables/public_users.yaml
- table:
    name: users
    schema: public
  permissions:
    - role: user
      permission:
        check:
          id:
            _eq: X-Hasura-User-Id
        columns:
          - id
          - email
          - username
          - created_at
```

### Custom Actions

```graphql
# hasura/metadata/actions.graphql
type Mutation {
  registerUser(email: String!, password: String!): UserResponse
}
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/database.yml
name: Database CI/CD

on:
  push:
    paths:
      - 'schema.dbml'
      - 'hasura/migrations/**'
      - 'seeds/**'

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install DBML CLI
        run: npm install -g @dbml/cli
      
      - name: Validate Schema
        run: dbml2sql schema.dbml --postgres -o /tmp/test.sql
      
      - name: Test Migrations
        run: |
          for migration in hasura/migrations/default/*/up.sql; do
            psql -h localhost -U postgres -d postgres < "$migration"
          done
        env:
          PGPASSWORD: test
      
      - name: Test Rollbacks
        run: |
          for migration in hasura/migrations/default/*/down.sql; do
            psql -h localhost -U postgres -d postgres < "$migration"
          done
        env:
          PGPASSWORD: test

  deploy:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Deploy to Production
        run: |
          # Your deployment script
          echo "Deploying database changes..."
```

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - test
  - deploy

test-migrations:
  stage: test
  image: postgres:14
  services:
    - postgres:14
  variables:
    POSTGRES_DB: test
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: test
  script:
    - apt-get update && apt-get install -y postgresql-client
    - for migration in hasura/migrations/default/*/up.sql; do
        PGPASSWORD=$POSTGRES_PASSWORD psql -h postgres -U postgres -d test < "$migration";
      done

deploy-migrations:
  stage: deploy
  only:
    - main
  script:
    - echo "Deploying to production..."
```

## Performance Considerations

### Index Strategy

```dbml
Table orders {
  id uuid [pk]
  user_id uuid [ref: > users.id, not null]
  status varchar(50) [not null]
  total decimal(10,2) [not null]
  created_at timestamptz [not null]
  
  Indexes {
    user_id                        // Foreign key lookups
    status                         // Filtering
    created_at                     // Sorting/range queries
    (user_id, created_at DESC)    // User's recent orders
    (status, created_at)           // Status filtering with sort
  }
}
```

### Partitioning

For large tables, consider partitioning:

```bash
nself db migrate:create partition_events_table

# up.sql
-- Create partitioned table
CREATE TABLE events (
  id uuid DEFAULT gen_random_uuid(),
  created_at timestamptz NOT NULL,
  data jsonb
) PARTITION BY RANGE (created_at);

-- Create partitions
CREATE TABLE events_2024_01 PARTITION OF events
  FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE events_2024_02 PARTITION OF events
  FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

### Query Optimization

```sql
-- Add covering index
CREATE INDEX idx_users_email_include 
ON users(email) 
INCLUDE (id, username, created_at);

-- Partial index for common queries
CREATE INDEX idx_active_users 
ON users(created_at) 
WHERE deleted_at IS NULL;

-- Expression index
CREATE INDEX idx_users_lower_email 
ON users(LOWER(email));
```

## Security Considerations

### Row-Level Security

```bash
nself db migrate:create add_row_level_security

# up.sql
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;

CREATE POLICY posts_owner_policy ON posts
  FOR ALL
  USING (author_id = current_setting('app.user_id')::uuid);

CREATE POLICY posts_public_read ON posts
  FOR SELECT
  USING (status = 'published');
```

### Data Encryption

```dbml
Table sensitive_data {
  id uuid [pk]
  user_id uuid [ref: > users.id, not null]
  encrypted_ssn bytea              // Encrypted at application level
  encrypted_credit_card bytea      // Encrypted at application level
  created_at timestamptz [default: `now()`]
}
```

### Audit Requirements

```sql
-- Comprehensive audit trigger
CREATE OR REPLACE FUNCTION audit_trigger_function()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO audit_logs (
    user_id,
    action,
    table_name,
    record_id,
    old_values,
    new_values,
    created_at
  ) VALUES (
    current_setting('app.user_id', true)::uuid,
    TG_OP,
    TG_TABLE_NAME,
    COALESCE(NEW.id, OLD.id),
    to_jsonb(OLD),
    to_jsonb(NEW),
    NOW()
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_users
AFTER INSERT OR UPDATE OR DELETE ON users
FOR EACH ROW EXECUTE FUNCTION audit_trigger_function();
```

## Monitoring and Maintenance

### Database Health Checks

```bash
# Check table sizes
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "
  SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
  FROM pg_tables
  WHERE schemaname = 'public'
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# Check slow queries
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "
  SELECT 
    query,
    calls,
    total_time,
    mean_time
  FROM pg_stat_statements
  ORDER BY mean_time DESC
  LIMIT 10;"
```

### Regular Maintenance

```bash
# Vacuum and analyze
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "VACUUM ANALYZE;"

# Reindex
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "REINDEX DATABASE nhost;"

# Check for unused indexes
docker exec myproject-postgres-1 psql -U postgres -d nhost -c "
  SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan
  FROM pg_stat_user_indexes
  WHERE idx_scan = 0
  ORDER BY schemaname, tablename;"
```

### Backup Strategy

```bash
# Automated daily backups
cat > backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backups/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# Backup database
docker exec myproject-postgres-1 pg_dump -U postgres nhost | gzip > "$BACKUP_DIR/nhost.sql.gz"

# Backup schema and migrations
cp -r hasura/migrations "$BACKUP_DIR/"
cp schema.dbml "$BACKUP_DIR/"

# Keep only last 30 days
find /backups -type d -mtime +30 -exec rm -rf {} \;
EOF

# Add to crontab
# 0 2 * * * /path/to/backup.sh
```

## Summary

The nself database tools provide a complete, production-ready database management system that:

1. **Simplifies Development**: Write schemas in DBML, auto-generate migrations
2. **Ensures Safety**: Automatic backups, easy rollbacks
3. **Supports Teams**: Git-friendly, clear workflows
4. **Handles Complexity**: From simple apps to multi-tenant SaaS
5. **Integrates Seamlessly**: Works with Hasura, Docker, CI/CD

Whether you're building a simple application or a complex multi-tenant system, these tools provide the foundation for reliable database management throughout your development lifecycle.