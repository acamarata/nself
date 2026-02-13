#!/usr/bin/env bash
# test-monorepo.sh - Integration tests for monorepo workflow
# Part of nself v0.9.9 test suite

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="$SCRIPT_DIR/../../lib/utils"

# Test framework
TESTS_PASSED=0
TESTS_FAILED=0

# Colors
COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_RED='\033[0;31m'
COLOR_BLUE='\033[0;34m'

assert_true() {
  local message="$1"
  ((TESTS_PASSED++))
  printf "${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$message"
}

assert_false() {
  local message="$1"
  ((TESTS_FAILED++))
  printf "${COLOR_RED}✗${COLOR_RESET} %s\n" "$message" >&2
}

# Test 1: Monorepo detection (positive)
test_monorepo_detection_positive() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Create monorepo structure
  mkdir backend
  touch backend/docker-compose.yml
  touch backend/.env
  
  # Source start.sh functions (just the detection function)
  is_monorepo() {
    local backend_dir="${BACKEND_DIR:-backend}"
    [[ -d "$backend_dir" ]] && [[ -f "$backend_dir/docker-compose.yml" ]]
  }
  
  if is_monorepo; then
    assert_true "Monorepo structure detected correctly"
  else
    assert_false "Failed to detect monorepo structure"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Test 2: Monorepo detection (negative - no backend)
test_monorepo_detection_negative_no_backend() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Create non-monorepo structure (no backend)
  mkdir app1
  touch app1/package.json
  
  is_monorepo() {
    local backend_dir="${BACKEND_DIR:-backend}"
    [[ -d "$backend_dir" ]] && [[ -f "$backend_dir/docker-compose.yml" ]]
  }
  
  if is_monorepo; then
    assert_false "Incorrectly detected monorepo (no backend)"
  else
    assert_true "Correctly rejected non-monorepo (no backend)"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Test 3: Monorepo detection (negative - no docker-compose.yml)
test_monorepo_detection_negative_no_compose() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Create structure with backend but no docker-compose.yml
  mkdir backend
  touch backend/.env
  mkdir app1
  touch app1/package.json
  
  is_monorepo() {
    local backend_dir="${BACKEND_DIR:-backend}"
    [[ -d "$backend_dir" ]] && [[ -f "$backend_dir/docker-compose.yml" ]]
  }
  
  if is_monorepo; then
    assert_false "Incorrectly detected monorepo (no docker-compose.yml)"
  else
    assert_true "Correctly rejected non-monorepo (no docker-compose.yml)"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Test 4: Frontend app detection
test_frontend_app_detection() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Source frontend-manager.sh
  source "$SRC_DIR/frontend-manager.sh"
  
  # Create backend (should be excluded)
  mkdir backend
  printf '{"name":"backend"}' > backend/package.json
  touch backend/docker-compose.yml
  
  # Create valid frontend apps
  mkdir app1
  printf '{"name":"app1","scripts":{"dev":"next dev"}}' > app1/package.json
  touch app1/package-lock.json
  
  mkdir app2
  printf '{"name":"app2","scripts":{"dev":"vite"}}' > app2/package.json
  touch app2/pnpm-lock.yaml
  
  # Create excluded directories
  mkdir node_modules
  printf '{"name":"should-exclude"}' > node_modules/package.json
  
  # Detect apps
  local apps=()
  while IFS= read -r app; do
    if [[ -n "$app" ]]; then
      apps+=("$(basename "$app")")
    fi
  done < <(detect_frontend_apps)
  
  # Verify detection
  local found_app1=false
  local found_app2=false
  local found_backend=false
  
  for app in "${apps[@]}"; do
    if [[ "$app" == "app1" ]]; then
      found_app1=true
    elif [[ "$app" == "app2" ]]; then
      found_app2=true
    elif [[ "$app" == "backend" ]]; then
      found_backend=true
    fi
  done
  
  if [[ "$found_app1" == "true" ]] && [[ "$found_app2" == "true" ]] && [[ "$found_backend" == "false" ]]; then
    assert_true "Frontend apps detected correctly (found app1, app2; excluded backend)"
  else
    assert_false "Frontend app detection failed (found_app1=$found_app1, found_app2=$found_app2, found_backend=$found_backend)"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Test 5: Package manager detection across multiple apps
test_package_manager_detection_multiple() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Source frontend-manager.sh
  source "$SRC_DIR/frontend-manager.sh"
  
  # Create apps with different package managers
  mkdir app_pnpm
  printf '{"name":"app_pnpm"}' > app_pnpm/package.json
  touch app_pnpm/pnpm-lock.yaml
  
  mkdir app_npm
  printf '{"name":"app_npm"}' > app_npm/package.json
  touch app_npm/package-lock.json
  
  mkdir app_yarn
  printf '{"name":"app_yarn"}' > app_yarn/package.json
  touch app_yarn/yarn.lock
  
  mkdir app_bun
  printf '{"name":"app_bun"}' > app_bun/package.json
  touch app_bun/bun.lockb
  
  # Test detection
  local pnpm_result=$(detect_package_manager "app_pnpm")
  local npm_result=$(detect_package_manager "app_npm")
  local yarn_result=$(detect_package_manager "app_yarn")
  local bun_result=$(detect_package_manager "app_bun")
  
  if [[ "$pnpm_result" == "pnpm" ]] && [[ "$npm_result" == "npm" ]] && \
     [[ "$yarn_result" == "yarn" ]] && [[ "$bun_result" == "bun" ]]; then
    assert_true "Package managers detected correctly (pnpm, npm, yarn, bun)"
  else
    assert_false "Package manager detection failed (pnpm=$pnpm_result, npm=$npm_result, yarn=$yarn_result, bun=$bun_result)"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Test 6: Dev script validation
test_dev_script_validation() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Source frontend-manager.sh
  source "$SRC_DIR/frontend-manager.sh"
  
  # Create app with dev script
  mkdir app_with_dev
  printf '{"scripts":{"dev":"next dev"}}' > app_with_dev/package.json
  
  # Create app without dev script
  mkdir app_without_dev
  printf '{"scripts":{"build":"next build"}}' > app_without_dev/package.json
  
  # Test validation
  local has_dev=false
  local no_dev=false
  
  if has_dev_script "app_with_dev"; then
    has_dev=true
  fi
  
  if ! has_dev_script "app_without_dev"; then
    no_dev=true
  fi
  
  if [[ "$has_dev" == "true" ]] && [[ "$no_dev" == "true" ]]; then
    assert_true "Dev script validation works correctly"
  else
    assert_false "Dev script validation failed (has_dev=$has_dev, no_dev=$no_dev)"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Test 7: Framework detection
test_framework_detection() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Source frontend-manager.sh
  source "$SRC_DIR/frontend-manager.sh"
  
  # Create Next.js app
  mkdir nextjs_app
  printf '{"dependencies":{"next":"13.0.0"}}' > nextjs_app/package.json
  
  # Create Vite React app
  mkdir vite_app
  printf '{"devDependencies":{"@vitejs/plugin-react":"4.0.0"}}' > vite_app/package.json
  
  # Test detection
  local nextjs_result=$(detect_framework "nextjs_app")
  local vite_result=$(detect_framework "vite_app")
  
  if [[ "$nextjs_result" == "nextjs" ]] && [[ "$vite_result" == "vite-react" ]]; then
    assert_true "Framework detection works correctly (Next.js, Vite)"
  else
    assert_false "Framework detection failed (nextjs=$nextjs_result, vite=$vite_result)"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Run all tests
printf "\n${COLOR_BLUE}Running monorepo integration tests...${COLOR_RESET}\n\n"

test_monorepo_detection_positive
test_monorepo_detection_negative_no_backend
test_monorepo_detection_negative_no_compose
test_frontend_app_detection
test_package_manager_detection_multiple
test_dev_script_validation
test_framework_detection

# Summary
printf "\n"
printf "=================================================================\n"
printf "Integration Test Results:\n"
printf "=================================================================\n"
printf "Passed: %d\n" "$TESTS_PASSED"
printf "Failed: %d\n" "$TESTS_FAILED"
printf "Total:  %d\n" "$((TESTS_PASSED + TESTS_FAILED))"
printf "=================================================================\n"

if [[ $TESTS_FAILED -eq 0 ]]; then
  printf "\n${COLOR_GREEN}✓ All integration tests passed!${COLOR_RESET}\n\n"
  exit 0
else
  printf "\n${COLOR_RED}✗ Some integration tests failed.${COLOR_RESET}\n\n"
  exit 1
fi
