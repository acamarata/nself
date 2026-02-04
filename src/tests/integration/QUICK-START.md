# Integration Tests - Quick Start Guide

## TL;DR

```bash
# Run all integration tests
cd src/tests/integration
./run-all-integration-tests.sh

# Run specific test
./run-all-integration-tests.sh --test full-deployment

# Run with verbose output
./run-all-integration-tests.sh --verbose
```

## What Gets Tested

1. **Full Deployment** - Complete init → build → start → verify workflow
2. **Multi-Tenant** - Tenant lifecycle, members, roles, isolation
3. **Backup/Restore** - Backup creation, restoration, incremental backups
4. **Migrations** - Schema changes, rollback, fresh migrations
5. **Monitoring** - All 10 monitoring services, metrics, dashboards
6. **Custom Services** - CS_N configuration, routing, isolation

## Quick Commands

### Run Single Test
```bash
./test-full-deployment.sh
./test-multi-tenant-workflow.sh
./test-backup-restore-workflow.sh
./test-migration-workflow.sh
./test-monitoring-stack.sh
./test-custom-services-workflow.sh
```

### Debug Failed Test
```bash
# Edit test file, set:
CLEANUP_ON_EXIT=false

# Run test
./test-full-deployment.sh

# Inspect environment
cd /tmp/nself-integration-test-*
docker-compose ps
docker-compose logs <service>
```

### Clean Up Stuck Tests
```bash
# Remove test containers
docker ps -a | grep nself-integration-test | awk '{print $1}' | xargs docker rm -f

# Remove test directories
rm -rf /tmp/nself-integration-test-*
```

## Expected Results

### Success
```
=================================================================
Test Summary
=================================================================
Total Tests: 71
Passed: 71
Failed: 0
Skipped: 0
=================================================================
```

### Failure
```
=================================================================
Test Summary
=================================================================
Total Tests: 71
Passed: 68
Failed: 3
Skipped: 0

Failed Tests:
  - test-full-deployment
  - test-monitoring-stack

[Last 20 lines of failed test output shown]
=================================================================
```

## Runtime

- **Full Deployment**: ~5-7 minutes
- **Multi-Tenant**: ~3-4 minutes
- **Backup/Restore**: ~4-5 minutes
- **Migrations**: ~3-4 minutes
- **Monitoring**: ~6-8 minutes
- **Custom Services**: ~4-5 minutes

**Total**: ~25-35 minutes

## Requirements

- Docker & Docker Compose installed
- 8 GB RAM (minimum 4 GB)
- 20 GB disk space (minimum 10 GB)
- nself installed and in PATH

## Troubleshooting

### Docker not running
```bash
# Check Docker
docker ps

# Start Docker (macOS)
open -a Docker

# Start Docker (Linux)
sudo systemctl start docker
```

### Port conflicts
```bash
# Check what's using port
lsof -i :5432

# Kill process
kill -9 <PID>
```

### Permission denied
```bash
# Make tests executable
chmod +x src/tests/integration/*.sh
```

### Tests timing out
```bash
# Increase timeout in test file
# Edit: DEFAULT_HEALTH_TIMEOUT=240
```

## CI/CD

Tests run automatically on:
- Push to `main` or `develop`
- Pull requests
- Manual trigger (workflow dispatch)

View results: GitHub Actions → Integration Tests

## Getting Help

1. Check `README.md` for detailed docs
2. Review `INTEGRATION-TEST-SUMMARY.md`
3. Run with `--verbose` for debug output
4. Create GitHub issue if stuck

---

**Quick Start Version**: 0.9.8
**Last Updated**: 2026-01-31
