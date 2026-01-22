# Next Release: v0.4.4 - Database Tools

**Target**: Q1-Q2 2026
**Status**: Planning Complete
**Focus**: Complete database lifecycle management from local → staging → production

---

## Executive Summary

v0.4.4 delivers **enterprise-grade database tooling** that matches or exceeds Supabase and Nhost. This is the most feature-rich release in nself history, adding 8 new commands and 150+ subcommands.

---

## New Commands (8)

| Command | Purpose | Subcommands |
|---------|---------|-------------|
| `nself db` | Database operations & migrations | 30+ |
| `nself backup` | Backup management | 15+ |
| `nself restore` | Restore operations | 10+ |
| `nself seed` | Data seeding | 10+ |
| `nself mock` | Mock data generation | 12+ |
| `nself data` | Data utilities | 15+ |
| `nself schema` | Schema management (NEW) | 20+ |
| `nself types` | Type generation (NEW) | 12+ |

**Total new commands**: 8
**Total new subcommands**: 120+

---

## Highlight Features

### 1. Deterministic Shareable Mock Data

**Problem**: Developers have different test data, causing "works on my machine" issues.

**Solution**: Git-tracked mock configuration with deterministic seed.

```bash
# Developer A configures mock data
nself mock init
nself mock generate --seed 42
git add nself/mock/
git commit -m "Add shared mock data"

# Developer B gets IDENTICAL data
git pull
nself mock generate  # Same seed = same data!
```

### 2. Environment-Aware Safety Guards

| Command | Local | Staging | Production |
|---------|-------|---------|------------|
| `nself mock generate` | Allowed | Allowed | **BLOCKED** |
| `nself data reset` | Allowed | Allowed | **BLOCKED** |
| `nself db migrate:fresh` | Allowed | Allowed | **BLOCKED** |
| `nself data sync --from prod` | N/A | **Requires --anonymize** | N/A |

### 3. Cross-Environment Data Sync

```bash
# Copy prod data to staging with automatic PII anonymization
nself data sync --from prod --to staging --anonymize

# Clone database with anonymization
nself db clone --from prod --to staging --anonymize
```

### 4. Multi-Language Type Generation

```bash
nself types typescript -o types/db.ts    # TypeScript
nself types go -o models/db.go           # Go
nself types python --pydantic            # Python with Pydantic
nself types swift -o Models/DB.swift     # Swift
nself types rust -o src/models/db.rs     # Rust
nself types graphql -o schema.graphql    # GraphQL
```

### 5. Database Branching

```bash
nself db branch create feature-auth     # Create branch database
nself db branch switch feature-auth     # Switch to it
nself db branch --auto                  # Auto-create per git branch
nself db branch merge feature-auth      # Merge migrations back
```

### 6. Schema Diffing & Visualization

```bash
nself schema diff staging               # Diff local vs staging
nself schema diff prod staging          # Diff prod vs staging
nself schema diff --create-migration    # Auto-generate migration

nself schema diagram                    # Generate ER diagram
nself schema docs --format html         # Generate documentation
```

### 7. Database Inspection Tools

```bash
nself db inspect cache-hit              # Check cache efficiency
nself db inspect unused-indexes         # Find unused indexes
nself db inspect bloat                  # Check table/index bloat
nself db inspect slow-queries           # Show slow query log
nself db inspect locks                  # View current locks

nself schema indexes --missing          # Index advisor
```

### 8. Multi-Cloud Backup Storage

```bash
nself backup storage set s3             # AWS S3
nself backup storage set minio          # Self-hosted MinIO
nself backup storage set b2             # Backblaze B2
nself backup storage set gcs            # Google Cloud Storage
nself backup storage set azure          # Azure Blob Storage
```

---

## Command Reference

### `nself db` - Database Operations

```bash
# Migrations
nself db migrate                    # Run pending migrations
nself db migrate:create <name>      # Create new migration
nself db migrate:rollback           # Rollback last migration
nself db migrate:status             # Show migration status
nself db migrate:fresh              # Drop all, re-run (blocked in prod)

# Shell & Queries
nself db shell                      # Interactive psql
nself db shell --env staging        # Connect to staging
nself db query "<sql>"              # Run SQL
nself db query -f script.sql        # Run SQL file

# Dump & Load
nself db dump                       # Full dump
nself db dump --schema-only         # Schema only
nself db load <file>                # Load dump

# Cloning
nself db clone staging              # Clone to staging
nself db clone --from prod --anonymize  # Clone with anonymization

# Inspection
nself db inspect cache-hit          # Cache efficiency
nself db inspect bloat              # Table/index bloat
nself db inspect slow-queries       # Slow queries
nself db inspect locks              # Current locks

# Testing & Linting
nself db test                       # Run pgTAP tests
nself db lint                       # Lint schema
nself db lint --fail-on warning     # CI mode

# Branching
nself db branch create <name>       # Create branch DB
nself db branch list                # List branches
nself db branch switch <name>       # Switch branch
nself db branch --auto              # Auto per git branch
```

### `nself schema` - Schema Management

```bash
# Diffing
nself schema diff                   # Diff local vs migrations
nself schema diff staging           # Diff local vs staging
nself schema diff --create-migration # Auto-create migration

# Visualization
nself schema show                   # Show current schema
nself schema show --format dbml     # DBML format
nself schema diagram                # ER diagram (browser)
nself schema docs --format html     # HTML documentation

# Index Management
nself schema indexes                # List all indexes
nself schema indexes --unused       # Show unused
nself schema indexes --missing      # Suggest missing (advisor)
```

### `nself types` - Type Generation

```bash
nself types typescript              # TypeScript types
nself types typescript --with-relations
nself types go --package models     # Go structs
nself types python --pydantic       # Python Pydantic
nself types swift                   # Swift structs
nself types rust                    # Rust structs
nself types graphql --with-hasura   # GraphQL schema
nself types typescript --watch      # Watch mode
```

### `nself backup` - Backup System

```bash
# Creation
nself backup create                 # Full backup
nself backup create --incremental   # Incremental
nself backup create --encrypt       # Encrypted

# Management
nself backup list                   # List local
nself backup list --remote          # List S3
nself backup verify <id>            # Verify integrity

# Scheduling
nself backup schedule --preset daily
nself backup schedule --cron "0 */6 * * *"

# Storage
nself backup storage set s3 --bucket my-backups
nself backup upload <id>            # Upload to remote
nself backup sync                   # Sync local ↔ remote

# Retention
nself backup retention --keep-daily 7 --keep-weekly 4
nself backup prune                  # Apply retention
```

### `nself restore` - Restore System

```bash
nself restore latest                # Restore most recent
nself restore <backup-id>           # Restore specific

# Point-in-Time Recovery
nself restore --point-in-time "2026-01-15 14:30:00"
nself restore --point-in-time "1 hour ago"

# Cross-Environment
nself restore <id> --to staging     # Restore to staging
nself restore <id> --to staging --anonymize  # With anonymization
```

### `nself mock` - Mock Data Generation

```bash
# Generation
nself mock generate                 # All tables
nself mock generate users           # Specific table
nself mock generate --seed 42       # Deterministic (shareable!)

# Presets
nself mock generate --preset minimal    # 10 rows/table
nself mock generate --preset medium     # 1,000 rows/table
nself mock generate --preset stress     # 100,000+ for load testing

# Configuration
nself mock init                     # Create config file
nself mock config                   # Edit config
nself mock preview users            # Preview data

# Team Sharing
nself mock export -o mock.tar.gz    # Export config + seed
nself mock import mock.tar.gz       # Import shared config
```

### `nself seed` - Seeding System

```bash
nself seed                          # Run all seeds
nself seed run users                # Run specific seed
nself seed run --env staging        # Env-specific seeds
nself seed run --fresh              # Clear and re-seed

# Management
nself seed init                     # Create seed directory
nself seed create <name>            # Create new seed
nself seed list                     # List seeds
nself seed status                   # Show run status

# Extraction
nself seed extract users            # Create seed from table
nself seed extract users --where "role = 'admin'"
```

### `nself data` - Data Utilities

```bash
# Export
nself data export                   # Export as JSON
nself data export --format csv      # As CSV
nself data export --format parquet  # As Parquet

# Import
nself data import file.json         # Import JSON
nself data import file.csv --table users
nself data import dir/ --upsert     # Upsert mode

# Anonymization
nself data anonymize                # Anonymize PII
nself data anonymize --preview      # Preview changes
nself data anonymize --config anonymize.yaml

# Sync
nself data sync --from prod --to staging --anonymize
nself data sync --from local --to staging

# Reset
nself data reset                    # Clear all data
nself data reset --keep-schema      # Keep schema
nself data reset --keep-seeds       # Re-run seeds after
```

---

## File Structure

```
project/
├── nself/
│   ├── migrations/
│   │   ├── 001_20260115_create_users.sql
│   │   └── 002_20260116_add_posts.sql
│   ├── seeds/
│   │   ├── 001_admin_users.sql
│   │   ├── local/
│   │   │   └── 001_test_data.sql
│   │   └── staging/
│   │       └── 001_demo_data.sql
│   ├── mock/
│   │   ├── mock.config.yaml      # Git-tracked!
│   │   └── mock.seed             # Deterministic seed
│   ├── tests/
│   │   └── 001_user_permissions.sql
│   └── anonymize.yaml
├── types/
│   ├── db.ts                     # Generated TypeScript
│   └── db.go                     # Generated Go
└── nself.toml
```

---

## Mock Configuration Example

```yaml
# nself/mock/mock.config.yaml
seed: 42  # Same seed = same data for ALL developers

tables:
  users:
    count: 100
    columns:
      email: { type: email }
      name: { type: fullName }
      avatar: { type: avatar }
      created_at: { type: pastDate, years: 2 }

  posts:
    count: 500
    columns:
      title: { type: sentence, words: 5 }
      body: { type: paragraphs, count: 3 }
      user_id: { type: foreignKey, table: users }
      published: { type: boolean, probability: 0.8 }

  comments:
    count: 2000
    columns:
      content: { type: paragraph }
      user_id: { type: foreignKey, table: users }
      post_id: { type: foreignKey, table: posts }
```

---

## Anonymization Configuration

```yaml
# nself/anonymize.yaml
tables:
  users:
    email: fake          # Generate fake email
    name: fake           # Generate fake name
    phone: mask          # Mask: ###-###-####
    ssn: hash            # SHA256 hash
    address: fake        # Generate fake address

  payments:
    credit_card: null    # Remove entirely
    billing_address: fake

  # Keep these tables unchanged
  skip:
    - roles
    - permissions
    - app_config
```

---

## Competitor Feature Comparison

| Feature | nself | Supabase | Nhost |
|---------|-------|----------|-------|
| Migrations | ✓ | ✓ | ✓ |
| Seeding | ✓ | ✓ | ✓ |
| Schema Diffing | ✓ | ✓ | ✗ |
| Type Generation | ✓ (6 langs) | ✓ (4 langs) | ✗ |
| Database Inspection | ✓ | ✓ | ✗ |
| Database Testing (pgTAP) | ✓ | ✓ | ✗ |
| Database Linting | ✓ | ✓ | ✗ |
| Database Branching | ✓ | ✓ | ✗ |
| Backup with PITR | ✓ | ✓ | ✓ |
| **Deterministic Mock Data** | ✓ | ✗ | ✗ |
| **Cross-Env Anonymization** | ✓ | ✗ | ✗ |
| **Environment Safety Guards** | ✓ | ✗ | ✗ |
| **Multi-Cloud Backup** | ✓ (5) | ✗ | ✗ |
| **ER Diagram Generation** | ✓ | ✗ | ✗ |

---

## Implementation Phases

### Phase 1: Core (Weeks 1-2)
- `nself db` - Migrations, shell, query, dump/load
- `nself backup` - Create, list, storage
- `nself restore` - Basic restore, PITR

### Phase 2: Data Management (Weeks 3-4)
- `nself seed` - Seeding with env-specific seeds
- `nself mock` - Deterministic mock generation
- `nself data` - Export/import/anonymize

### Phase 3: Advanced (Weeks 5-6)
- `nself schema` - Diffing, visualization, index advisor
- `nself types` - Multi-language type generation
- Database branching, inspection, testing

---

## Success Metrics

- [ ] All 8 commands implemented
- [ ] 100+ unit tests
- [ ] Cross-platform (macOS/Linux)
- [ ] Supabase CLI feature parity
- [ ] Nhost CLI feature parity
- [ ] Deterministic mock data works across team
- [ ] PITR restore < 1 minute
- [ ] Type generation: 6 languages

---

*Full planning document: [v0.4.4-PLAN.md](./v0.4.4-PLAN.md)*
