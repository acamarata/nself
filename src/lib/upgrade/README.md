# Zero-Downtime Upgrade Tools

Production-ready upgrade strategies for nself deployments.

## Overview

nself provides two upgrade strategies for production environments:

1. **Blue-Green Deployment** - Zero downtime, instant rollback
2. **Rolling Updates** - Gradual service updates, lower resource usage

## Blue-Green Deployment

### Concept

Blue-green deployment runs two identical production environments (blue and green). Only one serves live traffic at any time.

**Advantages:**
- Zero downtime
- Instant rollback (just switch traffic back)
- Full validation before switching
- Reduces risk

**Trade-offs:**
- Requires 2x resources during deployment
- Database migrations need special handling

### How It Works

```
Current State:
  Blue: Active (serving traffic)
  Green: Idle

Deployment:
  Blue: Active (serving traffic)
  Green: Deploying new version...

Health Checks:
  Blue: Active (serving traffic)
  Green: Running health checks...

Switch Traffic:
  Blue: Idle (kept for rollback)
  Green: Active (serving traffic)

Cleanup (optional):
  Blue: Stopped
  Green: Active (serving traffic)
```

### Usage

**Basic upgrade:**
```bash
nself upgrade perform
```

**Automated upgrade (no prompts):**
```bash
nself upgrade perform --auto-switch --auto-cleanup
```

**With custom health check timeout:**
```bash
nself upgrade perform --timeout 120
```

**Manual process (full control):**
```bash
# 1. Check current status
nself upgrade status

# 2. Deploy to inactive environment
nself upgrade perform --skip-health

# 3. Manually verify the deployment
docker ps
nself health check

# 4. Switch traffic when ready
nself upgrade switch green

# 5. Clean up old deployment
nself upgrade cleanup blue
```

### Health Checks

The upgrade tool checks:
- All critical services running
- Docker health checks passing
- Services responding to requests

Configuration:
```bash
# Increase timeout for slower services
HEALTH_CHECK_TIMEOUT=120 nself upgrade perform

# Reduce check interval
HEALTH_CHECK_INTERVAL=5 nself upgrade perform

# Skip health checks (not recommended)
nself upgrade perform --skip-health
```

### Rollback

If something goes wrong:

```bash
# Instant rollback (switch traffic back)
nself upgrade rollback
```

This switches traffic back to the previous deployment, which is still running.

---

## Rolling Updates

### Concept

Rolling updates gradually update services one at a time, without requiring duplicate infrastructure.

**Advantages:**
- Lower resource requirements
- No 2x capacity needed
- Suitable for resource-constrained environments

**Trade-offs:**
- Brief interruptions possible
- Slower deployment
- More complex rollback

### How It Works

```
Phase 1: Update non-critical services
  MinIO: Updating...
  Redis: Updating...
  Functions: Updating...

Phase 2: Update critical services (one at a time)
  PostgreSQL: Updating...
  (wait for health check)
  Hasura: Updating...
  (wait for health check)
  Auth: Updating...
  (wait for health check)
```

### Usage

```bash
# Perform rolling update
nself upgrade rolling
```

The tool automatically:
1. Identifies critical vs non-critical services
2. Updates non-critical services first
3. Updates critical services one by one
4. Waits for health checks between updates

---

## Deployment Status

### Check Current Deployment

```bash
nself upgrade status
```

Output:
```
╔════════════════════════════════════════╗
║       Deployment Status                ║
╚════════════════════════════════════════╝

Active Deployment:  green
Standby Deployment: blue

Containers Running: 25 / 25

Recent Deployments:
  2024-01-29T10:15:30Z | blue → green
  2024-01-28T14:22:10Z | green → blue
  2024-01-27T09:45:20Z | blue → green
```

### Manual Traffic Switch

```bash
# Switch to green deployment
nself upgrade switch green

# Switch to blue deployment
nself upgrade switch blue
```

### Cleanup Old Deployments

```bash
# Stop but don't remove (for rollback)
nself upgrade cleanup blue

# Fully remove
docker-compose down
```

---

## Database Migrations

### Backward-Compatible Migrations

For zero-downtime deployments, database migrations must be backward-compatible:

**Safe migrations:**
- Adding new tables ✅
- Adding nullable columns ✅
- Adding indexes ✅
- Adding new functions ✅

**Unsafe migrations (require downtime):**
- Removing columns ❌
- Renaming columns ❌
- Changing column types ❌
- Removing tables ❌

### Migration Strategy

**Approach 1: Expand-Migrate-Contract**

```sql
-- Step 1: Expand (add new column)
ALTER TABLE users ADD COLUMN full_name TEXT;

-- Deploy new code that writes to both columns
-- [Blue-green deployment]

-- Step 2: Migrate data
UPDATE users SET full_name = first_name || ' ' || last_name;

-- Step 3: Contract (remove old columns) - in next deployment
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;
```

**Approach 2: Feature Flags**

```bash
# Deploy with feature flag off
FEATURE_NEW_SCHEMA=false nself upgrade perform

# Switch traffic, enable feature
nself env set FEATURE_NEW_SCHEMA=true
```

**Approach 3: Blue-Green with Database Snapshot**

```bash
# Before deployment
nself db backup --label "pre-upgrade-v2.0"

# Deploy new version
nself upgrade perform

# If database migration fails
nself upgrade rollback
nself db restore --backup "pre-upgrade-v2.0"
```

---

## Environment Variables

Configure upgrade behavior:

```bash
# Blue-green deployment
DEPLOYMENT_DIR=.nself/deployments    # Where to store deployment metadata
ROLLBACK_LIMIT=5                      # Keep last 5 deployment snapshots
HEALTH_CHECK_TIMEOUT=60               # Health check timeout (seconds)
HEALTH_CHECK_INTERVAL=2               # Check interval (seconds)
AUTO_SWITCH=false                     # Auto switch traffic after health checks
AUTO_CLEANUP=false                    # Auto cleanup old deployment

# Example
AUTO_SWITCH=true AUTO_CLEANUP=true nself upgrade perform
```

---

## Deployment Snapshots

Every deployment creates a snapshot for rollback:

```
.nself/deployments/
├── blue-20240129-101530/
│   ├── .env
│   ├── docker-compose.yml
│   ├── containers.txt
│   └── metadata.json
├── green-20240129-141245/
│   ├── .env
│   ├── docker-compose.yml
│   ├── containers.txt
│   └── metadata.json
├── history.json
└── active
```

**Metadata includes:**
- Timestamp
- Git commit hash
- Git branch
- Container list
- Environment configuration

---

## Production Deployment Workflow

### Recommended Process

```bash
# 1. Pre-deployment checks
nself doctor                         # Verify system health
nself upgrade check                  # Check for updates
git status                           # Ensure clean state

# 2. Create backup
nself db backup --label "pre-v2.0-deploy"

# 3. Run deployment
nself upgrade perform --auto-switch

# 4. Post-deployment verification
nself health check                   # Verify all services
nself urls                           # Test all endpoints
nself db query "SELECT COUNT(*) FROM users;"  # Verify database

# 5. Monitor
nself logs -f                        # Watch logs
nself status --watch                 # Monitor service health

# 6. If issues detected
nself upgrade rollback               # Instant rollback
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Deploy to production
  run: |
    nself upgrade perform \
      --auto-switch \
      --timeout 120

- name: Verify deployment
  run: |
    nself health check
    nself status

- name: Rollback on failure
  if: failure()
  run: |
    nself upgrade rollback
```

---

## Monitoring During Deployment

### Watch deployment progress

```bash
# In separate terminal windows

# Terminal 1: Watch logs
nself logs -f

# Terminal 2: Monitor containers
watch -n 2 'docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"'

# Terminal 3: Monitor database
watch -n 5 'nself db query "SELECT COUNT(*) FROM pg_stat_activity;"'

# Terminal 4: Monitor HTTP endpoints
while true; do
  curl -s https://api.yourdomain.com/health
  sleep 2
done
```

### Deployment metrics

```bash
# Container resource usage during deployment
docker stats

# Service response times
nself health check --detailed

# Database connections
nself db query "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';"
```

---

## Troubleshooting

### Deployment Stuck

```bash
# Check container logs
nself logs postgres
nself logs hasura

# Check health status
docker inspect --format='{{.State.Health.Status}}' project_postgres

# Force cleanup
docker-compose down
nself upgrade perform --skip-health
```

### Traffic Switch Failed

```bash
# Check nginx status
docker logs project_nginx

# Manual reload
docker exec project_nginx nginx -s reload

# Verify routing
curl -I https://yourdomain.com
```

### Health Checks Failing

```bash
# Increase timeout
nself upgrade perform --timeout 180

# Skip health checks (not recommended)
nself upgrade perform --skip-health

# Check specific service
docker exec project_postgres pg_isready
docker exec project_hasura curl -f http://localhost:8080/healthz
```

### Rollback Failed

```bash
# Check previous deployment status
docker ps -a | grep project

# Manually restart old deployment
docker-compose -f .nself/deployments/blue-*/docker-compose.yml up -d

# Switch traffic manually
nself upgrade switch blue
```

---

## Advanced Scenarios

### Multi-Region Deployment

```bash
# Deploy to primary region
nself upgrade perform --auto-switch

# Replicate to secondary regions
nself sync push staging --region=us-west
nself sync push staging --region=eu-central

# Update DNS/load balancer
```

### Canary Deployment

```bash
# Deploy new version to green
nself upgrade perform

# Route 10% traffic to green (requires load balancer config)
# Monitor metrics

# Route 50% traffic to green
# Monitor metrics

# Route 100% traffic to green
nself upgrade switch green
```

### A/B Testing

```bash
# Run both versions simultaneously
nself upgrade perform

# Don't cleanup old deployment
# Configure load balancer to split traffic based on criteria

# After testing period
nself upgrade switch green
nself upgrade cleanup blue
```

---

## Best Practices

1. **Always backup before upgrades**
   ```bash
   nself db backup --label "pre-upgrade"
   ```

2. **Test in staging first**
   ```bash
   ENV=staging nself upgrade perform
   ```

3. **Monitor during deployment**
   ```bash
   nself logs -f
   ```

4. **Verify after deployment**
   ```bash
   nself health check
   nself status
   ```

5. **Keep rollback ready**
   - Don't cleanup old deployment immediately
   - Verify new deployment for 24h first

6. **Schedule during low-traffic**
   - Minimize impact of any issues
   - Easier to detect problems

7. **Communicate with team**
   - Notify before deployment
   - Share deployment status
   - Document any issues

---

## Future Enhancements

Coming soon:
- [ ] Kubernetes blue-green deployment
- [ ] Automated canary deployments
- [ ] Traffic splitting configuration
- [ ] Deployment approval workflows
- [ ] Slack/Discord notifications
- [ ] Prometheus metrics integration
- [ ] Automatic rollback on error rate spike

---

## Philosophy

> **Zero downtime is not a luxury - it's a requirement for production systems.**

nself upgrade tools are designed to make production deployments safe, predictable, and stress-free. No vendor lock-in, full control over your upgrade process.

**Free forever. Open source. Production-ready.**
