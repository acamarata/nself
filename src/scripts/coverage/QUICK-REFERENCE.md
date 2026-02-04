# Coverage Quick Reference

Fast reference for common coverage tasks.

## One-Line Commands

```bash
# Full coverage workflow
./collect-coverage.sh && ./generate-coverage-report.sh && ./verify-coverage.sh

# Quick check
./collect-coverage.sh && cat coverage/reports/coverage.txt

# View HTML report
./generate-coverage-report.sh && open coverage/reports/html/index.html

# Check if coverage meets requirements
./verify-coverage.sh && echo "✅ Coverage OK" || echo "❌ Coverage FAIL"

# Show trend
./track-coverage-history.sh show

# PR diff
./coverage-diff.sh diff origin/main HEAD
```

## Common Tasks

### 1. Check Current Coverage

```bash
cat coverage/reports/coverage.txt
```

### 2. Find Uncovered Code

```bash
./generate-coverage-report.sh
open coverage/reports/html/index.html
# Look for red lines
```

### 3. Add Coverage Entry

```bash
./track-coverage-history.sh track
```

### 4. Verify Before Commit

```bash
./verify-coverage.sh
```

### 5. Compare Branches

```bash
./coverage-diff.sh diff main feature-branch
```

## Script Summary

| Need | Script | Args |
|------|--------|------|
| Run tests with coverage | `collect-coverage.sh` | - |
| Create reports | `generate-coverage-report.sh` | - |
| Check requirements | `verify-coverage.sh` | - |
| Track history | `track-coverage-history.sh` | `track|report|show` |
| Compare branches | `coverage-diff.sh` | `diff|full|--quiet` |

## Report Locations

```bash
# Text
cat coverage/reports/coverage.txt

# JSON
cat coverage/reports/coverage.json
jq '.overall' coverage/reports/coverage.json

# HTML
open coverage/reports/html/index.html

# Badge
open coverage/reports/badge.svg

# History
cat coverage/.coverage-history.json
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success / Coverage OK ✅ |
| 1 | Failure / Coverage below requirement ❌ |

## Environment Variables

```bash
# Coverage requirements
export REQUIRED_LINE_COVERAGE=100.0
export REQUIRED_BRANCH_COVERAGE=95.0
export REQUIRED_FUNCTION_COVERAGE=100.0

# Enable coverage
export COVERAGE_ENABLED=true
export COVERAGE_FILE=coverage/data/coverage.dat

# Skip verification (emergency)
export SKIP_COVERAGE_CHECK=true
```

## CI/CD

```bash
# GitHub Actions workflow
.github/workflows/coverage.yml

# Run manually
gh workflow run coverage.yml

# View latest run
gh run list --workflow=coverage.yml
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| kcov not found | `sudo apt-get install kcov` |
| No coverage data | Run `./collect-coverage.sh` first |
| Coverage < 100% | View HTML report, add tests for red lines |
| Test fails | Fix tests before collecting coverage |

## Pre-Commit Hook

```bash
# .git/hooks/pre-commit
#!/bin/bash
./src/scripts/coverage/verify-coverage.sh || exit 1
```

## Quick Checks

```bash
# Current coverage percentage
grep '"line"' coverage/reports/coverage.json | head -1 | grep -o '[0-9.]*'

# Test count
grep -c '"name"' coverage/reports/coverage.json

# Last coverage update
stat -f "%Sm" coverage/reports/coverage.txt  # macOS
stat -c "%y" coverage/reports/coverage.txt   # Linux
```

## Full Documentation

- **Guide**: `docs/development/COVERAGE-GUIDE.md`
- **Dashboard**: `docs/development/COVERAGE-DASHBOARD.md`
- **Scripts**: `src/scripts/coverage/README.md`
