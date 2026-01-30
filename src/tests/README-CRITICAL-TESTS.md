# Critical Path Tests - Quick Reference

**Added**: 2026-01-30
**Coverage Improvement**: 30% → 60%+
**Total New Tests**: 77

## Quick Start

### Run All Critical Tests
```bash
cd src/tests
./run-critical-tests.sh
```

### Run Individual Test Suites
```bash
# Backup & Restore (16 tests)
bats backup_tests.bats

# Deploy (10 tests)
bats deploy_tests.bats

# Storage (15 tests)
bats storage_tests.bats

# SSL/TLS (16 tests)
bats ssl_tests.bats

# Multi-Tenancy (20 tests)
bats tenant_tests.bats
```

## Test Summary

| Test Suite | Tests | Docker Required | Coverage |
|------------|-------|-----------------|----------|
| `backup_tests.bats` | 16 | Some | 30% → 85% |
| `deploy_tests.bats` | 10 | Some | 0% → 60% |
| `storage_tests.bats` | 15 | Most | 20% → 75% |
| `ssl_tests.bats` | 16 | Few | Partial → 80% |
| `tenant_tests.bats` | 20 | Most | 70% → 90% |
| **Total** | **77** | - | **30% → 60%+** |

## Test Categories

### No Docker Required (Run Anywhere)
These tests validate command parsing, configuration, and error handling:
- Backup: list, verify missing files, retention config (8 tests)
- Deploy: help, validation (3 tests)
- Storage: help, status, validation (6 tests)
- SSL: help, verify missing, validation (6 tests)
- Tenant: help, validation (3 tests)

**Subtotal**: ~26 tests run without Docker

### Docker Required (Integration Tests)
These tests require `nself start` to run full service integration:
- Backup: create, restore, pruning (8 tests)
- Deploy: SSH, health checks (7 tests)
- Storage: MinIO operations (9 tests)
- SSL: Let's Encrypt, DNS validation (10 tests)
- Tenant: provisioning, isolation (17 tests)

**Subtotal**: ~51 tests require Docker

## Running with Services

For full coverage, start services first:

```bash
# 1. Initialize and build
nself init
nself build

# 2. Start all services
nself start

# 3. Run full test suite
./run-critical-tests.sh

# 4. Stop services when done
nself stop
```

## CI/CD Integration

Tests automatically run in GitHub Actions:
- `.github/workflows/ci.yml` includes new test suites
- Docker-dependent tests are skipped in CI
- Configuration tests always run
- No hardcoded paths or secrets

## Test File Locations

```
src/tests/
├── backup_tests.bats           # Backup & restore tests
├── deploy_tests.bats           # Deployment workflow tests
├── storage_tests.bats          # Storage & MinIO tests
├── ssl_tests.bats              # SSL/TLS certificate tests
├── tenant_tests.bats           # Multi-tenancy tests
├── run-critical-tests.sh       # Test runner script
├── TEST-COVERAGE-IMPROVEMENTS.md  # Full documentation
└── README-CRITICAL-TESTS.md    # This file
```

## What Gets Tested

### Backup & Restore
- ✅ Backup creation (database, config, full)
- ✅ Backup listing and verification
- ✅ Restoration workflows
- ✅ Pruning strategies (age, count, size, GFS)
- ✅ Retention policies
- ✅ Cloud sync configuration

### Deploy
- ✅ Deployment workflows
- ✅ Environment validation
- ✅ SSH connectivity checks
- ✅ Health check validation
- ✅ Rollback procedures
- ✅ Security preflight checks

### Storage
- ✅ MinIO integration
- ✅ Bucket operations (create, list, delete)
- ✅ File upload/download
- ✅ Storage quota management
- ✅ Presigned URL generation
- ✅ Access control validation

### SSL/TLS
- ✅ Self-signed certificate generation
- ✅ Certificate verification
- ✅ Let's Encrypt integration
- ✅ Auto-renewal setup
- ✅ Multi-domain (SAN) certificates
- ✅ Wildcard certificates
- ✅ Expiry warnings

### Multi-Tenancy
- ✅ Tenant provisioning
- ✅ Tenant lifecycle (create, suspend, resume, delete)
- ✅ Data isolation validation
- ✅ Subdomain generation
- ✅ Member management
- ✅ Resource quotas
- ✅ Billing integration
- ✅ Tenant-specific backups

## Troubleshooting

### Tests Show "Skipped"
This is normal! Tests that require services (Docker, SSH, DNS) are intentionally skipped if those services aren't available.

To run all tests:
1. Start Docker
2. Run `nself start`
3. Re-run tests

### Tests Fail with "command not found"
Ensure `nself` is in your PATH:
```bash
export PATH="$(pwd)/bin:$PATH"
```

### Tests Fail on macOS with "illegal option"
The tests include cross-platform handling for macOS/Linux differences. If you see these errors, it's a bug - please report it.

## Contributing

When adding tests:
1. Use descriptive test names
2. Mark service-dependent tests with `skip`
3. Clean up in `teardown()`
4. Test both success and failure paths
5. Add to `run-critical-tests.sh` if new file

See `TEST-COVERAGE-IMPROVEMENTS.md` for detailed guidelines.

## Next Steps

After running these tests:
1. ✅ Review any failures
2. ✅ Check coverage report: `.claude/qa/TEST-COVERAGE-REPORT.md`
3. ✅ Run full test suite: `./run-all-tests.sh`
4. ✅ Add more tests for remaining gaps (Phase 2)

## Support

- **Full Documentation**: `TEST-COVERAGE-IMPROVEMENTS.md`
- **Test Framework**: `test_framework.sh`
- **Integration Tests**: `integration/` directory
- **Unit Tests**: `unit/` directory
