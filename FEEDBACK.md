# nself CLI v0.9.8+ QA Testing Results - Source Version

**Date:** February 11, 2026, 8:45 PM EST
**Tester:** nself-web Team (Real-world Production Usage)
**Test Environment:** macOS, Docker Desktop, nself-web monorepo
**nself Version Tested:** Source version at `~/Sites/nself/bin/nself`
**Previous Test Results:** 5/9 services healthy (installed version)
**Current Test Results:** **6/9 services healthy (source version)** - ⚠️ **PARTIAL IMPROVEMENT**

---

## 🎯 EXECUTIVE SUMMARY

**Status:** 🟡 **MIXED RESULTS - 1/3 Critical Bugs Fixed, 2/3 Still Broken**

Testing the source version shows improvement over the installed version, but the deployment is still not production-ready. Out of 3 critical bugs identified in previous testing:

- ✅ **FIXED:** Hasura CORS configuration (1/3)
- ❌ **STILL BROKEN:** MinIO volume permissions (2/3)
- ❌ **STILL BROKEN:** MeiliSearch volume permissions (3/3)

**Key Discovery:** The init containers claimed to be implemented in DONE.md are **NOT present** in the generated `docker-compose.yml` file, causing persistent permission failures.

---

## 📊 TEST RESULTS BY SERVICE

### ✅ Healthy Services (6/9)

| Service | Status | Notes |
|---------|--------|-------|
| postgres | ✅ Healthy | Working perfectly |
| hasura | ✅ **FIXED!** | CORS variable fix worked |
| auth | ✅ Healthy | Working perfectly |
| redis | ✅ Healthy | Working perfectly |
| mailpit | ✅ Healthy | Working perfectly |
| ping_api | ✅ Healthy | Custom service working |

### ❌ Failing Services (3/9)

| Service | Status | Root Cause | Fix Needed |
|---------|--------|------------|------------|
| minio | ❌ Crash-loop | Volume permission denied | Init container missing |
| meilisearch | ❌ Crash-loop | Volume permission denied | Init container missing |
| nginx | ❌ Won't start | Can't find meilisearch upstream | Cascade failure from above |

---

## ✅ WHAT WORKS - HASURA CORS FIX (BUG #1)

### Test Scenario
Clean slate test with all volumes removed, testing if Hasura starts without CORS errors.

### Expected Behavior
Hasura should start successfully with `HASURA_GRAPHQL_CORS_DOMAIN` configured.

### Actual Behavior
✅ **SUCCESS** - Hasura is now healthy!

### Evidence
```bash
# Service status
$ ~/Sites/nself/bin/nself status
✓ hasura

# No CORS errors in logs
$ docker logs nself-web_hasura --tail 50 | grep -i "cors\|error\|fatal"
# No "invalid domain" errors found
```

### How It Works
The `.environments/dev/.env` file now contains:
```bash
HASURA_GRAPHQL_CORS_DOMAIN=https://*.nself.org,https://*.local.nself.org,http://localhost:*
```

This variable is properly loaded and Hasura accepts it without errors.

### Verdict
✅ **COMPLETE FIX** - This bug is fully resolved. Hasura CORS configuration works out-of-box.

---

## ❌ WHAT'S BROKEN - VOLUME PERMISSIONS (BUGS #2 & #3)

### Test Scenario
Clean slate test with all volumes removed, testing if MinIO and MeiliSearch can write to their data volumes.

### Expected Behavior
According to DONE.md:
> "Added `networks: - ${DOCKER_NETWORK}` to both init containers so they can run properly."
> "Init containers are in docker-compose.yml"

Init containers should:
1. Be present in `docker-compose.yml`
2. Run before MinIO and MeiliSearch
3. Fix volume permissions via `chown -R 1000:1000 /data`

### Actual Behavior
❌ **CRITICAL FAILURE** - Init containers are **NOT present** in generated `docker-compose.yml`

### Evidence

**Proof #1: Init containers don't exist in docker-compose.yml**
```bash
$ grep "minio-init:" ~/Sites/nself-web/backend/docker-compose.yml
# NO OUTPUT - Container definition not found

$ grep "meilisearch-init:" ~/Sites/nself-web/backend/docker-compose.yml
# NO OUTPUT - Container definition not found
```

**Proof #2: Init containers never ran**
```bash
$ docker ps -a | grep init
# NO OUTPUT - No init containers exist
# (only "dumb-init" from ping_api startup, not an init container)
```

**Proof #3: MinIO crashes with permission errors**
```bash
$ docker logs nself-web_minio --tail 20

FATAL Unable to initialize backend: file access denied

API: SYSTEM.storage
Time: 20:44:48 UTC 02/11/2026
Error: unable to rename (/data/.minio.sys/tmp -> /data/.minio.sys/tmp-old/...)
file access denied, drive may be faulty, please investigate
```

**Proof #4: MeiliSearch crashes with permission errors**
```bash
$ docker logs nself-web_meilisearch --tail 20

Error: Permission denied (os error 13)
Error: Permission denied (os error 13)
Error: Permission denied (os error 13)
```

**Proof #5: Nginx fails because meilisearch is down**
```bash
$ docker logs nself-web_nginx --tail 10

2026/02/11 20:44:57 [emerg] 1#1: host not found in upstream "meilisearch"
in /etc/nginx/sites/search.conf:11
nginx: [emerg] host not found in upstream "meilisearch" in /etc/nginx/sites/search.conf:11
```

### Root Cause Analysis

**Why volumes fail:**
1. Docker creates volumes owned by `root:root` with mode `755`
2. MinIO and MeiliSearch containers run as unprivileged user (UID `1000:1000`)
3. User 1000 cannot write to root-owned directories
4. Services crash-loop with "permission denied"

**Why init containers should fix this:**
1. Init container runs as `root` (has permissions to chown)
2. Changes volume ownership to `1000:1000`
3. Main service container can now write to volume

**Why it's not working:**
The init container code was **never added to the source templates**. When `nself build` runs, it generates `docker-compose.yml` from templates, but the templates don't include init container definitions.

### Files That Need Changes

Based on DONE.md claims, these files should have init containers:
- `~/Sites/nself/src/services/docker/compose-modules/core-services.sh` (minio-init)
- `~/Sites/nself/src/services/docker/compose-modules/utility-services.sh` (meilisearch-init)

**However**, these changes were NOT actually made to the source code.

### Verdict
❌ **NOT FIXED** - The DONE.md claims are incorrect. Init containers are missing from source.

---

## 🔬 DETAILED TEST METHODOLOGY

### Clean Slate Test Procedure

To ensure unbiased results, we performed a complete clean slate test:

```bash
# Step 1: Stop all services
cd ~/Sites/nself-web/backend
~/Sites/nself/bin/nself stop

# Step 2: Remove ALL volumes (complete clean slate)
docker volume ls | grep nself-web | awk '{print $2}' | xargs docker volume rm

# Step 3: Rebuild from source
~/Sites/nself/bin/nself build

# Step 4: Verify configuration files
cat .environments/dev/.env | grep HASURA_GRAPHQL_CORS_DOMAIN
# Result: ✅ Variable exists with correct value

# Step 5: Check if init containers are in generated docker-compose.yml
grep "minio-init:" docker-compose.yml
# Result: ❌ NOT FOUND

grep "meilisearch-init:" docker-compose.yml
# Result: ❌ NOT FOUND

# Step 6: Start services
~/Sites/nself/bin/nself start

# Step 7: Wait for startup (30 seconds)
sleep 30

# Step 8: Check service health
~/Sites/nself/bin/nself status
# Result: 6/9 services healthy
#   ✅ postgres, hasura, auth, redis, mailpit, ping_api
#   ❌ minio, meilisearch, nginx

# Step 9: Verify Hasura is healthy (CORS fix check)
docker logs nself-web_hasura | grep -i "cors\|error\|fatal"
# Result: ✅ No CORS errors, Hasura running normally

# Step 10: Check MinIO logs (permission check)
docker logs nself-web_minio --tail 20
# Result: ❌ "FATAL Unable to initialize backend: file access denied"

# Step 11: Check MeiliSearch logs (permission check)
docker logs nself-web_meilisearch --tail 20
# Result: ❌ "Error: Permission denied (os error 13)"

# Step 12: Check nginx logs (cascade failure check)
docker logs nself-web_nginx --tail 10
# Result: ❌ "host not found in upstream meilisearch"
```

### Test Environment Details

```bash
# Operating System
$ uname -a
Darwin 25.2.0 (macOS)

# Docker Version
$ docker --version
Docker version 28.x.x

# nself CLI Location
$ which ~/Sites/nself/bin/nself
/Users/admin/Sites/nself/bin/nself  # Source version, NOT installed

# Project Location
/Users/admin/Sites/nself-web/backend

# Environment File
.environments/dev/.env  # Using v0.9.8+ multi-environment structure

# Generated Files (from nself build)
- docker-compose.yml (generated, missing init containers)
- nginx/sites/*.conf (generated correctly)
- .env (symlink to .environments/dev/.env)
```

---

## 🎯 COMPARISON: SOURCE VS INSTALLED VERSION

| Metric | Installed Version | Source Version | Improvement |
|--------|------------------|----------------|-------------|
| Healthy Services | 5/9 | 6/9 | +1 |
| Hasura Status | ❌ CORS error | ✅ Healthy | ✅ Fixed |
| MinIO Status | ❌ Permission denied | ❌ Permission denied | ⚠️ No change |
| MeiliSearch Status | ❌ Permission denied | ❌ Permission denied | ⚠️ No change |
| Nginx Status | ❌ Dependency failure | ❌ Dependency failure | ⚠️ No change |
| Init Containers | Missing | Missing | ⚠️ No change |

**Conclusion:** Source version is better (1 bug fixed) but not production-ready (2 bugs remain).

---

## 🔧 WHAT NEEDS TO BE FIXED

### Critical Issue: Init Containers Not in Source Code

The DONE.md stated:
> "🔥 CRITICAL FIX #2: Init Containers Now Include Network"
> "Added `networks: - ${DOCKER_NETWORK}` to both init containers"
> "Files Modified: src/services/docker/compose-modules/core-services.sh (minio-init)"
> "Files Modified: src/services/docker/compose-modules/utility-services.sh (meilisearch-init)"

**Reality:** These files were NOT modified. Init containers are NOT in the generated output.

### Required Code Changes

#### Option A: Add Init Containers to Templates (Recommended)

**File:** `~/Sites/nself/src/services/docker/compose-modules/core-services.sh`

Add minio-init container before minio service:

```bash
cat >> docker-compose.yml << 'EOF'
  minio-init:
    image: busybox:latest
    container_name: ${PROJECT_NAME}_minio_init
    user: root
    networks:
      - ${DOCKER_NETWORK}
    volumes:
      - minio_data:/data
    command: >
      sh -c "
        echo '→ Fixing MinIO volume permissions...';
        chown -R 1000:1000 /data;
        chmod -R 755 /data;
        echo '✓ MinIO volume permissions fixed';
      "

EOF
```

Then add `depends_on: minio-init` to the minio service.

**File:** `~/Sites/nself/src/services/docker/compose-modules/utility-services.sh`

Add meilisearch-init container before meilisearch service:

```bash
cat >> docker-compose.yml << 'EOF'
  meilisearch-init:
    image: busybox:latest
    container_name: ${PROJECT_NAME}_meilisearch_init
    user: root
    networks:
      - ${DOCKER_NETWORK}
    volumes:
      - meilisearch_data:/meili_data
    command: >
      sh -c "
        echo '→ Fixing MeiliSearch volume permissions...';
        chown -R 1000:1000 /meili_data;
        chmod -R 755 /meili_data;
        echo '✓ MeiliSearch volume permissions fixed';
      "

EOF
```

Then add `depends_on: meilisearch-init` to the meilisearch service.

#### Option B: Run as Root User (Not Recommended for Security)

Modify service definitions to run as root:
```yaml
minio:
  user: root  # Security risk
```

**Why not recommended:** Running services as root increases attack surface.

#### Option C: Use Named Volumes with External Init (Complex)

Create volumes externally with correct ownership before `nself start`:
```bash
docker volume create --driver local \
  --opt type=none \
  --opt device=/tmp/minio-data \
  --opt o=bind,uid=1000,gid=1000 \
  minio_data
```

**Why not recommended:** Requires manual setup, defeats "works out-of-box" goal.

### Recommended Fix: Option A (Init Containers)

This is the approach claimed in DONE.md and provides the best balance of:
- ✅ Security (services run as non-root)
- ✅ Automation (works out-of-box)
- ✅ Compatibility (works on all platforms)
- ✅ Maintainability (standard Docker pattern)

---

## 📋 VERIFICATION CHECKLIST FOR NEXT RELEASE

When the CLI team implements the init container fixes, verify with these tests:

### Pre-Flight Checks

```bash
# 1. Verify source code changes were made
$ grep -n "minio-init:" ~/Sites/nself/src/services/docker/compose-modules/core-services.sh
# Should show line numbers where minio-init is defined

$ grep -n "meilisearch-init:" ~/Sites/nself/src/services/docker/compose-modules/utility-services.sh
# Should show line numbers where meilisearch-init is defined
```

### Build-Time Checks

```bash
# 2. Generate docker-compose.yml and verify init containers are present
cd ~/Sites/nself-web/backend
~/Sites/nself/bin/nself build

$ grep -A 15 "minio-init:" docker-compose.yml
# Should show full minio-init service definition with networks

$ grep -A 15 "meilisearch-init:" docker-compose.yml
# Should show full meilisearch-init service definition with networks
```

### Runtime Checks

```bash
# 3. Clean slate test
docker volume ls | grep nself-web | awk '{print $2}' | xargs docker volume rm
~/Sites/nself/bin/nself start
sleep 30

# 4. Verify init containers ran successfully
$ docker ps -a | grep init
# Should show:
# nself-web_minio_init        busybox:latest   "sh -c ..."   X minutes ago   Exited (0)
# nself-web_meilisearch_init  busybox:latest   "sh -c ..."   X minutes ago   Exited (0)

# 5. Check init container logs
$ docker logs nself-web_minio_init
# Expected output:
# → Fixing MinIO volume permissions...
# ✓ MinIO volume permissions fixed

$ docker logs nself-web_meilisearch_init
# Expected output:
# → Fixing MeiliSearch volume permissions...
# ✓ MeiliSearch volume permissions fixed

# 6. Verify all services are healthy
$ ~/Sites/nself/bin/nself status
# Expected: 9/9 services healthy
#   ✓ postgres
#   ✓ hasura
#   ✓ auth
#   ✓ nginx          ← Should now work (meilisearch is up)
#   ✓ minio          ← Should now work (permissions fixed)
#   ✓ redis
#   ✓ mailpit
#   ✓ meilisearch    ← Should now work (permissions fixed)
#   ✓ ping_api

# 7. Verify no permission errors in logs
$ docker logs nself-web_minio --tail 50 | grep -i "permission\|denied\|fatal"
# Expected: No errors

$ docker logs nself-web_meilisearch --tail 50 | grep -i "permission\|denied"
# Expected: No errors

# 8. Verify nginx is routing correctly
$ docker logs nself-web_nginx --tail 20 | grep -i "error\|emerg"
# Expected: No "host not found" errors
```

### Integration Checks

```bash
# 9. Test actual service functionality

# MinIO health check
$ curl -I http://localhost:9000/minio/health/live
# Expected: HTTP 200 OK

# MeiliSearch health check
$ curl http://localhost:7700/health
# Expected: {"status": "available"}

# Nginx routing to MeiliSearch
$ curl -k https://search.local.nself.org/health
# Expected: {"status": "available"}
```

### Success Criteria

✅ **Release is ready when ALL of these are true:**

1. Init container definitions exist in source template files
2. Init containers appear in generated docker-compose.yml
3. Init containers run successfully (exit code 0)
4. Init container logs show permission fixes applied
5. All 9 services report healthy status
6. No permission errors in MinIO logs
7. No permission errors in MeiliSearch logs
8. Nginx routes successfully to all upstream services
9. Service functionality tests pass (health endpoints respond)

---

## 💡 LESSONS LEARNED & RECOMMENDATIONS

### What Went Well

1. **CORS Fix Implementation:** The Hasura CORS fix works perfectly. Variables are loaded correctly from `.environments/dev/.env` and Hasura accepts them without errors.

2. **Source vs Installed Discovery:** Understanding the distinction between source and installed versions helped identify where fixes were actually applied.

3. **Clean Slate Testing:** Complete volume removal ensured unbiased test results.

### What Needs Improvement

1. **Code Review Before Release:** The DONE.md claimed init containers were implemented, but they weren't in the source code. This suggests fixes were planned but not actually committed.

2. **Automated Testing:** Consider adding integration tests that verify:
   - All services start successfully
   - No permission errors in logs
   - All health checks pass
   - Generated docker-compose.yml contains expected components

3. **Build Verification:** After running `nself build`, automatically verify that generated files contain expected components (like init containers).

4. **Documentation Accuracy:** Ensure DONE.md / release notes only claim fixes that are actually present in the codebase.

### Recommendations for Release Process

1. **Pre-Release QA Checklist:**
   - ✅ Code changes committed to repository
   - ✅ `nself build` generates expected output
   - ✅ Clean slate test shows 9/9 healthy services
   - ✅ All health checks pass
   - ✅ No errors in service logs

2. **Automated Build Verification:**
   ```bash
   # After nself build, verify critical components
   if ! grep -q "minio-init:" docker-compose.yml; then
     echo "❌ ERROR: minio-init not found in generated docker-compose.yml"
     exit 1
   fi
   ```

3. **Integration Test Suite:**
   ```bash
   # test.sh - Run after nself start
   #!/bin/bash

   # Wait for services
   sleep 30

   # Check status
   STATUS=$(nself status --json)
   HEALTHY=$(echo "$STATUS" | jq '.healthy_count')
   TOTAL=$(echo "$STATUS" | jq '.total_count')

   if [ "$HEALTHY" -ne "$TOTAL" ]; then
     echo "❌ Only $HEALTHY/$TOTAL services healthy"
     exit 1
   fi

   echo "✅ All services healthy"
   ```

---

## 🎯 BOTTOM LINE

### Current State
- **6/9 services healthy** (source version)
- **1/3 critical bugs fixed** (Hasura CORS)
- **2/3 critical bugs remain** (volume permissions)

### Remaining Work
The init containers need to be **actually added to the source code templates**, not just documented in DONE.md.

**Estimated effort:** 30-60 minutes to add init containers to templates + testing.

**Files to modify:**
1. `~/Sites/nself/src/services/docker/compose-modules/core-services.sh`
2. `~/Sites/nself/src/services/docker/compose-modules/utility-services.sh`

### Expected Outcome After Fix
- **9/9 services healthy** out-of-box
- Clean slate test passes 100%
- Production-ready for dev, staging, and prod environments

---

## 📞 NEXT STEPS

### For CLI Team

1. **Add init containers to source templates** (both minio-init and meilisearch-init)
2. **Verify changes** using the pre-flight checklist above
3. **Run clean slate test** to confirm 9/9 services healthy
4. **Update DONE.md** with proof (screenshot or logs showing 9/9 healthy)
5. **Tag release** only after verification passes

### For nself-web Team

1. **Wait for next DONE.md** from CLI team
2. **Re-test with source nself** using verification checklist
3. **Report results** (pass/fail with evidence)
4. **If all tests pass:** Install updated nself and deploy to staging

---

## 📝 TEST LOG SUMMARY

```
Test Date: February 11, 2026, 8:45 PM EST
Test Duration: ~15 minutes (clean slate + verification)
Test Method: Complete environment rebuild from scratch

Environment:
  nself CLI: ~/Sites/nself/bin/nself (source version)
  Project: ~/Sites/nself-web/backend
  Config: .environments/dev/.env
  Platform: macOS (Docker Desktop)

Results:
  Services Started: 9/9
  Services Healthy: 6/9
  Services Failed: 3/9

  ✅ Healthy:
    - postgres
    - hasura (previously failing, now fixed!)
    - auth
    - redis
    - mailpit
    - ping_api

  ❌ Failed:
    - minio (permission denied - init container missing)
    - meilisearch (permission denied - init container missing)
    - nginx (cascade failure - can't find meilisearch)

Critical Findings:
  1. ✅ HASURA CORS FIX WORKS - Variable loads correctly, no CORS errors
  2. ❌ INIT CONTAINERS MISSING - Not in docker-compose.yml despite DONE.md claims
  3. ❌ VOLUME PERMISSIONS UNFIXED - MinIO and MeiliSearch still crash-loop
  4. ⚠️ SOURCE BETTER THAN INSTALLED - But still not production-ready

Recommendation: ⚠️ DO NOT RELEASE YET
  - Add init containers to source templates
  - Re-test for 9/9 healthy services
  - Then tag release
```

---

## 🙏 ACKNOWLEDGMENTS

Thank you to the nself CLI team for:
- ✅ Fixing the Hasura CORS bug (works perfectly!)
- 📋 Detailed DONE.md explaining source vs installed distinction
- 🔍 Root cause analysis on environment variable loading

The CORS fix demonstrates the team can implement solutions correctly. The init container fix just needs to be actually added to the source code templates.

**We're close!** One more iteration should get us to 9/9 healthy services out-of-box.

---

**Contact:** nself-web Team
**Test Environment:** Production monorepo (nself eating its own dog food)
**Status:** ⏳ Awaiting init container implementation in source templates
**Priority:** 🔴 High - Blocks local development, staging, and production deployments
