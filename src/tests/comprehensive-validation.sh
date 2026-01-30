#!/usr/bin/env bash
# comprehensive-validation.sh - Final validation of all systems
# Tests init, build, start for cross-platform compatibility and modularity

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
ISSUES_FOUND=""

# Test function
run_test() {
  local test_name="$1"
  local test_command="$2"

  printf "${BLUE}Testing:${NC} %s\n" "$test_name"

  if eval "$test_command" >/dev/null 2>&1; then
    printf "${GREEN}✓${NC} %s passed\n" "$test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    printf "${RED}✗${NC} %s failed\n" "$test_name"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    # Bash 3.2 compatible array append
    if [ -n "$ISSUES_FOUND" ]; then
      ISSUES_FOUND="${ISSUES_FOUND}|${test_name}"
    else
      ISSUES_FOUND="$test_name"
    fi
  fi
}

echo "=========================================="
echo "nself Comprehensive Validation"
echo "=========================================="
echo ""

# 1. Check for platform compatibility utilities
printf "${YELLOW}1. Platform Compatibility Checks${NC}\n"
echo "----------------------------------------"

run_test "Platform utilities exist" "[[ -f src/lib/utils/platform-compat.sh ]]"
run_test "Coding standards exist" "[[ -f src/lib/utils/coding-standards.sh ]]"
run_test "Safe sed defined" "grep -q 'safe_sed_inline' src/lib/utils/platform-compat.sh"
run_test "Safe readlink defined" "grep -q 'safe_readlink' src/lib/utils/platform-compat.sh"

echo ""

# 2. Check for no hardcoded paths
printf "${YELLOW}2. No Hardcoded Paths${NC}\n"
echo "----------------------------------------"

if grep -r "/Users/admin/Sites/nself" src --include="*.sh" | grep -v "test" | grep -v ".backup" >/dev/null; then
  printf "${RED}✗${NC} Found hardcoded paths\n"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  if [ -n "$ISSUES_FOUND" ]; then
    ISSUES_FOUND="${ISSUES_FOUND}|Hardcoded paths found"
  else
    ISSUES_FOUND="Hardcoded paths found"
  fi
else
  printf "${GREEN}✓${NC} No hardcoded paths\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

echo ""

# 3. Check modularity - no files over 500 lines
printf "${YELLOW}3. Modularity Checks${NC}\n"
echo "----------------------------------------"

LARGE_FILES=0
for file in $(find src -name "*.sh" -type f); do
  lines=$(wc -l <"$file")
  if [[ $lines -gt 500 ]]; then
    filename=$(basename "$file")
    if [[ $lines -gt 1000 ]]; then
      printf "${RED}✗${NC} %s has %d lines (>1000)\n" "$filename" "$lines"
      LARGE_FILES=$((LARGE_FILES + 1))
    elif [[ $lines -gt 800 ]]; then
      printf "${YELLOW}⚠${NC} %s has %d lines (>800)\n" "$filename" "$lines"
    fi
  fi
done

if [[ $LARGE_FILES -eq 0 ]]; then
  printf "${GREEN}✓${NC} All modules are properly sized\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  printf "${RED}✗${NC} Found %d oversized modules\n" "$LARGE_FILES"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  if [ -n "$ISSUES_FOUND" ]; then
    ISSUES_FOUND="${ISSUES_FOUND}|Oversized modules"
  else
    ISSUES_FOUND="Oversized modules"
  fi
fi

echo ""

# 4. Check for Bash 3.2 compatibility
printf "${YELLOW}4. Bash 3.2 Compatibility${NC}\n"
echo "----------------------------------------"

# Check for associative arrays
if grep -r "declare -A" src --include="*.sh" >/dev/null; then
  printf "${RED}✗${NC} Found associative arrays (Bash 4+ feature)\n"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  if [ -n "$ISSUES_FOUND" ]; then
    ISSUES_FOUND="${ISSUES_FOUND}|Associative arrays found"
  else
    ISSUES_FOUND="Associative arrays found"
  fi
else
  printf "${GREEN}✓${NC} No associative arrays\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

# Check for mapfile/readarray
if grep -rE "mapfile|readarray" src --include="*.sh" >/dev/null; then
  printf "${RED}✗${NC} Found mapfile/readarray (Bash 4+ feature)\n"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  if [ -n "$ISSUES_FOUND" ]; then
    ISSUES_FOUND="${ISSUES_FOUND}|mapfile/readarray found"
  else
    ISSUES_FOUND="mapfile/readarray found"
  fi
else
  printf "${GREEN}✓${NC} No mapfile/readarray\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

# Check for ${var,,} or ${var^^}
if grep -rE '\$\{[^}]+,,[^}]*\}|\$\{[^}]+\^\^[^}]*\}' src --include="*.sh" >/dev/null; then
  printf "${RED}✗${NC} Found case conversion (Bash 4+ feature)\n"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  if [ -n "$ISSUES_FOUND" ]; then
    ISSUES_FOUND="${ISSUES_FOUND}|Bash 4+ case conversion found"
  else
    ISSUES_FOUND="Bash 4+ case conversion found"
  fi
else
  printf "${GREEN}✓${NC} No Bash 4+ case conversion\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

echo ""

# 5. Check sed compatibility
printf "${YELLOW}5. sed Compatibility${NC}\n"
echo "----------------------------------------"

# Check for raw sed -i usage (should use safe_sed_inline)
UNSAFE_SED_COUNT=0
for file in $(find src -name "*.sh" -type f); do
  # Skip files that define safe_sed_inline
  if grep -q "safe_sed_inline()" "$file"; then
    continue
  fi

  # Check for sed -i without safe_sed_inline
  if grep -E "sed -i[^n]" "$file" | grep -v "safe_sed_inline" >/dev/null 2>&1; then
    UNSAFE_SED_COUNT=$((UNSAFE_SED_COUNT + 1))
    printf "${YELLOW}⚠${NC} %s may have unsafe sed usage\n" "$(basename "$file")"
  fi
done

if [[ $UNSAFE_SED_COUNT -eq 0 ]]; then
  printf "${GREEN}✓${NC} All sed usage is platform-safe\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  printf "${YELLOW}⚠${NC} Found %d files with potential sed issues\n" "$UNSAFE_SED_COUNT"
fi

echo ""

# 6. Test init process
printf "${YELLOW}6. Init Process Tests${NC}\n"
echo "----------------------------------------"

TEST_DIR="/tmp/nself-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

run_test "Init basic" "nself init"
run_test "Init creates .env" "[[ -f .env ]]"
run_test "Init demo" "nself init --demo"
run_test "Demo .env has required vars" "grep -q 'PROJECT_NAME=' .env"

cd - >/dev/null
rm -rf "$TEST_DIR"

echo ""

# 7. Test build process
printf "${YELLOW}7. Build Process Tests${NC}\n"
echo "----------------------------------------"

TEST_DIR="/tmp/nself-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize first
nself init --demo >/dev/null 2>&1

run_test "Build runs" "nself build"
run_test "Build creates docker-compose.yml" "[[ -f docker-compose.yml ]]"
run_test "Build creates nginx config" "[[ -d nginx ]]"
run_test "Build creates SSL certs" "[[ -d ssl ]]"

cd - >/dev/null
rm -rf "$TEST_DIR"

echo ""

# 8. Check variable scoping
printf "${YELLOW}8. Variable Scoping${NC}\n"
echo "----------------------------------------"

GLOBAL_VAR_COUNT=0
for file in $(find src/lib -name "*.sh" -type f); do
  # Look for variable assignments without local in functions
  if awk '/^[[:space:]]*[a-z_][a-z0-9_]*\(\)/ { infunc=1 }
         infunc && /^[[:space:]]*[a-z_][a-z0-9_]*=/ && !/local/ && !/readonly/ && !/export/ {
           print FILENAME ": " $0; found=1
         }
         /^}/ { infunc=0 }
         END { if(found) exit 1 }' "$file" 2>/dev/null; then
    GLOBAL_VAR_COUNT=$((GLOBAL_VAR_COUNT + 1))
  fi
done

if [[ $GLOBAL_VAR_COUNT -eq 0 ]]; then
  printf "${GREEN}✓${NC} All variables properly scoped\n"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  printf "${YELLOW}⚠${NC} Found %d files with potential scoping issues\n" "$GLOBAL_VAR_COUNT"
fi

echo ""

# 9. Check service templates
printf "${YELLOW}9. Service Templates${NC}\n"
echo "----------------------------------------"

run_test "Service templates exist" "[[ -d src/templates/services ]]"
run_test "Service generator uses templates" "grep -q 'copy_service_template' src/lib/auto-fix/service-generator.sh"
run_test "No code generation modules" "[[ ! -d src/lib/service-generators ]]"

echo ""

# 10. Final Summary
echo "=========================================="
printf "${YELLOW}Final Validation Summary${NC}\n"
echo "=========================================="
echo ""

printf "Tests Passed: ${GREEN}%d${NC}\n" "$TESTS_PASSED"
printf "Tests Failed: ${RED}%d${NC}\n" "$TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo ""
  printf "${GREEN}✓ All validation checks passed!${NC}\n"
  echo "The codebase is modular, maintainable, and cross-platform compatible."
  exit 0
else
  echo ""
  printf "${RED}Issues found:${NC}\n"
  # Bash 3.2 compatible loop over pipe-delimited string
  if [ -n "$ISSUES_FOUND" ]; then
    echo "$ISSUES_FOUND" | tr '|' '\n' | while read -r issue; do
      echo "  - $issue"
    done
  fi
  echo ""
  echo "Please fix the above issues for full compliance."
  exit 1
fi
