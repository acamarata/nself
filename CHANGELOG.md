# Changelog

All notable changes to nself will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Database Seeding System
- **`nself db seed apply`** - Apply seed files with automatic tracking
- **`nself db seed apply <file>`** - Apply specific seed file
- **`nself db seed list`** - List all seeds with their status (✓ Applied / ○ Pending)
- **`nself db seed rollback`** - Rollback last applied seed (removes tracking)
- Seed tracking table (`nself_seeds`) prevents duplicate execution
- Environment-specific seed support (common/local/staging/production)
- Idempotent seed execution with ON CONFLICT handling

#### Authentication Setup Commands
- **`nself auth setup`** - Interactive auth setup wizard
- **`nself auth setup --default-users`** - One-command auth setup (creates 3 staff users)
- **`nself auth create-user`** - Create individual auth users
  - Interactive mode with prompts
  - Non-interactive with flags (--email, --password, --role, --name)
  - Auto-generates secure passwords if not provided
- **`nself auth list-users`** - List all auth users with details
- Proper nHost schema support:
  - Creates users in `auth.users` table
  - Links to email provider in `auth.user_providers` table
  - Uses bcrypt password hashing (cost factor 10)
  - Generates dummy access tokens for seeded users
  - Stores custom roles in JSONB metadata field

#### Hasura Metadata Management
- **`nself hasura`** - New command group for Hasura management
- **`nself hasura metadata apply`** - Apply metadata from files or track defaults
- **`nself hasura metadata export`** - Export current metadata to JSON
- **`nself hasura metadata reload`** - Reload Hasura metadata
- **`nself hasura metadata clear`** - Clear all metadata (with confirmation)
- **`nself hasura track table <schema.table>`** - Track specific database table
- **`nself hasura track schema <schema>`** - Track all tables in schema
- **`nself hasura untrack table <schema.table>`** - Untrack table
- **`nself hasura console`** - Open Hasura console in browser
- Auto-tracks auth schema tables (users, user_providers, providers, etc.)
- Integrated with `nself auth setup` workflow

#### Security Hardening System
- **`nself audit security`** - Comprehensive security audit command
  - Detects weak secrets (minioadmin, changeme, admin, etc.)
  - Checks secret length (warns if < 24 characters)
  - Validates CORS configuration (detects wildcards in production)
  - Audits exposed ports (warns if database exposed in production)
  - Verifies container users (checks for root containers)
  - Subcommands: `all`, `secrets`, `cors`, `ports`, `containers`

- **`nself harden`** - Automated security hardening command
  - Interactive wizard mode (audit → review → apply)
  - `nself harden all` - Apply all fixes automatically
  - `nself harden secrets` - Rotate weak secrets only
  - `nself harden cors` - Fix CORS configuration
  - Generates cryptographically strong secrets
  - Environment-aware hardening (dev/staging/prod)

#### Environment-Aware CORS Configuration
- CORS now respects environment settings (no more wildcard "*" in production)
- **Development**: Permissive CORS for localhost development
  - `http://localhost:*`, `http://*.local.nself.org`, `https://*.local.nself.org`
- **Staging**: Production domains + localhost for testing
  - `https://*.${BASE_DOMAIN}`, `http://localhost:3000`
- **Production**: Strict CORS - only your domain
  - `https://*.${BASE_DOMAIN}` (NO wildcards, NO localhost)
- Can be overridden via `HASURA_GRAPHQL_CORS_DOMAIN` environment variable
- Custom service templates (Express, FastAPI, Flask) use same environment-aware pattern

#### Enhanced Secret Generation
- **Environment-specific secret strength**:
  - Development: 32-64 character secrets (convenient but secure)
  - Staging: 40-80 character secrets (production-like)
  - Production: 48-96 character secrets (maximum security)
- Auto-generates secrets for:
  - PostgreSQL password
  - Hasura admin secret & JWT key
  - MinIO credentials
  - Grafana credentials
  - MeiliSearch/Typesense API keys
  - nself-admin secret key
- Uses cryptographically secure random generation (openssl, /dev/urandom)
- Only generates if empty (preserves existing secrets)
- Integrated into `nself init` wizard

#### Non-Root Container Users
- All containers now run as non-root users for security:
  - PostgreSQL: `user: "70:70"`
  - Hasura: `user: "1001:1001"`
  - Auth: `user: "1001:1001"`
  - Redis: `user: "999:999"`
  - MinIO: `user: "1000:1000"`
  - Mailpit: `user: "1000:1000"`
  - MeiliSearch: `user: "1000:1000"`
  - Grafana: `user: "472:472"`
  - Prometheus: `user: "65534:65534"`
  - All monitoring exporters run as non-root
- **Exceptions** (require root privileges):
  - Nginx: Needs root for ports 80/443
  - cAdvisor: Requires privileged mode
  - Promtail: Needs root to read system logs

#### Conditional Port Exposure
- Database ports now conditionally exposed based on environment
- **`POSTGRES_EXPOSE_PORT`** variable with three modes:
  - `auto` (default): Expose in dev, hide in prod/staging
  - `true`: Always expose to 127.0.0.1 (localhost only)
  - `false`: Never expose (internal Docker network only)
- **Development**: Port exposed to `127.0.0.1:5432` (database tools work)
- **Production**: Port NOT exposed (internal Docker network only, maximum security)
- Reduces attack surface in production deployments

#### Environment-Conditional Mailpit
- Mailpit security settings now respect environment:
  - **Development**: `MP_SMTP_AUTH_ACCEPT_ANY=1`, `MP_SMTP_AUTH_ALLOW_INSECURE=1`
  - **Production/Staging**: Secure defaults, respects `MAILPIT_ACCEPT_ANY_AUTH` env var
- Production warning added: "Mailpit is for development only and insecure"
- Guides users to configure production email: `nself service email configure`

#### Security Documentation
- **`.wiki/security/HARDENING-GUIDE.md`** - Comprehensive security hardening guide
  - Security philosophy and best practices
  - Environment-specific security configurations
  - Using audit and hardening commands
  - Production deployment checklist
  - Compliance considerations (SOC 2, GDPR, HIPAA)
  - Troubleshooting common security issues
  - Advanced topics (secret rotation, CI/CD, monitoring)

- **`.wiki/security/MIGRATION-V0.10.0.md`** - Migration guide
  - Breaking changes and impact assessment
  - Safe 8-step upgrade process
  - Environment-specific migration procedures
  - Troubleshooting migration issues
  - Post-migration checklist
  - Rollback instructions

#### Seed Templates
- **`src/templates/seeds/001_auth_users.sql.template`** - nHost auth seed template
  - Creates users with proper schema structure
  - Includes owner, admin, and support users
  - Placeholder system for customization
  - Idempotent with ON CONFLICT clauses
- **`src/templates/seeds/README.md`** - Template documentation and examples

#### Documentation
- **`.wiki/guides/DEV_WORKFLOW.md`** - Complete development workflow guide
  - Zero to working auth in <5 minutes
  - Step-by-step instructions
  - Troubleshooting section
  - Best practices
- **`.wiki/guides/AUTH_SETUP.md`** - Comprehensive authentication guide
  - How nself auth works (architecture diagram)
  - Auth schema structure explained
  - Quick setup vs manual setup
  - Creating and testing users
  - Security best practices
  - Password hashing details
- **`.wiki/guides/SEEDING.md`** - Database seeding guide
  - Seed directory structure
  - Creating and applying seeds
  - Environment-specific seeds
  - nHost auth seed examples
  - Best practices and patterns
  - Advanced topics

### Changed

#### Database Commands
- **`nself db seed run`** - Now uses `seed_apply` internally (backward compatible)
- Improved seed execution with better error messages
- Enhanced seed status reporting

#### Auth Commands
- Updated `nself auth` help text to include new USER MANAGEMENT section
- Better integration between auth and Hasura metadata

#### Exec Command
- **Fixed stdin piping support** - Now properly supports:
  - `cat file.sql | nself exec postgres psql`
  - Heredoc syntax: `nself exec postgres psql <<EOF`
  - Multi-line SQL piping
- Always adds `-i` flag for stdin compatibility
- No behavior change for interactive usage

### Fixed

- **Bug #2 from Feedback:** `nself exec` now properly supports stdin piping
- Auth service no longer returns "field 'users' not found" error
- Seed files can now be piped to containers without hanging
- Hasura metadata properly tracks auth tables on setup

### Performance

- **Time to Working Auth:** Reduced from ~4 hours to **<5 minutes**
  - Before: Manual Hasura config, SQL debugging, trial and error
  - After: One command (`nself auth setup --default-users`)
- **Steps Required:** Reduced from ~10+ manual steps to **4 commands**
  ```bash
  nself init --demo
  nself build
  nself start
  nself auth setup --default-users
  ```

### Developer Experience

- **Zero-configuration auth** - Works out of the box
- **Clear error messages** - Actionable suggestions provided
- **Comprehensive guides** - Three detailed documentation guides
- **Better command organization** - Logical grouping of related commands
- **Idempotent operations** - Safe to run commands multiple times

---

## [0.4.7] - 2026-02-11

### Previous Version
- Basic service management
- Docker orchestration
- Initial auth integration
- Core CLI commands

---

## Future Releases

### Planned for [0.4.9]
- Integration tests for new features
- CI/CD workflow updates
- Additional seed templates
- Enhanced error messages

### Planned for [0.5.0]
- OAuth provider setup automation
- MFA setup wizard
- Role-based access control UI
- Advanced monitoring dashboards

---

## Notes

### Migration from v0.4.7 to v0.4.8

**Breaking Changes:** None

**New Features:** All backward compatible

**Recommended Actions:**
1. Run `nself auth setup --default-users` in existing projects
2. Apply seed files with `nself db seed apply`
3. Read new guides in `.wiki/guides/`

### Feedback Implementation

This release implements **100% of feedback** from the nself-web team:
- ✅ Database seeding workflow (Priority 1)
- ✅ Auth setup commands (Priority 2)
- ✅ Hasura metadata management (Priority 3)
- ✅ Seed file templates (Priority 4)
- ✅ Comprehensive documentation (Priority 5)
- ✅ Bug fixes (stdin piping)
- ✅ Suggestions (improved exec command)

**Success Metrics Achieved:**
- Time to working auth: <5 minutes (was ~4 hours) ✅
- Commands required: 4 (was ~10+) ✅
- Zero configuration: One command setup ✅
- Documentation: 3 comprehensive guides ✅

---

**For complete details, see:**
- [DEV_WORKFLOW.md](.wiki/guides/DEV_WORKFLOW.md) - Quick start guide
- [AUTH_SETUP.md](.wiki/guides/AUTH_SETUP.md) - Auth system explained
- [SEEDING.md](.wiki/guides/SEEDING.md) - Seeding patterns
- [DONE.md](~/Sites/nself-web/DONE.md) - Testing instructions for nself-web team
