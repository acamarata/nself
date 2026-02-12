# ✅❌ Database Deployment Automation - Partial Success

**Date**: February 12, 2026, 7:00 AM EST
**nself CLI Version**: 0.9.8
**Testing Status**: 🟡 **DATABASE DEPLOYMENT WORKS, BUT DEPLOYMENT HAS CRITICAL NGINX ISSUES**

---

## Executive Summary

The `nself deploy sync full` command **successfully deployed the database** with:
- ✅ 11 migrations applied automatically
- ✅ 39 tables created in database
- ✅ 5 environment-specific seeds applied (000-004 for staging)
- ✅ Health checks passed
- ✅ Database automation works as designed

**HOWEVER**, the deployment revealed **3 critical infrastructure issues** that prevent the GraphQL API from being accessible:
1. ❌ Nginx configs use wrong BASE_DOMAIN (local.nself.org instead of staging.nself.org)
2. ❌ Disabled services (admin, mailpit, frontends) have configs that crash nginx
3. ❌ Hasura metadata not applied (tables exist but not exposed in GraphQL)

**Bottom Line**: Database deployment is production-ready, but the broader deployment workflow needs fixes for nginx and Hasura metadata.

---

## ✅ What Worked: Database Deployment

### Test Environment
- **Server**: staging (167.235.233.65)
- **Command**: `nself deploy sync full staging`
- **Result**: Database successfully deployed

### Deployment Output

```bash
Step 4.5: Database Files (Hasura)
  Syncing migrations... OK
  Syncing seeds... OK
  Syncing metadata... OK
  ✓ Synced 3 Hasura directory/directories

Step 6: Database Deployment
  Environment: staging
  Database: nself_web_db

  Applying migrations...
    → 1644123456789_create_users
    → 1644123456790_create_products
    ... (9 more migrations)
  ✓ Applied 11 migration(s)

  Applying seeds...
    Strategy: System + basic demo data (000-004)
    → 000_production_owner.sql
    → 001_system_roles_and_permissions.sql
    → 002_demo_users.sql
    → 003_demo_licenses.sql
    → 004_demo_organizations_and_servers.sql
    ○ 005_demo_chat_workspace.sql (skipped for staging)
    ○ 006_demo_telemetry_and_stats.sql (skipped for staging)
    ○ 007_subscription_and_verification_seeds.sql (skipped for staging)
  ✓ Applied 5 seed(s)

  ✓ Database has 39 table(s)
  ✓ Database deployment successful

  Running health checks...
    ✓ Database health check passed (39 tables)
```

### Verification Via SSH

```bash
$ ssh root@167.235.233.65 "docker exec nself-web_postgres psql -U postgres -d nself_web_db -c '\dt'"

             List of relations
Schema |        Name         | Type  |  Owner
--------+---------------------+-------+----------
public | audit_logs          | table | postgres
public | badges              | table | postgres
public | channels            | table | postgres
... (39 tables total) ✅
```

### Docker Container Status

```bash
$ ssh root@167.235.233.65 "docker ps"

NAMES                         STATUS
nself-web_postgres            Up (healthy) ✅
nself-web_hasura              Up (healthy) ✅
nself-web_auth                Up (healthy) ✅
nself-web_redis               Up (healthy) ✅
nself-web_minio               Up (healthy) ✅
nself-web_meilisearch         Up (healthy) ✅
nself-web_ping_api            Up (healthy) ✅
nself-web_nginx               Restarting ❌ (crash loop)
```

**Assessment**: Database deployment automation is **100% functional** and production-ready. ✅

---

## ❌ What Failed: Nginx & Hasura Configuration

### Issue #1: Nginx Configuration Not Environment-Aware

**Problem**: Nginx configs are generated locally with `BASE_DOMAIN=local.nself.org` and synced to staging, but staging needs `BASE_DOMAIN=staging.nself.org`.

**Evidence**:
```bash
# On staging server
$ grep BASE_DOMAIN /opt/nself-web/.env
BASE_DOMAIN=staging.nself.org  ✅ Correct in .env

$ grep server_name /opt/nself-web/nginx/sites/hasura.conf
server_name api.local.nself.org;  ❌ Wrong in nginx config
```

**Impact**: GraphQL API returns `405 Not Allowed` because nginx can't match the `api.staging.nself.org` request to any server block.

**Root Cause**: The deployment workflow:
1. Runs `nself build` locally (generates configs for `local.nself.org`)
2. Syncs those configs to remote
3. Remote has different `BASE_DOMAIN` in `.env` but configs aren't regenerated

**Required Fix**:
- Option A: Run `nself build` on remote server after syncing .env (requires nself CLI installed)
- Option B: Run `nself build` locally with target environment's .env before syncing
- Option C: Make nginx configs environment-agnostic (use env vars or wildcards)

---

### Issue #2: Disabled Services Break Nginx

**Problem**: Nginx configs for disabled services (admin, mailpit, frontend apps) are synced to staging and cause nginx to crash.

**Evidence**:
```bash
$ docker logs nself-web_nginx
nginx: [emerg] host not found in upstream "host.docker.internal" in /etc/nginx/sites/admin.conf:11
nginx: [emerg] host not found in upstream "mailpit" in /etc/nginx/sites/mailpit.conf:11
```

**Services That Crashed Nginx**:
1. `admin.conf` - Uses `host.docker.internal:3025` (macOS/Windows only, not Linux)
2. `mailpit.conf` - Mailpit disabled (`MAILPIT_ENABLED=false`)
3. `frontend-*.conf` - 6 frontend configs using `host.docker.internal` (dev-only)

**Manual Fix Applied** (to unblock testing):
```bash
ssh root@167.235.233.65 "cd /opt/nself-web/nginx/sites &&
  mv admin.conf admin.conf.disabled &&
  mv mailpit.conf mailpit.conf.disabled &&
  mv frontend-*.conf *.disabled"
```

After disabling these configs, nginx started successfully. ✅

**Required Fix**:
- Don't sync nginx configs for disabled services
- Or: Use `sites-available/sites-enabled` pattern (symlinks for enabled services only)
- Or: Build step should skip generating configs for disabled services
- Fix `host.docker.internal` usage (only works on Docker Desktop, not Linux)

---

### Issue #3: Hasura Metadata Not Applied

**Problem**: Database has 39 tables, but they're not tracked in Hasura metadata, so GraphQL API returns "no queries available".

**Evidence**:
```bash
# Database has tables ✅
$ ssh root@167.235.233.65 "docker exec nself-web_postgres psql -U postgres -d nself_web_db -c '\dt'" | wc -l
44  # ~39 tables + headers

# GraphQL API broken ❌
$ curl -X POST https://api.staging.nself.org/v1/graphql \
  -d '{"query": "{ users { primary_email } }"}'

{"errors":[{
  "message": "field 'users' not found in type: 'query_root'"
}]}

# Hasura only tracking 9 tables (auth.* tables)
$ curl -X POST https://api.staging.nself.org/v1/metadata \
  -H "x-hasura-admin-secret: XXX" \
  -d '{"type": "export_metadata", "args": {}}' | jq '.sources[0].tables | length'
9  # Should be 39
```

**Root Cause**: The deployment syncs `hasura/metadata/` directory, but:
1. Metadata only has config for 1 table (`newsletter_subscribers`)
2. Deployment doesn't run `hasura metadata apply`
3. Tables need to be tracked before they appear in GraphQL schema

**Attempted Workaround**:
```bash
# Installed Hasura CLI on staging
$ ssh root@167.235.233.65 "curl -L https://github.com/hasura/graphql-engine/releases/download/v2.36.0/cli-hasura-linux-amd64 -o /usr/local/bin/hasura && chmod +x /usr/local/bin/hasura"

# Exported metadata (to track existing tables)
$ ssh root@167.235.233.65 "cd /opt/nself-web/hasura && hasura metadata export"
# Only exported 9 auth.* tables, not public.* tables

# Tried to track public.users
$ curl -X POST https://api.staging.nself.org/v1/metadata \
  -d '{"type": "pg_track_table", "args": {"table": {"schema": "public", "name": "users"}}}'

{"error": "Encountered conflicting definitions... Fields must not be defined more than once"}
# Conflict between public.users and auth.users
```

**Additional Issue Discovered**: Database schema has conflicting tables:
- `auth.users` (from nHost Auth service)
- `public.users` (from our migrations)

Both try to expose as `users` in GraphQL, causing conflicts.

**Required Fixes**:
1. **Short-term**: Deployment should run `hasura metadata apply` after migrations/seeds
2. **Medium-term**: Install Hasura CLI on remote during deployment (or use direct API calls)
3. **Long-term**: Fix database schema design:
   - Use `auth.users` for authentication (don't create duplicate `public.users`)
   - Or: Rename `public.users` to `public.accounts` or similar
   - Or: Configure Hasura custom root fields to avoid naming conflicts

---

## Detailed Test Results

### Test 1: Staging Deployment ✅❌

**Command**:
```bash
cd ~/Sites/nself-web/backend
nself deploy sync full staging
```

**Results**:
| Component | Status | Details |
|-----------|--------|---------|
| SSH Connection | ✅ Pass | Connected to 167.235.233.65 |
| File Sync (.env) | ⚠️  Warning | Failed (permissions?) but worked eventually |
| Nginx Sync | ✅ Pass | 22 files synced |
| Services Sync | ✅ Pass | 1209 files synced (ping_api) |
| Hasura Files Sync | ✅ Pass | migrations, seeds, metadata synced |
| Database Migrations | ✅ Pass | 11 migrations applied |
| Database Seeds | ✅ Pass | 5 seeds applied (staging strategy) |
| Database Health Check | ✅ Pass | 39 tables verified |
| Nginx Startup | ❌ Fail | Crash loop (host.docker.internal errors) |
| GraphQL API | ❌ Fail | 405 / metadata not applied |

**Manual Fixes Required**:
1. Disable admin/mailpit/frontend configs → nginx started ✅
2. Fix nginx server_name domains → API became accessible ✅
3. Install Hasura CLI + apply metadata → Still blocked by schema conflicts ⚠️

---

### Test 2: GraphQL API Testing

**Health Endpoint**:
```bash
$ curl https://api.staging.nself.org/healthz
healthy  ✅
```

**GraphQL Endpoint** (after fixing nginx):
```bash
$ curl -X POST https://api.staging.nself.org/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ users { primary_email role } }"}'

{"errors":[{
  "message": "field 'users' not found in type: 'query_root'",
  "extensions": {"code": "validation-failed"}
}]}  ❌
```

**Introspection Query**:
```bash
$ curl -X POST https://api.staging.nself.org/v1/graphql \
  -d '{"query": "{ __schema { queryType { fields { name }}}}"}'

{
  "data": {
    "__schema": {
      "queryType": {
        "fields": [{"name": "no_queries_available"}]
      }
    }
  }
}  ❌
```

**Tracked Tables**:
```bash
$ curl -X POST https://api.staging.nself.org/v1/metadata \
  -H "x-hasura-admin-secret: XXX" \
  -d '{"type": "export_metadata", "args": {}}' \
  | jq '.sources[0].tables | length'

9  # Only auth.* tables, missing 30+ public.* tables ❌
```

---

## Production Testing: Skipped

**Reason**: Given the critical nginx and metadata issues found in staging, deploying to production would result in the same failures. Production testing should be done only after these issues are fixed.

---

## Required Fixes (Priority Order)

### 🔥 **CRITICAL - Blocks Production Use**

#### 1. Fix Nginx Environment-Aware Configuration

**Problem**: Configs generated with wrong BASE_DOMAIN
**Impact**: API completely inaccessible
**Proposed Solution**:

```bash
# In deploy.sh, after syncing .env, rebuild configs on remote:
sync_env_files() {
  # ... existing sync code ...

  # NEW: Check if nself CLI exists on remote
  if ssh "$remote_user@$remote_host" "command -v nself >/dev/null 2>&1"; then
    cli_info "Rebuilding configs for $ENV environment..."
    ssh "$remote_user@$remote_host" "cd $deploy_path && nself build"
  else
    cli_warn "nself CLI not found on remote - configs not regenerated"
    cli_warn "Install nself CLI on remote for automatic config rebuild"
  fi
}
```

**Alternative**: Make nginx configs use env vars:
```nginx
# Instead of:
server_name api.staging.nself.org;

# Use:
server_name api.${BASE_DOMAIN};  # Requires envsubst in entrypoint
```

---

#### 2. Fix Nginx Configs for Disabled Services

**Problem**: Disabled services crash nginx
**Impact**: Entire API stack unavailable
**Proposed Solution**:

```bash
# In build process, skip nginx config generation for disabled services
generate_nginx_site() {
  local service=$1
  local enabled_var="${service}_ENABLED"

  # Check if service is enabled
  if [[ "${!enabled_var}" != "true" ]]; then
    cli_info "Skipping nginx config for disabled service: $service"
    return 0
  fi

  # ... generate config ...
}
```

**Or use sites-enabled pattern**:
```bash
nginx/
  ├── sites-available/  # All configs
  │   ├── admin.conf
  │   ├── mailpit.conf
  │   └── hasura.conf
  └── sites-enabled/    # Symlinks to enabled configs only
      └── hasura.conf -> ../sites-available/hasura.conf
```

---

#### 3. Apply Hasura Metadata After Deployment

**Problem**: Tables exist but not exposed in GraphQL
**Impact**: API returns "no queries available"
**Proposed Solution**:

```bash
# In deploy.sh, Step 6: Database Deployment
deploy_database() {
  # ... existing migration + seed code ...

  # NEW: Apply Hasura metadata
  cli_info "Applying Hasura metadata..."

  if ssh_command "command -v hasura >/dev/null 2>&1"; then
    # Use Hasura CLI
    ssh_command "cd $deploy_path/hasura && hasura metadata apply"
  else
    # Use direct API call (fallback)
    local admin_secret=$(ssh_command "docker exec ${PROJECT_NAME}_hasura env | grep HASURA_GRAPHQL_ADMIN_SECRET | cut -d'=' -f2")
    ssh_command "curl -X POST http://localhost:8080/v1/metadata \
      -H 'x-hasura-admin-secret: $admin_secret' \
      -d '{\"type\": \"replace_metadata\", \"args\": $(cat $deploy_path/hasura/metadata/metadata.json)}'"
  fi
}
```

**Prerequisite**: Fix metadata.json to include all tables (see Issue #4)

---

### ⚠️ **IMPORTANT - Needed for Complete Functionality**

#### 4. Fix Database Schema Conflicts

**Problem**: `public.users` conflicts with `auth.users`
**Impact**: Cannot track public.users in Hasura
**Proposed Solution** (in nself-web project):

**Option A**: Use auth.users for everything (RECOMMENDED)
```sql
-- In migrations: Don't create public.users
-- Instead, add custom fields to auth.users if needed
ALTER TABLE auth.users ADD COLUMN organization_id UUID REFERENCES organizations(id);
```

**Option B**: Rename public.users
```sql
-- In migration: Rename table
CREATE TABLE accounts (...);  -- Instead of users
```

**Option C**: Use custom GraphQL root fields
```yaml
# In hasura metadata
- table:
    schema: public
    name: users
  configuration:
    custom_root_fields:
      select: appUsers
      select_by_pk: appUser
```

---

#### 5. Fix host.docker.internal Usage

**Problem**: Frontend/admin configs use `host.docker.internal` (Docker Desktop only)
**Impact**: Configs fail on Linux servers
**Solution**:

```nginx
# Instead of:
proxy_pass http://host.docker.internal:3025;

# Use (on Linux):
proxy_pass http://172.17.0.1:3025;  # Docker bridge IP

# Or better (environment-aware):
proxy_pass http://nself-admin:3025;  # Use container name if running in Docker
```

**Or**: Don't generate these configs for staging/prod at all (frontends should be on Vercel)

---

#### 6. Install Hasura CLI on Remote During Deployment

**Problem**: Manual installation required for metadata commands
**Impact**: Extra manual step, error-prone
**Proposed Solution**:

```bash
# In deploy.sh, pre-deployment check
ensure_hasura_cli() {
  if ! ssh_command "command -v hasura >/dev/null 2>&1"; then
    cli_info "Installing Hasura CLI on remote..."
    ssh_command "curl -L https://github.com/hasura/graphql-engine/releases/download/v2.36.0/cli-hasura-linux-amd64 -o /usr/local/bin/hasura && chmod +x /usr/local/bin/hasura"
  fi
}
```

---

### 📋 **NICE TO HAVE - Future Enhancements**

#### 7. Dry-Run Mode for Deployments

From original FEEDWEB.md request (Phase 2):
```bash
nself deploy sync full staging --dry-run
# Shows what would be synced/changed without actually deploying
```

#### 8. Deployment Rollback Support

From original FEEDWEB.md request (Phase 2):
```bash
nself deploy rollback staging
# Rolls back last deployment (database + configs)
```

#### 9. Environment Comparison Tool

From original FEEDWEB.md request (Phase 2):
```bash
nself env diff staging prod
# Shows differences between staging and production configs
```

---

## Summary: What's Blocking Production

| Issue | Severity | Blocks Production? | Workaround Available? |
|-------|----------|--------------------|-----------------------|
| Database deployment | ✅ WORKS | No | N/A - it works! |
| Nginx BASE_DOMAIN | 🔴 Critical | **YES** | Manual sed fix |
| Disabled service configs | 🔴 Critical | **YES** | Manual removal |
| Hasura metadata | 🟠 High | **YES** | Manual hasura CLI |
| Schema conflicts | 🟡 Medium | Partial | Rename tables |
| host.docker.internal | 🟡 Medium | **YES** (on Linux) | Use bridge IP |

**Current State**:
- Database deployment: **Production-ready** ✅
- Overall deployment: **Not production-ready** ❌

**Estimated Fix Time**:
- Critical issues (#1, #2, #3): 4-8 hours
- Important issues (#4, #5, #6): 8-16 hours
- Total: 1-2 days of focused development

---

## Testing Checklist for Next Iteration

When CLI team provides fixes, test:

### ✅ Database Deployment (Already Works)
- [x] Migrations applied automatically
- [x] Seeds applied (environment-specific)
- [x] Health checks pass
- [x] Table count correct (39 tables)

### ❌ Nginx Configuration (Needs Fixes)
- [ ] Configs generated with correct BASE_DOMAIN
- [ ] Disabled services don't have configs
- [ ] Nginx starts without crashes
- [ ] API endpoint accessible (https://api.staging.nself.org)
- [ ] Health endpoint returns 200

### ❌ Hasura Metadata (Needs Fixes)
- [ ] Metadata applied automatically during deployment
- [ ] All tables tracked in Hasura
- [ ] GraphQL introspection shows queries
- [ ] Can query tables via GraphQL
- [ ] No schema conflicts

### 🔄 Full Stack (Ready to Test After Fixes)
- [ ] Staging deployment: 100% functional
- [ ] Production deployment: 100% functional
- [ ] GraphQL APIs work on all environments
- [ ] Frontend apps can connect to APIs

---

## Acknowledgments

**What the CLI Team Got Right** ✅:
1. Database deployment automation is **excellent** - works exactly as designed
2. Environment-aware seeding is perfect (prod: 000-001, staging: 000-004, dev: all)
3. Health checks provide good verification
4. Fallback mode (docker exec) works when nself CLI isn't on remote
5. DONE.md documentation was comprehensive and accurate

**What Needs Work** ❌:
1. Nginx configuration needs to be environment-aware
2. Disabled services shouldn't generate configs
3. Hasura metadata needs to be part of deployment workflow
4. Better testing on Linux environments (host.docker.internal issue)

**Overall Assessment**: The database automation is production-ready, but the broader deployment workflow needs refinement. The issues found are fixable and well-documented here.

---

## Direct Quotes from Testing

> "The deployment succeeded in applying all migrations and seeds. The database has 39 tables as expected. However, the API is completely inaccessible due to nginx configuration issues."

> "After manually fixing nginx configs (wrong domains + disabled services), the API became accessible, but returns 'no queries available' because Hasura metadata wasn't applied during deployment."

> "The database deployment automation works perfectly. The issue is that deployment doesn't regenerate configs for the target environment, and doesn't complete the Hasura setup."

---

**Status**: 🟡 **DATABASE AUTOMATION WORKS, DEPLOYMENT WORKFLOW NEEDS FIXES**

**Next Steps**:
1. Fix 3 critical nginx/metadata issues
2. Re-test staging deployment end-to-end
3. Test production deployment
4. Provide final sign-off

---

— nself-web Team

**P.S.** The database automation you built is solid and production-ready. The deployment issues are separate from the DB automation and are fixable. Great work on the core functionality! 👍

---

*Generated from comprehensive staging deployment testing*
*Testing completed: February 12, 2026, 7:00 AM EST*
*Tested on: staging.nself.org (167.235.233.65)*
