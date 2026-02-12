# 🔴 CRITICAL: Deployment Automation Has Blocking Bugs

**Date**: February 12, 2026, 12:30 PM EST
**nself CLI Version**: 0.9.8 (commit 1d3872b)
**Testing Status**: 🔴 **DEPLOYMENT COMPLETELY BROKEN - CRITICAL BUGS FOUND**

---

## Executive Summary

Tested the new deployment fixes (commit 1d3872b) across staging environment. **RESULT: Deployment fails completely** due to **3 critical bugs** that block all automated deployments.

**Status:**
- ❌ **Bug #1**: `nself hasura` command doesn't exist in minimal installations
- ❌ **Bug #2**: `nself build` crashes on remote servers (bash syntax error)
- ❌ **Bug #3**: Step 1 (.env sync) always fails (permission issues?)
- 🟡 **Secondary**: Config rebuild fails → nginx configs still broken

**Bottom Line**: The deployment automation cannot work until these bugs are fixed. No workarounds possible without manual intervention.

---

## Test Environment

- **Local**: macOS, nself v0.9.8 (full installation from git clone)
- **Staging**: Linux (Hetzner VPS), nself v0.9.8 (minimal installation via install.sh)
- **Command Tested**: `nself deploy sync full staging`
- **nself CLI on Remote**: ✅ Installed at `/usr/local/bin/nself`

---

## 🔴 CRITICAL BUG #1: `nself hasura` Command Doesn't Exist

### Problem

The deployment script tries to run `nself hasura metadata apply` on the remote server, but this command **does not exist** in minimal CLI installations.

### Evidence

**On Local (Full Installation):**
```bash
$ nself hasura
Usage: nself hasura <subcommand>

SUBCOMMANDS:
  metadata apply     Apply metadata
  metadata export    Export metadata
  metadata reload    Reload metadata cache
  console            Open Hasura Console
```
✅ Works locally

**On Staging (Minimal Installation):**
```bash
$ ssh root@167.235.233.65 "nself hasura"
✗ Unknown command: hasura
Run 'nself help' to see available commands
```
❌ Doesn't exist on remote

**Deployment Output:**
```bash
→ Step 6: Database Deployment
  ✓ nself CLI found on remote: /usr/local/bin/nself

Running: nself hasura metadata apply
✗ Unknown command: hasura
Run 'nself help' to see available commands
metadata_failed

  ⚠ Some database commands may have failed
```

### Root Cause

The `hasura.sh` file exists in the source code but **is not included in minimal installations**:

```bash
# Local (full installation)
$ ls ~/Sites/nself/src/cli/hasura.sh
-rwxr-xr-x  1 admin  staff  3134 Feb 11 17:03 hasura.sh  ✅

# Staging (minimal installation)
$ ssh root@167.235.233.65 "ls /root/.nself/src/cli/hasura.sh"
ls: cannot access '/root/.nself/src/cli/hasura.sh': No such file or directory  ❌
```

The minimal installation (via install.sh) doesn't include all command files.

### Impact

- ❌ Hasura metadata is NEVER applied during deployment
- ❌ Tables exist in database but aren't exposed in GraphQL API
- ❌ GraphQL API returns "no queries available"
- ❌ Complete API failure

### Code Location

**File**: `/src/cli/deploy.sh`
**Lines**: 2905-2906

```bash
echo 'Running: nself hasura metadata apply'
nself hasura metadata apply 2>&1 || echo 'metadata_failed'
```

This code assumes `nself hasura` exists, but it doesn't in minimal installations.

### Why Fallback Doesn't Work

The deployment has TWO paths:

**Path A (nself CLI found):** Uses `nself hasura metadata apply` ❌ BROKEN
**Path B (nself CLI NOT found):** Uses direct Hasura CLI or API call ✅ HAS FALLBACK

Since we installed nself CLI on staging, it takes Path A which is broken.

The fallback code (lines 2812-2844) that uses direct API calls is **ONLY in Path B**, not in Path A.

### Required Fix

**Option 1: Include hasura.sh in Minimal Installations**
```bash
# In install.sh, ensure hasura.sh is always included
cp src/cli/hasura.sh $INSTALL_DIR/src/cli/
```

**Option 2: Add Fallback to Path A (RECOMMENDED)**
```bash
# In deploy.sh, check if nself hasura exists before using it
if nself hasura >/dev/null 2>&1; then
  nself hasura metadata apply
else
  # Fall back to direct hasura CLI or API call
  if command -v hasura >/dev/null 2>&1; then
    cd hasura && hasura metadata apply --endpoint http://localhost:8080 --admin-secret "$ADMIN_SECRET"
  else
    # Use direct API call
    curl -X POST http://localhost:8080/v1/metadata \
      -H "x-hasura-admin-secret: $ADMIN_SECRET" \
      -d @hasura/metadata/metadata.json
  fi
fi
```

**Option 3: Always Use Direct Hasura CLI** (not nself wrapper)
```bash
# Don't rely on nself hasura at all
command -v hasura >/dev/null 2>&1 && cd hasura && hasura metadata apply
```

---

## 🔴 CRITICAL BUG #2: `nself build` Crashes on Remote Servers

### Problem

The deployment's Step 1.5 (Rebuild Configuration) tries to run `nself build --force` on the remote server, but it **crashes with a bash syntax error**.

### Evidence

**Deployment Output:**
```bash
→ Step 1.5: Rebuild Configuration
  Rebuilding configs for target environment... PARTIAL
  Note: Configs may use local BASE_DOMAIN
```

**Manual Test on Staging:**
```bash
$ ssh root@167.235.233.65 "cd /opt/nself-web && nself build --force 2>&1 | head -10"

Security Validation
═══════════════════════════════════════

/root/.nself/src/cli/../lib/security/secure-defaults.sh: line 52: errors +   ✓ POSTGRES_PASSWORD: Set
  ✓ REDIS_PASSWORD: Set
  ✓ HASURA_GRAPHQL_ADMIN_SECRET: Set
  ✓ MEILISEARCH_MASTER_KEY: Set
  ✓ MINIO_ROOT_PASSWORD: Set
0: syntax error: operand expected (error token is "✓ POSTGRES_PASSWORD: Set
  ✓ REDIS_PASSWORD: Set
  ✓ HASURA_GRAPHQL_ADMIN_SECRET: Set
  ✓ MEILISEARCH_MASTER_KEY: Set
  ✓ MINIO_ROOT_PASSWORD: Set
0")
```

### Root Cause

**File**: `/src/lib/security/secure-defaults.sh`
**Line**: 52

There's a bash syntax error in the secure-defaults.sh script that causes `nself build` to crash during the security validation phase.

The error suggests a problem with arithmetic operations involving the unicode checkmark character (✓) in the output.

### Impact

- ❌ Config rebuild completely fails
- ❌ Nginx configs use wrong BASE_DOMAIN (local.nself.org instead of staging.nself.org)
- ❌ Disabled services configs (admin, mailpit, frontends) still present
- ❌ Nginx crashes in boot loop due to host.docker.internal errors
- ❌ **ENTIRE API STACK INACCESSIBLE**

### Current State on Staging

```bash
$ ssh root@167.235.233.65 "docker ps | grep nginx"
nself-web_nginx    Restarting (1) 49 seconds ago  ❌ CRASH LOOP

$ ssh root@167.235.233.65 "docker logs nself-web_nginx --tail 5"
nginx: [emerg] host not found in upstream "host.docker.internal" in /etc/nginx/sites/admin.conf:11
```

Because the config rebuild failed, nginx configs still have:
1. Wrong server_name (api.local.nself.org instead of api.staging.nself.org)
2. host.docker.internal references (doesn't exist on Linux)
3. Configs for disabled services (admin, mailpit, frontends)

### Required Fix

**Immediate**: Fix the bash syntax error in `/src/lib/security/secure-defaults.sh` line 52

Likely issues to check:
- Arithmetic operations with string output
- Variable escaping in bash let/$(()) expressions
- Character encoding issues with unicode characters

**Without seeing the exact code, suggestions:**
```bash
# Instead of:
let errors=$errors+$some_output  # If some_output contains unicode, will fail

# Use:
let errors=$errors+1  # Or use proper integer variables only
```

---

## 🔴 CRITICAL BUG #3: Step 1 (.env Sync) Always Fails

### Problem

The deployment's Step 1 (Environment Files) shows FAILED for both .env and .env.secrets sync, but the deployment continues anyway.

### Evidence

**Deployment Output:**
```bash
→ Step 1: Environment Files
  Syncing .env... FAILED
  Syncing .env.secrets... FAILED

→ Step 1.5: Rebuild Configuration
  Rebuilding configs for target environment... PARTIAL
```

### Impact

- ⚠️ Remote server may have stale .env configuration
- ⚠️ Config rebuild uses old .env values
- ⚠️ Service settings may not match what was intended

### Root Cause

Unknown - could be:
1. Permission issues
2. rsync errors
3. File path issues
4. SSH connection problems

### Required Fix

**Need more logging** to diagnose:
```bash
# In deploy.sh, add verbose error output
if ! rsync ... .env; then
  printf "  ${CLI_RED}FAILED${CLI_RESET}\n"
  printf "  ${CLI_DIM}Error: $(last rsync error)${CLI_RESET}\n"  # ADD THIS
else
  printf "  ${CLI_GREEN}OK${CLI_RESET}\n"
fi
```

**Possible fixes:**
1. Check file permissions on remote
2. Ensure deploy_path exists before sync
3. Use scp instead of rsync if rsync isn't available
4. Add --verbose flag to rsync to see actual error

---

## 📊 Deployment Test Results

### Staging Deployment (167.235.233.65)

**Command:**
```bash
cd ~/Sites/nself-web/backend
nself deploy sync full staging
```

**Results:**

| Step | Expected | Actual | Status |
|------|----------|--------|--------|
| **Step 1**: Sync .env | ✅ Success | ❌ FAILED | **BLOCKED** |
| **Step 1.5**: Rebuild configs | ✅ Success | ❌ PARTIAL | **BLOCKED** (Bug #2) |
| **Step 2**: Sync docker-compose | ✅ Success | ❌ FAILED | **BLOCKED** |
| **Step 3**: Sync nginx | ✅ Success | ✅ OK | ✅ |
| **Step 4**: Sync services | ✅ Success | ✅ OK | ✅ |
| **Step 4.5**: Sync hasura | ✅ Success | ✅ OK | ✅ |
| **Step 5**: Restart services | ✅ Success | ✅ OK | ✅ |
| **Step 6**: Database deploy | ✅ Success | 🟡 PARTIAL | **PARTIALLY BLOCKED** |
| └─ Migrations | ✅ Success | ✅ OK (no pending) | ✅ |
| └─ Seeds | ✅ Success | ✅ OK | ✅ |
| └─ Metadata | ✅ Success | ❌ FAILED | **BLOCKED** (Bug #1) |

### Service Status After Deployment

```bash
$ ssh root@167.235.233.65 "docker ps --format 'table {{.Names}}\t{{.Status}}'"

NAMES                         STATUS
nself-web_postgres            Up (healthy)  ✅
nself-web_hasura              Up (healthy)  ✅
nself-web_auth                Up (healthy)  ✅
nself-web_redis               Up (healthy)  ✅
nself-web_minio               Up (healthy)  ✅
nself-web_meilisearch         Up (healthy)  ✅
nself-web_ping_api            Up (healthy)  ✅
nself-web_nginx               Restarting    ❌ CRASH LOOP
```

### API Accessibility

```bash
$ curl https://api.staging.nself.org/healthz
curl: (7) Failed to connect to api.staging.nself.org port 443 after 128 ms: Couldn't connect to server
```
❌ **COMPLETELY INACCESSIBLE** (nginx crashed)

### Database Status

```bash
$ ssh root@167.235.233.65 "docker exec nself-web_postgres psql -U postgres -d nself_web_db -c '\dt' | wc -l"
44
```
✅ Database has ~39 tables (migrations/seeds worked)

### GraphQL API Status

**Cannot test** - nginx is down, API completely inaccessible.

---

## 🔍 Detailed Bug Analysis

### Why Minimal Installation Breaks

The nself CLI has two installation modes:

**Full Installation (git clone):**
- ✅ All command files present
- ✅ All library scripts present
- ✅ nself hasura works
- ✅ nself build works
- **Use case**: Development, local testing

**Minimal Installation (install.sh):**
- ❌ Some command files missing (hasura.sh)
- ❌ Some library scripts broken (secure-defaults.sh syntax error)
- ❌ nself hasura doesn't exist
- ❌ nself build crashes
- **Use case**: Production deployments

**THE PROBLEM**: The deployment automation was tested with full installations, but production uses minimal installations. The minimal installation is **fundamentally broken** for deployment use cases.

### Why This Wasn't Caught

1. **Local testing works** - Developers use full installations
2. **Fallback code exists** - But only for "nself CLI not found" path
3. **Deploy script assumes success** - Continues even when steps fail
4. **No integration tests** - Minimal installation never tested end-to-end

### Cascade of Failures

```
Bug #3: .env sync fails
  ↓
Bug #2: nself build crashes (syntax error)
  ↓
Configs not rebuilt → Wrong BASE_DOMAIN + host.docker.internal
  ↓
Nginx crashes → API inaccessible
  ↓
Bug #1: nself hasura doesn't exist
  ↓
Metadata not applied → GraphQL broken anyway
  ↓
COMPLETE DEPLOYMENT FAILURE
```

---

## 🚨 Blocking Issues Summary

| Issue | Severity | Blocks Deployment? | Workaround? | Fix Complexity |
|-------|----------|-----------------------|-------------|----------------|
| Bug #1: nself hasura missing | 🔴 Critical | **YES** - GraphQL API broken | None | **MEDIUM** - Add to minimal install or add fallback |
| Bug #2: nself build crashes | 🔴 Critical | **YES** - Nginx completely broken | None | **EASY** - Fix bash syntax error |
| Bug #3: .env sync fails | 🟠 High | Partial - Uses stale config | Manual scp | **MEDIUM** - Debug rsync issue |
| Config rebuild (depends on #2) | 🟠 High | **YES** - Wrong domains | Manual sed | Blocked by Bug #2 |
| Nginx crash loop (depends on config rebuild) | 🔴 Critical | **YES** - API inaccessible | Manual config disable | Blocked by Bug #2 |

**All issues are blocking. No deployment can succeed until Bugs #1 and #2 are fixed.**

---

## ✅ What Actually Worked

Despite all the failures, some parts DID work:

1. ✅ **nself CLI Installation** - install.sh successfully installed CLI on remote
2. ✅ **SSH Connection** - Deployment can connect to remote server
3. ✅ **File Syncing** - nginx, services, hasura directories synced successfully
4. ✅ **Docker Restart** - Services restarted correctly
5. ✅ **Database Migrations** - `nself db migrate up` worked perfectly
6. ✅ **Database Seeds** - `nself db seed` worked with correct environment filtering
7. ✅ **Database Health Checks** - Table count verification worked

**The database automation itself is solid.** The issues are all in:
- Minimal installation quality
- Config rebuild implementation
- Hasura metadata automation

---

## 📋 Required Fixes (Priority Order)

### 🔥 CRITICAL - Must Fix for Any Deployment to Work

#### Fix #1: Fix bash Syntax Error in secure-defaults.sh

**Priority**: 🔴 **HIGHEST** - Blocks everything

**File**: `/src/lib/security/secure-defaults.sh`
**Line**: 52

**Action Required**:
1. Identify the exact line causing the syntax error
2. Fix arithmetic operations to handle string output correctly
3. Test `nself build` on a clean Linux server
4. Verify it completes without errors

**Test**:
```bash
# Must succeed without errors
ssh root@167.235.233.65 "cd /opt/nself-web && nself build --force"
```

---

#### Fix #2: Add nself hasura to Minimal Installations OR Add Fallback

**Priority**: 🔴 **HIGHEST** - Blocks GraphQL API

**Option A: Include in Minimal Installation (RECOMMENDED)**

**File**: `/install.sh`

**Change**:
```bash
# Ensure hasura.sh is always included
REQUIRED_FILES=(
  "src/cli/hasura.sh"  # ADD THIS
  "src/cli/db.sh"
  "src/cli/build.sh"
  # ... other required files
)

for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$SOURCE_DIR/$file" ]; then
    error "Missing required file: $file"
  fi
  cp "$SOURCE_DIR/$file" "$INSTALL_DIR/$file"
done
```

**Option B: Add Fallback to Deployment Script**

**File**: `/src/cli/deploy.sh`
**Lines**: 2900-2910

**Change**:
```bash
# Before:
echo 'Running: nself hasura metadata apply'
nself hasura metadata apply 2>&1 || echo 'metadata_failed'

# After:
echo 'Running: nself hasura metadata apply'
if nself hasura >/dev/null 2>&1; then
  nself hasura metadata apply 2>&1 || echo 'metadata_failed'
else
  # Fallback: Use direct hasura CLI or API
  if command -v hasura >/dev/null 2>&1; then
    echo '  ⚠ nself hasura not available, using hasura CLI'
    cd hasura && hasura metadata apply --endpoint http://localhost:8080 --admin-secret \"\$HASURA_GRAPHQL_ADMIN_SECRET\" 2>&1 || echo 'metadata_failed'
    cd ..
  else
    echo '  ⚠ Neither nself hasura nor hasura CLI available'
    if [ -f 'hasura/metadata/metadata.json' ]; then
      echo '  → Attempting direct API call'
      curl -s -X POST http://localhost:8080/v1/metadata \
        -H \"x-hasura-admin-secret: \$HASURA_GRAPHQL_ADMIN_SECRET\" \
        -H \"Content-Type: application/json\" \
        -d @hasura/metadata/metadata.json >/dev/null 2>&1 && echo '  ✓ Metadata applied via API' || echo 'metadata_failed'
    else
      echo 'metadata_failed'
    fi
  fi
fi
```

**Test**:
```bash
# Must succeed and apply metadata
nself deploy sync full staging
# Then:
curl -X POST https://api.staging.nself.org/v1/graphql \
  -d '{"query": "{ __schema { queryType { fields { name }}}}"}'
# Should return queries, not "no_queries_available"
```

---

#### Fix #3: Debug and Fix .env Sync Failures

**Priority**: 🟠 **HIGH** - Affects config accuracy

**File**: `/src/cli/deploy.sh`
**Lines**: ~2450-2470 (env file sync section)

**Action Required**:
1. Add verbose error logging
2. Check file permissions
3. Verify deploy_path exists
4. Test rsync vs scp

**Change**:
```bash
# Add verbose logging
printf "  Syncing .env... "
if rsync -avz "${rsync_args[@]}" ".env" "${user}@${host}:${deploy_path}/.env" 2>&1 | tee /tmp/rsync-env.log; then
  printf "${CLI_GREEN}OK${CLI_RESET}\n"
else
  printf "${CLI_RED}FAILED${CLI_RESET}\n"
  printf "  ${CLI_DIM}Error details:${CLI_RESET}\n"
  tail -5 /tmp/rsync-env.log | sed 's/^/    /'
  printf "  ${CLI_DIM}Attempting fallback with scp...${CLI_RESET}\n"
  if scp ".env" "${user}@${host}:${deploy_path}/.env"; then
    printf "  ${CLI_GREEN}OK (via scp)${CLI_RESET}\n"
  else
    printf "  ${CLI_RED}FAILED (scp also failed)${CLI_RESET}\n"
  fi
fi
```

---

### ⚠️ IMPORTANT - Needed for Full Functionality

#### Fix #4: Config Rebuild Error Handling

**Priority**: 🟡 **MEDIUM** - Depends on Fix #1

**Current**: Continues with "PARTIAL" warning even when build completely fails

**Required**: Treat build failure as critical error

**File**: `/src/cli/deploy.sh`
**Lines**: 2514-2542

**Change**:
```bash
if echo "$rebuild_result" | grep -q "rebuild_failed" || echo "$rebuild_result" | grep -q "syntax error"; then
  printf "${CLI_RED}FAILED${CLI_RESET}\n"
  printf "  ${CLI_DIM}Config rebuild failed - deployment cannot continue${CLI_RESET}\n"
  printf "  ${CLI_DIM}Fix the build errors and try again${CLI_RESET}\n"
  exit 1  # STOP DEPLOYMENT - Don't continue with broken configs
else
  printf "${CLI_GREEN}OK${CLI_RESET}\n"
  printf "  ${CLI_DIM}Configs regenerated with remote .env${CLI_RESET}\n"
fi
```

---

#### Fix #5: Disable Services Config Generation

**Priority**: 🟡 **MEDIUM** - Quality of life

**Problem**: Configs for disabled services still generated and crash nginx

**File**: `/src/cli/build.sh` (nginx config generation section)

**Required**: Skip nginx config generation for disabled services

**Change**:
```bash
# Before generating nginx config for a service:
if [[ "${SERVICE_NAME}_ENABLED" != "true" ]]; then
  cli_info "Skipping nginx config for disabled service: $SERVICE_NAME"
  continue
fi

# Then generate config...
```

---

### 📚 NICE TO HAVE - Future Enhancements

#### Enhancement #1: Minimal Installation Quality

**Action**: Create integration tests for minimal installation

```bash
# Test suite:
1. Install minimal CLI on clean Linux VM
2. Run nself build (must succeed)
3. Run nself db migrate up (must succeed)
4. Run nself hasura metadata apply (must succeed)
5. Deploy to remote with minimal installation
6. Verify all API endpoints accessible
```

#### Enhancement #2: Better Error Messages

**Current**: "PARTIAL", "FAILED" with no details
**Needed**: Specific error messages and troubleshooting steps

#### Enhancement #3: Deployment Rollback

**Current**: Broken deployment leaves system in unusable state
**Needed**: Automatic rollback on critical failures

---

## 🧪 Testing Checklist for Next Iteration

When CLI team provides fixes:

### ✅ Pre-Deployment Tests
- [ ] Fresh minimal installation on clean Linux server succeeds
- [ ] `nself --help` shows all commands including `hasura`
- [ ] `nself build --force` completes without errors
- [ ] `nself hasura metadata apply` works (or fallback exists)

### ✅ Deployment Tests (Staging)
- [ ] Step 1: .env sync succeeds (not FAILED)
- [ ] Step 1.5: Config rebuild succeeds (not PARTIAL)
- [ ] Step 2: docker-compose sync succeeds
- [ ] Step 6: Database automation completes without metadata_failed
- [ ] All Docker containers start successfully (no crash loops)
- [ ] Nginx starts and stays running (not Restarting)

### ✅ API Tests (Staging)
- [ ] Health endpoint accessible: `curl https://api.staging.nself.org/healthz`
- [ ] GraphQL introspection works (returns queries)
- [ ] Can query tables: `{ users { primary_email } }`
- [ ] Server name correct in nginx (staging.nself.org not local.nself.org)
- [ ] No host.docker.internal errors in nginx logs

### ✅ End-to-End Tests
- [ ] Deploy to production works identically to staging
- [ ] Production uses correct seed strategy (000-001 only)
- [ ] Schema consistency across all 3 environments
- [ ] Frontend apps can connect to APIs

---

## 💬 Summary for CLI Team

**Good News** 🎉:
- Database automation (migrations, seeds) works perfectly
- File syncing works
- SSH connectivity works
- Environment-aware seeding is flawless

**Bad News** 🚨:
- Minimal installation is fundamentally broken for deployments
- Three critical bugs block ALL automated deployments
- No workarounds possible without manual intervention

**Critical Path**:
1. Fix bash syntax error in secure-defaults.sh (BLOCKS EVERYTHING)
2. Add nself hasura to minimal install OR add fallback (BLOCKS GRAPHQL API)
3. Debug .env sync failures (AFFECTS CONFIG ACCURACY)

**Estimated Fix Time**:
- Bug #1 (syntax error): 1-2 hours
- Bug #2 (hasura command): 2-4 hours
- Bug #3 (.env sync): 2-4 hours
- **Total**: 5-10 hours of focused development

**Testing Recommendation**:
- Set up automated integration test with minimal installation
- Test full deployment workflow on clean Linux VPS
- Never assume minimal installation == full installation

---

## 📁 Test Artifacts

**Staging Server**: root@167.235.233.65
**Deployment Log**: `/tmp/staging-deploy.log` (on local machine)
**nself Version on Staging**: v0.9.8 (minimal installation)
**Git Commit Tested**: 1d3872b

**Files for CLI Team to Review**:
1. `/root/.nself/src/lib/security/secure-defaults.sh` (line 52) - syntax error
2. `/root/.nself/src/cli/` - missing hasura.sh
3. `/opt/nself-web/.env` - check if sync succeeded

**Commands to Reproduce**:
```bash
# 1. Install nself on clean VPS
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash

# 2. Try to build
cd /opt/nself-web && nself build --force
# RESULT: Syntax error in secure-defaults.sh

# 3. Check for hasura command
nself hasura
# RESULT: Unknown command: hasura

# 4. Try deployment
# From local machine:
cd ~/Sites/nself-web/backend
nself deploy sync full staging
# RESULT: Partial success, nginx crashes, API inaccessible
```

---

**Status**: 🔴 **CRITICAL BUGS - DEPLOYMENT BLOCKED**

**Next Steps**:
1. CLI team fixes 3 critical bugs
2. Re-test full deployment workflow
3. Verify API accessibility on all environments
4. Test schema consistency
5. Final sign-off

---

— nself-web Team

**P.S.** The database automation you built is still excellent. These are integration issues with minimal installations and deployment infrastructure, not with the core database functionality. Fix the bugs above and we'll be ready for production! 💪

---

*Comprehensive testing completed: February 12, 2026, 12:30 PM EST*
*Tested on: staging.nself.org (167.235.233.65)*
*Deployment completely blocked by 3 critical bugs*
