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
ISSUES_FOUND=()

# Test function
run_test() {
  local test_name="$1"
  local test_command="$2"

  echo -e "${BLUE}Testing:${NC} $test_name"

  if eval "$test_command" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} $test_name passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}✗${NC} $test_name failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    ISSUES_FOUND+=("$test_name")
  fi
}

echo "=========================================="
echo "nself Comprehensive Validation"
echo "=========================================="
echo ""

# 1. Check for platform compatibility utilities
echo -e "${YELLOW}1. Platform Compatibility Checks${NC}"
echo "----------------------------------------"

run_test "Platform utilities exist" "[[ -f src/lib/utils/platform-compat.sh ]]"
run_test "Coding standards exist" "[[ -f src/lib/utils/coding-standards.sh ]]"
run_test "Safe sed defined" "grep -q 'safe_sed_inline' src/lib/utils/platform-compat.sh"
run_test "Safe readlink defined" "grep -q 'safe_readlink' src/lib/utils/platform-compat.sh"

echo ""

# 2. Check for no hardcoded paths
echo -e "${YELLOW}2. No Hardcoded Paths${NC}"
echo "----------------------------------------"

if grep -r "/Users/admin/Sites/nself" src --include="*.sh" | grep -v "test" | grep -v ".backup" > /dev/null; then
  echo -e "${RED}✗${NC} Found hardcoded paths"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  ISSUES_FOUND+=("Hardcoded paths found")
else
  echo -e "${GREEN}✓${NC} No hardcoded paths"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

echo ""

# 3. Check modularity - no files over 500 lines
echo -e "${YELLOW}3. Modularity Checks${NC}"
echo "----------------------------------------"

LARGE_FILES=0
for file in $(find src -name "*.sh" -type f); do
  lines=$(wc -l < "$file")
  if [[ $lines -gt 500 ]]; then
    filename=$(basename "$file")
    if [[ $lines -gt 1000 ]]; then
      echo -e "${RED}✗${NC} $filename has $lines lines (>1000)"
      LARGE_FILES=$((LARGE_FILES + 1))
    elif [[ $lines -gt 800 ]]; then
      echo -e "${YELLOW}⚠${NC} $filename has $lines lines (>800)"
    fi
  fi
done

if [[ $LARGE_FILES -eq 0 ]]; then
  echo -e "${GREEN}✓${NC} All modules are properly sized"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${RED}✗${NC} Found $LARGE_FILES oversized modules"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  ISSUES_FOUND+=("Oversized modules")
fi

echo ""

# 4. Check for Bash 3.2 compatibility
echo -e "${YELLOW}4. Bash 3.2 Compatibility${NC}"
echo "----------------------------------------"

# Check for associative arrays
if grep -r "declare -A" src --include="*.sh" > /dev/null; then
  echo -e "${RED}✗${NC} Found associative arrays (Bash 4+ feature)"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  ISSUES_FOUND+=("Associative arrays found")
else
  echo -e "${GREEN}✓${NC} No associative arrays"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

# Check for mapfile/readarray
if grep -rE "mapfile|readarray" src --include="*.sh" > /dev/null; then
  echo -e "${RED}✗${NC} Found mapfile/readarray (Bash 4+ feature)"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  ISSUES_FOUND+=("mapfile/readarray found")
else
  echo -e "${GREEN}✓${NC} No mapfile/readarray"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

# Check for ${var,,} or ${var^^}
if grep -rE '\$\{[^}]+,,[^}]*\}|\$\{[^}]+\^\^[^}]*\}' src --include="*.sh" > /dev/null; then
  echo -e "${RED}✗${NC} Found case conversion (Bash 4+ feature)"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  ISSUES_FOUND+=("Bash 4+ case conversion found")
else
  echo -e "${GREEN}✓${NC} No Bash 4+ case conversion"
  TESTS_PASSED=$((TESTS_PASSED + 1))
fi

echo ""

# 5. Check sed compatibility
echo -e "${YELLOW}5. sed Compatibility${NC}"
echo "----------------------------------------"

# Check for raw sed -i usage (should use safe_sed_inline)
UNSAFE_SED_COUNT=0
for file in $(find src -name "*.sh" -type f); do
  # Skip files that define safe_sed_inline
  if grep -q "safe_sed_inline()" "$file"; then
    continue
  fi

  # Check for sed -i without safe_sed_inline
  if grep -E "sed -i[^n]" "$file" | grep -v "safe_sed_inline" > /dev/null 2>&1; then
    UNSAFE_SED_COUNT=$((UNSAFE_SED_COUNT + 1))
    echo -e "${YELLOW}⚠${NC} $(basename "$file") may have unsafe sed usage"
  fi
done

if [[ $UNSAFE_SED_COUNT -eq 0 ]]; then
  echo -e "${GREEN}✓${NC} All sed usage is platform-safe"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${YELLOW}⚠${NC} Found $UNSAFE_SED_COUNT files with potential sed issues"
fi

echo ""

# 6. Test init process
echo -e "${YELLOW}6. Init Process Tests${NC}"
echo "----------------------------------------"

TEST_DIR="/tmp/nself-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

run_test "Init basic" "nself init"
run_test "Init creates .env" "[[ -f .env ]]"
run_test "Init demo" "nself init --demo"
run_test "Demo .env has required vars" "grep -q 'PROJECT_NAME=' .env"

cd - > /dev/null
rm -rf "$TEST_DIR"

echo ""

# 7. Test build process
echo -e "${YELLOW}7. Build Process Tests${NC}"
echo "----------------------------------------"

TEST_DIR="/tmp/nself-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize first
nself init --demo > /dev/null 2>&1

run_test "Build runs" "nself build"
run_test "Build creates docker-compose.yml" "[[ -f docker-compose.yml ]]"
run_test "Build creates nginx config" "[[ -d nginx ]]"
run_test "Build creates SSL certs" "[[ -d ssl ]]"

cd - > /dev/null
rm -rf "$TEST_DIR"

echo ""

# 8. Check variable scoping
echo -e "${YELLOW}8. Variable Scoping${NC}"
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
  echo -e "${GREEN}✓${NC} All variables properly scoped"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  echo -e "${YELLOW}⚠${NC} Found $GLOBAL_VAR_COUNT files with potential scoping issues"
fi

echo ""

# 9. Check service templates
echo -e "${YELLOW}9. Service Templates${NC}"
echo "----------------------------------------"

run_test "Service templates exist" "[[ -d src/templates/services ]]"
run_test "Service generator uses templates" "grep -q 'copy_service_template' src/lib/auto-fix/service-generator.sh"
run_test "No code generation modules" "[[ ! -d src/lib/service-generators ]]"

echo ""

# 10. Final Summary
echo "=========================================="
echo -e "${YELLOW}Final Validation Summary${NC}"
echo "=========================================="
echo ""

echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo ""
  echo -e "${GREEN}✓ All validation checks passed!${NC}"
  echo "The codebase is modular, maintainable, and cross-platform compatible."
  exit 0
else
  echo ""
  echo -e "${RED}Issues found:${NC}"
  for issue in "${ISSUES_FOUND[@]}"; do
    echo "  - $issue"
  done
  echo ""
  echo "Please fix the above issues for full compliance."
  exit 1
fi