# Coverage Scripts

Comprehensive test coverage collection, reporting, and tracking for nself.

## Scripts Overview

### Core Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `collect-coverage.sh` | Run all tests with coverage tracking | `./collect-coverage.sh` |
| `generate-coverage-report.sh` | Generate text, HTML, JSON reports | `./generate-coverage-report.sh` |
| `verify-coverage.sh` | Enforce coverage requirements | `./verify-coverage.sh` |
| `track-coverage-history.sh` | Track coverage over time | `./track-coverage-history.sh track` |
| `coverage-diff.sh` | Show coverage changes (PRs) | `./coverage-diff.sh diff main HEAD` |

## Quick Start

### 1. Collect Coverage

```bash
./collect-coverage.sh
```

This runs all test suites with coverage tracking enabled.

**Output**:
- `coverage/data/unit/` - Unit test coverage
- `coverage/data/integration/` - Integration test coverage
- `coverage/data/security/` - Security test coverage
- `coverage/data/e2e/` - End-to-end test coverage

### 2. Generate Reports

```bash
./generate-coverage-report.sh
```

Creates multiple report formats.

**Output**:
- `coverage/reports/coverage.txt` - Terminal-friendly text report
- `coverage/reports/coverage.json` - Machine-readable JSON
- `coverage/reports/html/index.html` - Interactive HTML report
- `coverage/reports/badge.svg` - Coverage badge SVG

### 3. Verify Requirements

```bash
./verify-coverage.sh
```

Checks if coverage meets requirements (100% line coverage).

**Exit codes**:
- `0` - Coverage requirements met ✅
- `1` - Coverage below requirements ❌

### 4. Track History

```bash
./track-coverage-history.sh track
```

Adds current coverage to historical trend data.

**Commands**:
- `track` - Add current entry to history
- `report` - Generate trend report
- `show` - Show trend chart

### 5. Coverage Diff (PRs)

```bash
./coverage-diff.sh diff main HEAD
```

Shows coverage changes between branches.

**Commands**:
- `diff` - Show quick diff
- `full` - Show detailed diff with file breakdown
- `--quiet` - Return numeric diff only

## Full Workflow

Run complete coverage workflow:

```bash
# 1. Collect coverage from all test suites
./collect-coverage.sh

# 2. Generate all report formats
./generate-coverage-report.sh

# 3. Track in history
./track-coverage-history.sh track

# 4. Verify requirements
./verify-coverage.sh
```

One-liner:

```bash
./collect-coverage.sh && \
./generate-coverage-report.sh && \
./track-coverage-history.sh track && \
./verify-coverage.sh
```

## Environment Variables

### collect-coverage.sh

```bash
COVERAGE_ENABLED=true         # Enable coverage tracking
COVERAGE_FILE=path/to/file    # Coverage data output file
```

### verify-coverage.sh

```bash
REQUIRED_LINE_COVERAGE=100.0      # Required line coverage (default: 100%)
REQUIRED_BRANCH_COVERAGE=95.0     # Required branch coverage (default: 95%)
REQUIRED_FUNCTION_COVERAGE=100.0  # Required function coverage (default: 100%)
```

## Coverage Tools

### Required

- **Bash 3.2+** - Shell execution

### Optional (for full functionality)

- **kcov** - Bash code coverage collection
  ```bash
  # Ubuntu/Debian
  sudo apt-get install kcov

  # macOS
  brew install kcov
  ```

- **lcov** - Coverage data merging
  ```bash
  sudo apt-get install lcov
  ```

- **jq** - JSON processing
  ```bash
  sudo apt-get install jq
  ```

### Fallback

If coverage tools aren't available, scripts use manual instrumentation (slower but functional).

## Output Structure

```
coverage/
├── data/                          # Raw coverage data
│   ├── unit/                      # Unit test coverage
│   │   ├── index.html            # kcov HTML report
│   │   ├── coverage.json         # kcov JSON data
│   │   └── output.log            # Test output
│   ├── integration/              # Integration coverage
│   ├── security/                 # Security coverage
│   └── e2e/                      # E2E coverage
├── reports/                       # Generated reports
│   ├── coverage.txt              # Text summary
│   ├── coverage.json             # JSON data
│   ├── badge.svg                 # Coverage badge
│   ├── summary.txt               # Quick summary
│   ├── trend.txt                 # Trend report
│   └── html/                     # HTML reports
│       └── index.html            # Main HTML report
└── .coverage-history.json        # Historical data
```

## Report Formats

### Text Report

Terminal-friendly summary with progress bar and statistics.

```
========================================
nself Test Coverage Report
========================================

Overall Coverage:     100.0%  (target: 100%)
Line Coverage:        100.0%  (5,234 / 5,234 lines)
Branch Coverage:      98.5%   (1,234 / 1,253 branches)

Progress: [==================================================] 100.0%

Test Statistics:
  Total Tests:      700
  Passed:          700  (100.0%)
```

### JSON Report

Machine-readable data for automation.

```json
{
  "timestamp": "2026-01-31T21:45:00Z",
  "overall": {
    "line": 100.0,
    "branch": 98.5,
    "function": 100.0
  },
  "lines": {
    "total": 5234,
    "covered": 5234,
    "percentage": 100.0
  }
}
```

### HTML Report

Interactive browser-based report with:
- File browser
- Line-by-line highlighting
- Branch coverage visualization
- Uncovered code identification

### Badge

SVG badge for README:

![Coverage](../../../coverage/reports/badge.svg)

## CI/CD Integration

### GitHub Actions

Coverage runs automatically on push and PR.

Workflow: `/.github/workflows/coverage.yml`

### PR Comments

Automatic coverage comment on PRs:

```markdown
## 📊 Coverage Report

**Overall Coverage:** 100.0%
**Target:** 100%
**Gap:** 0.0%

🎉 **100% Coverage Achieved!**
```

### Coverage Enforcement

CI fails if:
- Line coverage < 100%
- Coverage decreases in PR

## Troubleshooting

### kcov not found

```bash
# Install kcov
sudo apt-get install kcov  # Ubuntu/Debian
brew install kcov          # macOS
```

Or use manual instrumentation (automatic fallback).

### No coverage data

```bash
# Ensure tests run successfully first
./src/tests/run-all-tests.sh

# Then collect coverage
./collect-coverage.sh
```

### Coverage below requirements

```bash
# View uncovered code
./generate-coverage-report.sh
open coverage/reports/html/index.html

# Add tests for red lines
# Re-run coverage
./collect-coverage.sh
```

## Examples

### Check coverage for specific feature

```bash
# Run only related tests
COVERAGE_ENABLED=true kcov coverage/data/feature \
  ./src/tests/unit/test-feature.sh

# Generate report
./generate-coverage-report.sh

# View results
cat coverage/reports/coverage.txt
```

### Pre-commit verification

```bash
# .git/hooks/pre-commit
#!/bin/bash

echo "Verifying coverage..."
./src/scripts/coverage/collect-coverage.sh
./src/scripts/coverage/verify-coverage.sh

if [ $? -ne 0 ]; then
    echo "❌ Coverage verification failed"
    exit 1
fi
```

### Coverage diff for PR

```bash
# Compare branches
./coverage-diff.sh diff origin/main HEAD

# Show detailed diff
./coverage-diff.sh full origin/main HEAD

# Get numeric diff only
DIFF=$(./coverage-diff.sh --quiet)
echo "Coverage changed by: $DIFF%"
```

## Advanced Usage

### Custom coverage threshold

```bash
# Require 95% instead of 100%
REQUIRED_LINE_COVERAGE=95.0 ./verify-coverage.sh
```

### Skip coverage verification

```bash
# Emergency bypass (not recommended!)
SKIP_COVERAGE_CHECK=true ./verify-coverage.sh
```

### Coverage for single file

```bash
# Test specific file with coverage
kcov --include-pattern=oauth.sh coverage/data/oauth \
  ./src/tests/unit/test-oauth.sh
```

## Documentation

- [Coverage Guide](../../../.wiki/development/COVERAGE-GUIDE.md) - Complete guide
- [Coverage Dashboard](../../../.wiki/development/COVERAGE-DASHBOARD.md) - Live dashboard
- [Testing Guidelines](../../../src/tests/TESTING_GUIDELINES.md) - Test writing guide

## Support

Issues with coverage scripts?

1. Check script output for error messages
2. Review [Coverage Guide](../../../.wiki/development/COVERAGE-GUIDE.md)
3. Verify tools are installed: `kcov`, `lcov`, `jq`
4. Run with debug: `bash -x ./collect-coverage.sh`

## Contributing

When modifying coverage scripts:

1. Test on Ubuntu and macOS
2. Ensure backward compatibility
3. Update this README
4. Update COVERAGE-GUIDE.md if needed

## License

Part of nself - MIT License
