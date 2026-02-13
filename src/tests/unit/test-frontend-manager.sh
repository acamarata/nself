#!/usr/bin/env bash
# test-frontend-manager.sh - Unit tests for frontend-manager.sh
# Part of nself v0.9.9 test suite

# Get script directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="$TEST_DIR/../../lib/utils"

# Source the frontend manager
source "$SRC_DIR/frontend-manager.sh"

# Test framework
TESTS_PASSED=0
TESTS_FAILED=0

assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="${3:-}"
  
  if [[ "$expected" == "$actual" ]]; then
    ((TESTS_PASSED++))
    printf "."
  else
    ((TESTS_FAILED++))
    printf "F"
    if [[ -n "$message" ]]; then
      printf "\n  FAIL: %s\n" "$message" >&2
    fi
    printf "\n  Expected: %s\n" "$expected" >&2
    printf "\n  Actual:   %s\n" "$actual" >&2
  fi
}

assert_true() {
  local message="${1:-}"
  ((TESTS_PASSED++))
  printf "."
}

assert_false() {
  local message="${1:-}"
  ((TESTS_FAILED++))
  printf "F"
  if [[ -n "$message" ]]; then
    printf "\n  FAIL: %s\n" "$message" >&2
  fi
}

# Test: detect_package_manager with pnpm
test_detect_package_manager_pnpm() {
  local test_dir=$(mktemp -d)
  touch "$test_dir/pnpm-lock.yaml"
  
  local result=$(detect_package_manager "$test_dir")
  assert_equals "pnpm" "$result" "detect_package_manager should return pnpm"
  
  rm -rf "$test_dir"
}

# Test: detect_package_manager with npm
test_detect_package_manager_npm() {
  local test_dir=$(mktemp -d)
  touch "$test_dir/package-lock.json"
  
  local result=$(detect_package_manager "$test_dir")
  assert_equals "npm" "$result" "detect_package_manager should return npm"
  
  rm -rf "$test_dir"
}

# Test: detect_package_manager with yarn
test_detect_package_manager_yarn() {
  local test_dir=$(mktemp -d)
  touch "$test_dir/yarn.lock"
  
  local result=$(detect_package_manager "$test_dir")
  assert_equals "yarn" "$result" "detect_package_manager should return yarn"
  
  rm -rf "$test_dir"
}

# Test: detect_package_manager with bun
test_detect_package_manager_bun() {
  local test_dir=$(mktemp -d)
  touch "$test_dir/bun.lockb"
  
  local result=$(detect_package_manager "$test_dir")
  assert_equals "bun" "$result" "detect_package_manager should return bun"
  
  rm -rf "$test_dir"
}

# Test: detect_package_manager priority (pnpm > yarn > bun > npm)
test_detect_package_manager_priority() {
  local test_dir=$(mktemp -d)
  
  # Create all lock files
  touch "$test_dir/pnpm-lock.yaml"
  touch "$test_dir/yarn.lock"
  touch "$test_dir/package-lock.json"
  touch "$test_dir/bun.lockb"
  
  local result=$(detect_package_manager "$test_dir")
  assert_equals "pnpm" "$result" "pnpm should have priority"
  
  # Remove pnpm, test yarn priority
  rm "$test_dir/pnpm-lock.yaml"
  result=$(detect_package_manager "$test_dir")
  assert_equals "yarn" "$result" "yarn should have priority after pnpm"
  
  # Remove yarn, test bun priority
  rm "$test_dir/yarn.lock"
  result=$(detect_package_manager "$test_dir")
  assert_equals "bun" "$result" "bun should have priority after yarn"
  
  # Remove bun, npm should be last
  rm "$test_dir/bun.lockb"
  result=$(detect_package_manager "$test_dir")
  assert_equals "npm" "$result" "npm should be last priority"
  
  rm -rf "$test_dir"
}

# Test: get_dev_command for each package manager
test_get_dev_command_pnpm() {
  local result=$(get_dev_command "pnpm")
  assert_equals "pnpm dev" "$result"
}

test_get_dev_command_npm() {
  local result=$(get_dev_command "npm")
  assert_equals "npm run dev" "$result"
}

test_get_dev_command_yarn() {
  local result=$(get_dev_command "yarn")
  assert_equals "yarn run dev" "$result"
}

test_get_dev_command_bun() {
  local result=$(get_dev_command "bun")
  assert_equals "bun dev" "$result"
}

# Test: has_dev_script with valid package.json
test_has_dev_script_valid() {
  local test_dir=$(mktemp -d)
  printf '{"scripts":{"dev":"next dev"}}' > "$test_dir/package.json"
  
  if has_dev_script "$test_dir"; then
    assert_true "has_dev_script should return true for valid dev script"
  else
    assert_false "has_dev_script failed for valid dev script"
  fi
  
  rm -rf "$test_dir"
}

# Test: has_dev_script with invalid package.json (no dev script)
test_has_dev_script_invalid() {
  local test_dir=$(mktemp -d)
  printf '{"scripts":{"build":"next build"}}' > "$test_dir/package.json"
  
  if has_dev_script "$test_dir"; then
    assert_false "has_dev_script should return false for missing dev script"
  else
    assert_true "has_dev_script correctly returned false"
  fi
  
  rm -rf "$test_dir"
}

# Test: has_dev_script with missing package.json
test_has_dev_script_missing() {
  local test_dir=$(mktemp -d)
  
  if has_dev_script "$test_dir"; then
    assert_false "has_dev_script should return false for missing package.json"
  else
    assert_true "has_dev_script correctly returned false for missing file"
  fi
  
  rm -rf "$test_dir"
}

# Test: get_frontend_port with explicit port
test_get_frontend_port_explicit() {
  local test_dir=$(mktemp -d)
  printf '{"scripts":{"dev":"next dev -p 3001"}}' > "$test_dir/package.json"
  
  local result=$(get_frontend_port "$test_dir")
  assert_equals "3001" "$result" "get_frontend_port should extract explicit port"
  
  rm -rf "$test_dir"
}

# Test: get_frontend_port with default
test_get_frontend_port_default() {
  local test_dir=$(mktemp -d)
  printf '{"scripts":{"dev":"next dev"}}' > "$test_dir/package.json"
  
  local result=$(get_frontend_port "$test_dir")
  assert_equals "3000" "$result" "get_frontend_port should default to 3000"
  
  rm -rf "$test_dir"
}

# Test: detect_framework for Next.js
test_detect_framework_nextjs() {
  local test_dir=$(mktemp -d)
  printf '{"dependencies":{"next":"13.0.0","react":"18.0.0"}}' > "$test_dir/package.json"
  
  local result=$(detect_framework "$test_dir")
  assert_equals "nextjs" "$result" "detect_framework should detect Next.js"
  
  rm -rf "$test_dir"
}

# Test: detect_framework for Vite React
test_detect_framework_vite_react() {
  local test_dir=$(mktemp -d)
  printf '{"devDependencies":{"@vitejs/plugin-react":"^4.0.0","vite":"^4.0.0"}}' > "$test_dir/package.json"
  
  local result=$(detect_framework "$test_dir")
  assert_equals "vite-react" "$result" "detect_framework should detect Vite React"
  
  rm -rf "$test_dir"
}

# Test: detect_framework for Create React App
test_detect_framework_cra() {
  local test_dir=$(mktemp -d)
  printf '{"dependencies":{"react-scripts":"5.0.0"}}' > "$test_dir/package.json"
  
  local result=$(detect_framework "$test_dir")
  assert_equals "cra" "$result" "detect_framework should detect CRA"
  
  rm -rf "$test_dir"
}

# Test: detect_frontend_apps
test_detect_frontend_apps() {
  local test_dir=$(mktemp -d)
  cd "$test_dir"
  
  # Create backend (should be excluded)
  mkdir backend
  printf '{"name":"backend"}' > backend/package.json
  
  # Create valid frontend apps
  mkdir app1
  printf '{"name":"app1"}' > app1/package.json
  
  mkdir app2
  printf '{"name":"app2"}' > app2/package.json
  
  # Create excluded directory
  mkdir node_modules
  printf '{"name":"node_modules"}' > node_modules/package.json
  
  # Detect apps
  local apps=()
  while IFS= read -r app; do
    apps+=("$(basename "$app")")
  done < <(detect_frontend_apps)
  
  # Should find app1 and app2, but not backend or node_modules
  local found_app1=false
  local found_app2=false
  local found_backend=false
  local found_node_modules=false
  
  for app in "${apps[@]}"; do
    if [[ "$app" == "app1" ]]; then
      found_app1=true
    elif [[ "$app" == "app2" ]]; then
      found_app2=true
    elif [[ "$app" == "backend" ]]; then
      found_backend=true
    elif [[ "$app" == "node_modules" ]]; then
      found_node_modules=true
    fi
  done
  
  if [[ "$found_app1" == "true" ]] && [[ "$found_app2" == "true" ]] && \
     [[ "$found_backend" == "false" ]] && [[ "$found_node_modules" == "false" ]]; then
    assert_true "detect_frontend_apps correctly detected apps"
  else
    assert_false "detect_frontend_apps failed to correctly detect apps"
  fi
  
  cd - >/dev/null
  rm -rf "$test_dir"
}

# Run all tests
printf "Running frontend-manager.sh unit tests...\n\n"

test_detect_package_manager_pnpm
test_detect_package_manager_npm
test_detect_package_manager_yarn
test_detect_package_manager_bun
test_detect_package_manager_priority
test_get_dev_command_pnpm
test_get_dev_command_npm
test_get_dev_command_yarn
test_get_dev_command_bun
test_has_dev_script_valid
test_has_dev_script_invalid
test_has_dev_script_missing
test_get_frontend_port_explicit
test_get_frontend_port_default
test_detect_framework_nextjs
test_detect_framework_vite_react
test_detect_framework_cra
test_detect_frontend_apps

# Summary
printf "\n\n"
printf "=================================================================\n"
printf "Test Results:\n"
printf "=================================================================\n"
printf "Passed: %d\n" "$TESTS_PASSED"
printf "Failed: %d\n" "$TESTS_FAILED"
printf "Total:  %d\n" "$((TESTS_PASSED + TESTS_FAILED))"
printf "=================================================================\n"

if [[ $TESTS_FAILED -eq 0 ]]; then
  printf "\n✓ All tests passed!\n\n"
  exit 0
else
  printf "\n✗ Some tests failed.\n\n"
  exit 1
fi
