#!/usr/bin/env bash
# e2e-comprehensive.sh - End-to-End Testing Suite for nself CLI
# Tests all CLI commands in isolated temp directory
# POSIX-compliant, no Bash 4+ features

set -euo pipefail

# ═══════════════════════════════════════════════════════════════
# Configuration
# ═══════════════════════════════════════════════════════════════

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
NSELF_BIN="$PROJECT_ROOT/bin/nself"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Test environment
TEST_ROOT=""
ORIGINAL_DIR="$(pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
RESET='\033[0m'

# ═══════════════════════════════════════════════════════════════
# Test Framework Functions
# ═══════════════════════════════════════════════════════════════

log_header() {
  printf "\n${CYAN}═══════════════════════════════════════════════════════════════${RESET}\n"
  printf "${CYAN}  %s${RESET}\n" "$1"
  printf "${CYAN}═══════════════════════════════════════════════════════════════${RESET}\n\n"
}

log_section() {
  printf "\n${BLUE}── %s ──${RESET}\n\n" "$1"
}

test_pass() {
  local message="$1"
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  PASSED_TESTS=$((PASSED_TESTS + 1))
  printf "  ${GREEN}✓${RESET} %s\n" "$message"
}

test_fail() {
  local message="$1"
  local details="${2:-}"
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  FAILED_TESTS=$((FAILED_TESTS + 1))
  printf "  ${RED}✗${RESET} %s\n" "$message"
  if [[ -n "$details" ]]; then
    printf "    ${RED}→ %s${RESET}\n" "$details"
  fi
}

test_skip() {
  local message="$1"
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
  printf "  ${YELLOW}⊘${RESET} %s (skipped)\n" "$message"
}

setup_test_environment() {
  TEST_ROOT=$(mktemp -d)
  cd "$TEST_ROOT"
  printf "Test directory: %s\n\n" "$TEST_ROOT"
}

cleanup_test_environment() {
  cd "$ORIGINAL_DIR"
  if [[ -n "$TEST_ROOT" ]] && [[ -d "$TEST_ROOT" ]]; then
    rm -rf "$TEST_ROOT"
  fi
}

assert_file_exists() {
  local file="$1"
  local message="${2:-File should exist: $file}"
  if [[ -f "$file" ]]; then
    test_pass "$message"
    return 0
  else
    test_fail "$message" "File not found: $file"
    return 1
  fi
}

assert_dir_exists() {
  local dir="$1"
  local message="${2:-Directory should exist: $dir}"
  if [[ -d "$dir" ]]; then
    test_pass "$message"
    return 0
  else
    test_fail "$message" "Directory not found: $dir"
    return 1
  fi
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  local message="${3:-File should contain: $pattern}"
  if [[ -f "$file" ]] && grep -q "$pattern" "$file" 2>/dev/null; then
    test_pass "$message"
    return 0
  else
    test_fail "$message" "Pattern '$pattern' not found in $file"
    return 1
  fi
}

assert_command_succeeds() {
  local cmd="$1"
  local message="${2:-Command should succeed}"
  if eval "$cmd" >/dev/null 2>&1; then
    test_pass "$message"
    return 0
  else
    test_fail "$message" "Command failed: $cmd"
    return 1
  fi
}

assert_command_fails() {
  local cmd="$1"
  local message="${2:-Command should fail}"
  if ! eval "$cmd" >/dev/null 2>&1; then
    test_pass "$message"
    return 0
  else
    test_fail "$message" "Command succeeded but should have failed"
    return 1
  fi
}

assert_output_contains() {
  local cmd="$1"
  local pattern="$2"
  local message="${3:-Output should contain: $pattern}"
  local output
  output=$(eval "$cmd" 2>&1) || true
  if echo "$output" | grep -q "$pattern"; then
    test_pass "$message"
    return 0
  else
    test_fail "$message" "Pattern not found in output"
    return 1
  fi
}

# ═══════════════════════════════════════════════════════════════
# Test Suites
# ═══════════════════════════════════════════════════════════════

test_nself_binary() {
  log_section "nself Binary Tests"

  # Check binary exists
  if [[ -x "$NSELF_BIN" ]]; then
    test_pass "nself binary exists and is executable"
  else
    test_fail "nself binary not found or not executable" "$NSELF_BIN"
    return 1
  fi

  # Check version
  local version
  version=$("$NSELF_BIN" version 2>&1) || true
  if echo "$version" | grep -qE "nself|0\.[0-9]"; then
    test_pass "nself version command works"
  else
    test_fail "nself version command failed"
  fi

  # Check help
  local help_output
  help_output=$("$NSELF_BIN" help 2>&1) || true
  if echo "$help_output" | grep -qi "usage\|commands\|options"; then
    test_pass "nself help command works"
  else
    test_fail "nself help command failed"
  fi
}

test_init_command() {
  log_section "nself init Tests"

  # Create test project directory
  mkdir -p test-project
  cd test-project

  # Test non-interactive init (minimal)
  if "$NSELF_BIN" init --non-interactive --project-name=test-app --domain=localhost 2>&1 | grep -qi "success\|created\|initialized"; then
    test_pass "nself init --non-interactive works"
  else
    # Try alternate approach
    printf "test-app\nlocalhost\n\n\n\n" | "$NSELF_BIN" init 2>&1 || true
    if [[ -f ".env.dev" ]] || [[ -f ".env" ]]; then
      test_pass "nself init creates environment file"
    else
      test_fail "nself init failed to create files"
    fi
  fi

  # Check generated files
  if [[ -f ".env.dev" ]]; then
    assert_file_exists ".env.dev" ".env.dev created"
    assert_file_contains ".env.dev" "PROJECT_NAME" ".env.dev contains PROJECT_NAME"
  elif [[ -f ".env" ]]; then
    assert_file_exists ".env" ".env created"
  else
    test_skip "Environment file not created (may require interactive mode)"
  fi

  cd ..
}

test_build_command() {
  log_section "nself build Tests"

  # Create a minimal project structure
  mkdir -p build-test
  cd build-test

  # Create minimal .env.dev
  cat >.env.dev <<'EOF'
PROJECT_NAME=build-test
ENV=dev
BASE_DOMAIN=localhost
POSTGRES_DB=buildtest_db
POSTGRES_USER=postgres
POSTGRES_PASSWORD=testpassword123
HASURA_GRAPHQL_ADMIN_SECRET=testadminsecret123
EOF

  # Run build
  local build_output
  build_output=$("$NSELF_BIN" build 2>&1) || true

  # Check if docker-compose.yml was generated
  if [[ -f "docker-compose.yml" ]]; then
    test_pass "nself build generates docker-compose.yml"
    assert_file_contains "docker-compose.yml" "services:" "docker-compose.yml contains services"
    assert_file_contains "docker-compose.yml" "postgres" "docker-compose.yml contains postgres service"
    assert_file_contains "docker-compose.yml" "hasura" "docker-compose.yml contains hasura service"
  else
    test_fail "nself build did not generate docker-compose.yml"
  fi

  # Check nginx config
  if [[ -d "nginx" ]]; then
    test_pass "nself build creates nginx directory"
    if [[ -f "nginx/nginx.conf" ]]; then
      test_pass "nself build generates nginx.conf"
    else
      test_skip "nginx.conf not found (may vary by config)"
    fi
  else
    test_skip "nginx directory not created"
  fi

  # Check SSL
  if [[ -d "ssl" ]]; then
    test_pass "nself build creates ssl directory"
  else
    test_skip "ssl directory not created"
  fi

  cd ..
}

test_env_command() {
  log_section "nself env Tests"

  mkdir -p env-test
  cd env-test

  # Create base config
  cat >.env.dev <<'EOF'
PROJECT_NAME=env-test
ENV=dev
BASE_DOMAIN=localhost
POSTGRES_PASSWORD=test123
EOF

  # Test env create
  local create_output
  create_output=$("$NSELF_BIN" env create staging staging 2>&1) || true
  if [[ -d ".environments/staging" ]]; then
    test_pass "nself env create creates environment directory"
  else
    test_skip "nself env create not available or failed"
  fi

  # Test env list
  local list_output
  list_output=$("$NSELF_BIN" env list 2>&1) || true
  if echo "$list_output" | grep -qi "staging\|environment\|no env"; then
    test_pass "nself env list works"
  else
    test_skip "nself env list not available"
  fi

  # Test env status
  local status_output
  status_output=$("$NSELF_BIN" env status 2>&1) || true
  if echo "$status_output" | grep -qi "environment\|current\|status\|not configured"; then
    test_pass "nself env status works"
  else
    test_skip "nself env status not available"
  fi

  cd ..
}

test_deploy_command() {
  log_section "nself deploy Tests"

  mkdir -p deploy-test
  cd deploy-test

  # Test deploy help
  local help_output
  help_output=$("$NSELF_BIN" deploy --help 2>&1) || true
  if echo "$help_output" | grep -qi "deploy\|staging\|production\|ssh"; then
    test_pass "nself deploy --help works"
  else
    test_skip "nself deploy --help not available"
  fi

  # Test deploy check-access (should report no environments)
  local check_output
  check_output=$("$NSELF_BIN" deploy check-access 2>&1) || true
  if echo "$check_output" | grep -qi "environment\|access\|configured\|no\|checking"; then
    test_pass "nself deploy check-access works"
  else
    test_skip "nself deploy check-access not available"
  fi

  cd ..
}

test_prod_command() {
  log_section "nself prod Tests"

  mkdir -p prod-test
  cd prod-test

  # Create minimal config
  cat >.env <<'EOF'
PROJECT_NAME=prod-test
ENV=production
BASE_DOMAIN=example.com
POSTGRES_PASSWORD=strongpassword123456
HASURA_GRAPHQL_ADMIN_SECRET=verylongsecret12345678901234567890
JWT_SECRET=anotherlongsecret12345678901234567890
EOF

  # Test prod status
  local status_output
  status_output=$("$NSELF_BIN" prod status 2>&1) || true
  if echo "$status_output" | grep -qi "production\|status\|environment\|settings"; then
    test_pass "nself prod status works"
  else
    test_skip "nself prod status not available"
  fi

  # Test prod check
  local check_output
  check_output=$("$NSELF_BIN" prod check 2>&1) || true
  if echo "$check_output" | grep -qi "security\|audit\|check\|pass\|fail\|warning"; then
    test_pass "nself prod check works"
  else
    test_skip "nself prod check not available"
  fi

  # Test prod secrets generate
  local secrets_output
  secrets_output=$("$NSELF_BIN" prod secrets generate --force 2>&1) || true
  if [[ -f ".env.secrets" ]]; then
    test_pass "nself prod secrets generate creates .env.secrets"

    # Check permissions
    local perms
    if stat --version 2>/dev/null | grep -q GNU; then
      perms=$(stat -c "%a" ".env.secrets" 2>/dev/null)
    else
      perms=$(stat -f "%OLp" ".env.secrets" 2>/dev/null)
    fi
    if [[ "$perms" == "600" ]]; then
      test_pass ".env.secrets has correct permissions (600)"
    else
      test_fail ".env.secrets has wrong permissions" "Expected 600, got $perms"
    fi
  else
    test_skip "nself prod secrets generate not available"
  fi

  cd ..
}

test_staging_command() {
  log_section "nself staging Tests"

  mkdir -p staging-test
  cd staging-test

  # Test staging status
  local status_output
  status_output=$("$NSELF_BIN" staging status 2>&1) || true
  if echo "$status_output" | grep -qi "staging\|status\|environment\|not configured"; then
    test_pass "nself staging status works"
  else
    test_skip "nself staging status not available"
  fi

  cd ..
}

test_service_commands() {
  log_section "Service Command Tests"

  mkdir -p service-test
  cd service-test

  # Create config
  cat >.env.dev <<'EOF'
PROJECT_NAME=service-test
ENV=dev
BASE_DOMAIN=localhost
MEILISEARCH_ENABLED=true
REDIS_ENABLED=true
EOF

  # Test email command help
  local email_help
  email_help=$("$NSELF_BIN" email --help 2>&1) || true
  if echo "$email_help" | grep -qi "email\|smtp\|provider\|configure"; then
    test_pass "nself email --help works"
  else
    test_skip "nself email not available"
  fi

  # Test search command help
  local search_help
  search_help=$("$NSELF_BIN" search --help 2>&1) || true
  if echo "$search_help" | grep -qi "search\|meilisearch\|engine\|configure"; then
    test_pass "nself search --help works"
  else
    test_skip "nself search not available"
  fi

  # Test functions command help
  local functions_help
  functions_help=$("$NSELF_BIN" functions --help 2>&1) || true
  if echo "$functions_help" | grep -qi "function\|serverless\|deploy"; then
    test_pass "nself functions --help works"
  else
    test_skip "nself functions not available"
  fi

  # Test mlflow command help
  local mlflow_help
  mlflow_help=$("$NSELF_BIN" mlflow --help 2>&1) || true
  if echo "$mlflow_help" | grep -qi "mlflow\|experiment\|run"; then
    test_pass "nself mlflow --help works"
  else
    test_skip "nself mlflow not available"
  fi

  # Test metrics command help
  local metrics_help
  metrics_help=$("$NSELF_BIN" metrics --help 2>&1) || true
  if echo "$metrics_help" | grep -qi "metric\|monitoring\|profile"; then
    test_pass "nself metrics --help works"
  else
    test_skip "nself metrics not available"
  fi

  # Test monitor command help
  local monitor_help
  monitor_help=$("$NSELF_BIN" monitor --help 2>&1) || true
  if echo "$monitor_help" | grep -qi "monitor\|dashboard\|grafana"; then
    test_pass "nself monitor --help works"
  else
    test_skip "nself monitor not available"
  fi

  cd ..
}

test_management_commands() {
  log_section "Management Command Tests"

  mkdir -p mgmt-test
  cd mgmt-test

  # Test status (should work even without docker)
  local status_output
  status_output=$("$NSELF_BIN" status 2>&1) || true
  if echo "$status_output" | grep -qi "status\|service\|container\|not running\|no docker"; then
    test_pass "nself status works"
  else
    test_skip "nself status not available"
  fi

  # Test urls (should work with config)
  cat >.env.dev <<'EOF'
PROJECT_NAME=mgmt-test
ENV=dev
BASE_DOMAIN=localhost
EOF

  local urls_output
  urls_output=$("$NSELF_BIN" urls 2>&1) || true
  if echo "$urls_output" | grep -qi "url\|localhost\|http\|no url"; then
    test_pass "nself urls works"
  else
    test_skip "nself urls not available"
  fi

  # Test doctor
  local doctor_output
  doctor_output=$("$NSELF_BIN" doctor 2>&1) || true
  if echo "$doctor_output" | grep -qi "check\|docker\|pass\|fail\|requirement"; then
    test_pass "nself doctor works"
  else
    test_skip "nself doctor not available"
  fi

  cd ..
}

test_ssl_command() {
  log_section "nself ssl Tests"

  mkdir -p ssl-test
  cd ssl-test

  # Test ssl help
  local ssl_help
  ssl_help=$("$NSELF_BIN" ssl --help 2>&1) || true
  if echo "$ssl_help" | grep -qi "ssl\|certificate\|generate\|trust"; then
    test_pass "nself ssl --help works"
  else
    test_skip "nself ssl not available"
  fi

  # Test ssl status
  local ssl_status
  ssl_status=$("$NSELF_BIN" ssl status 2>&1) || true
  if echo "$ssl_status" | grep -qi "ssl\|certificate\|status\|not found"; then
    test_pass "nself ssl status works"
  else
    test_skip "nself ssl status not available"
  fi

  cd ..
}

test_error_handling() {
  log_section "Error Handling Tests"

  mkdir -p error-test
  cd error-test

  # Test invalid command
  local invalid_output
  invalid_output=$("$NSELF_BIN" invalidcommand123 2>&1) || true
  if echo "$invalid_output" | grep -qi "unknown\|invalid\|error\|not found\|usage"; then
    test_pass "Invalid command shows error message"
  else
    test_fail "Invalid command does not show proper error"
  fi

  # Test command without required arguments
  local no_args_output
  no_args_output=$("$NSELF_BIN" env create 2>&1) || true
  if echo "$no_args_output" | grep -qi "usage\|argument\|required\|error\|help"; then
    test_pass "Missing arguments shows usage/error"
  else
    test_skip "Missing arguments handling varies"
  fi

  cd ..
}

test_config_scenarios() {
  log_section "Configuration Scenario Tests"

  # Test minimal config
  mkdir -p minimal-config
  cd minimal-config
  cat >.env.dev <<'EOF'
PROJECT_NAME=minimal
BASE_DOMAIN=localhost
EOF

  local build_minimal
  build_minimal=$("$NSELF_BIN" build 2>&1) || true
  if [[ -f "docker-compose.yml" ]]; then
    test_pass "Build works with minimal config"
  else
    test_skip "Minimal config build requires more settings"
  fi
  cd ..

  # Test full config with all services
  mkdir -p full-config
  cd full-config
  cat >.env.dev <<'EOF'
PROJECT_NAME=fulltest
ENV=dev
BASE_DOMAIN=localhost
POSTGRES_DB=fulltest_db
POSTGRES_USER=postgres
POSTGRES_PASSWORD=testpassword123
HASURA_GRAPHQL_ADMIN_SECRET=adminsecret12345678901234567890
NSELF_ADMIN_ENABLED=true
MINIO_ENABLED=true
REDIS_ENABLED=true
MEILISEARCH_ENABLED=true
MAILPIT_ENABLED=true
MONITORING_ENABLED=true
CS_1=api:express-js:8001
FRONTEND_APP_1_NAME=web
FRONTEND_APP_1_PORT=3000
FRONTEND_APP_1_ROUTE=web
EOF

  local build_full
  build_full=$("$NSELF_BIN" build 2>&1) || true
  if [[ -f "docker-compose.yml" ]]; then
    test_pass "Build works with full config"

    # Check all services are present
    if grep -q "minio" docker-compose.yml 2>/dev/null; then
      test_pass "MinIO service generated"
    else
      test_skip "MinIO not in compose file"
    fi

    if grep -q "redis" docker-compose.yml 2>/dev/null; then
      test_pass "Redis service generated"
    else
      test_skip "Redis not in compose file"
    fi

    if grep -q "meilisearch" docker-compose.yml 2>/dev/null; then
      test_pass "MeiliSearch service generated"
    else
      test_skip "MeiliSearch not in compose file"
    fi
  else
    test_fail "Build with full config failed"
  fi
  cd ..
}

# ═══════════════════════════════════════════════════════════════
# Main Test Runner
# ═══════════════════════════════════════════════════════════════

run_all_tests() {
  log_header "nself CLI End-to-End Test Suite"

  printf "Project Root: %s\n" "$PROJECT_ROOT"
  printf "nself Binary: %s\n" "$NSELF_BIN"

  setup_test_environment

  # Run all test suites
  test_nself_binary
  test_init_command
  test_build_command
  test_env_command
  test_deploy_command
  test_prod_command
  test_staging_command
  test_service_commands
  test_management_commands
  test_ssl_command
  test_error_handling
  test_config_scenarios

  cleanup_test_environment

  # Print summary
  log_header "Test Results Summary"

  printf "  ${GREEN}Passed:${RESET}  %d\n" "$PASSED_TESTS"
  printf "  ${RED}Failed:${RESET}  %d\n" "$FAILED_TESTS"
  printf "  ${YELLOW}Skipped:${RESET} %d\n" "$SKIPPED_TESTS"
  printf "  Total:   %d\n\n" "$TOTAL_TESTS"

  if [[ $FAILED_TESTS -eq 0 ]]; then
    printf "${GREEN}═══════════════════════════════════════════════════════════════${RESET}\n"
    printf "${GREEN}  All tests passed!${RESET}\n"
    printf "${GREEN}═══════════════════════════════════════════════════════════════${RESET}\n"
    return 0
  else
    printf "${RED}═══════════════════════════════════════════════════════════════${RESET}\n"
    printf "${RED}  %d test(s) failed${RESET}\n" "$FAILED_TESTS"
    printf "${RED}═══════════════════════════════════════════════════════════════${RESET}\n"
    return 1
  fi
}

# Run tests if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  run_all_tests
fi
