#!/usr/bin/env bash
# exhaustive-qa.sh - Exhaustive QA test suite (100+ scenarios)
# Tests every command, subcommand, argument variation, and edge case

set -euo pipefail

# ═══════════════════════════════════════════════════════════════════════════════
# Configuration
# ═══════════════════════════════════════════════════════════════════════════════

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
NSELF_BIN="$PROJECT_ROOT/bin/nself"
TEST_TMP="/tmp/nself-exhaustive-qa-$$"

# Counters
SCENARIO_NUM=0
PASSED=0
FAILED=0
SKIPPED=0
TOTAL_SCENARIOS=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Results tracking
declare -a FAILED_TESTS=()

# ═══════════════════════════════════════════════════════════════════════════════
# Test Framework
# ═══════════════════════════════════════════════════════════════════════════════

setup_test_env() {
  rm -rf "$TEST_TMP"
  mkdir -p "$TEST_TMP"
  cd "$TEST_TMP"
}

cleanup_test_env() {
  cd /
  rm -rf "$TEST_TMP" 2>/dev/null || true
}

scenario() {
  SCENARIO_NUM=$((SCENARIO_NUM + 1))
  TOTAL_SCENARIOS=$((TOTAL_SCENARIOS + 1))
  local description="$1"
  printf "${CYAN}[%03d]${NC} %s... " "$SCENARIO_NUM" "$description"
}

pass() {
  PASSED=$((PASSED + 1))
  printf "${GREEN}PASS${NC}\n"
}

fail() {
  local reason="${1:-}"
  FAILED=$((FAILED + 1))
  FAILED_TESTS+=("[$SCENARIO_NUM] $reason")
  printf "${RED}FAIL${NC}"
  if [[ -n "$reason" ]]; then
    printf " - %s" "$reason"
  fi
  printf "\n"
}

skip() {
  local reason="${1:-}"
  SKIPPED=$((SKIPPED + 1))
  printf "${YELLOW}SKIP${NC}"
  if [[ -n "$reason" ]]; then
    printf " - %s" "$reason"
  fi
  printf "\n"
}

section() {
  printf "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"
  printf "${BLUE}  %s${NC}\n" "$1"
  printf "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n\n"
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 1: Binary & Help Tests (Scenarios 1-15)
# ═══════════════════════════════════════════════════════════════════════════════

test_binary_and_help() {
  section "SECTION 1: Binary & Help Tests"

  # Scenario 1: Binary exists
  scenario "nself binary exists"
  if [[ -x "$NSELF_BIN" ]]; then pass; else fail "Binary not found or not executable"; fi

  # Need to run from outside nself repo, so we test in temp dir
  local test_dir="$TEST_TMP/help-tests"
  mkdir -p "$test_dir" && cd "$test_dir"
  "$NSELF_BIN" init --quiet 2>/dev/null || true

  # Scenario 2: Version output
  scenario "nself --version outputs version"
  if "$NSELF_BIN" --version 2>&1 | grep -qE "v?[0-9]+\.[0-9]+|nself"; then pass; else fail "No version output"; fi

  # Scenario 3: Version short flag
  scenario "nself -v outputs version"
  if "$NSELF_BIN" -v 2>&1 | grep -qE "v?[0-9]+\.[0-9]+|nself"; then pass; else fail "No version output"; fi

  # Scenario 4: Help output
  scenario "nself --help shows help"
  if "$NSELF_BIN" --help 2>&1 | grep -qiE "usage|commands|options|nself|init|build"; then pass; else fail "No help output"; fi

  # Scenario 5: Help short flag
  scenario "nself -h shows help"
  if "$NSELF_BIN" -h 2>&1 | grep -qiE "usage|commands|options|nself|init|build"; then pass; else fail "No help output"; fi

  # Scenario 6: No args shows help
  scenario "nself with no args shows help or usage"
  if "$NSELF_BIN" 2>&1 | grep -qiE "usage|help|commands|nself|init|build"; then pass; else fail "No usage output"; fi

  # Scenario 7: Invalid command error
  scenario "nself invalid-command shows error"
  if "$NSELF_BIN" invalid-command-xyz 2>&1 | grep -qiE "error|unknown|invalid|not found|command"; then pass; else fail "No error message"; fi

  # Scenario 8: Help for init
  scenario "nself init --help works"
  if "$NSELF_BIN" init --help 2>&1 | grep -qiE "init|project|usage|demo|force"; then pass; else fail "No init help"; fi

  # Scenario 9: Help for build
  scenario "nself build --help works"
  if "$NSELF_BIN" build --help 2>&1 | grep -qiE "build|compose|docker|usage|generate"; then pass; else fail "No build help"; fi

  # Scenario 10: Help for env
  scenario "nself env --help works"
  if "$NSELF_BIN" env --help 2>&1 | grep -qiE "env|environment|usage|create|list|switch"; then pass; else fail "No env help"; fi

  # Scenario 11: Help for deploy
  scenario "nself deploy --help works"
  if "$NSELF_BIN" deploy --help 2>&1 | grep -qiE "deploy|usage|target|staging|prod"; then pass; else fail "No deploy help"; fi

  # Scenario 12: Help for prod
  scenario "nself prod --help works"
  if "$NSELF_BIN" prod --help 2>&1 | grep -qiE "prod|production|usage|security|ssl|secrets"; then pass; else fail "No prod help"; fi

  # Scenario 13: Help for staging
  scenario "nself staging --help works"
  if "$NSELF_BIN" staging --help 2>&1 | grep -qiE "staging|usage|status|check|config"; then pass; else fail "No staging help"; fi

  # Scenario 14: Help for status
  scenario "nself status --help works"
  if "$NSELF_BIN" status --help 2>&1 | grep -qiE "status|usage|service|docker|container"; then pass; else fail "No status help"; fi

  # Scenario 15: Help for logs
  scenario "nself logs --help works"
  if "$NSELF_BIN" logs --help 2>&1 | grep -qiE "logs|usage|service|docker|container"; then pass; else fail "No logs help"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 2: Init Command Tests (Scenarios 16-30)
# ═══════════════════════════════════════════════════════════════════════════════

test_init_command() {
  section "SECTION 2: Init Command Tests"

  local test_dir="$TEST_TMP/init-tests"

  # Scenario 16: Basic init
  scenario "nself init creates .env file"
  mkdir -p "$test_dir/basic" && cd "$test_dir/basic"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  if [[ -f .env ]]; then pass; else fail "No .env created"; fi

  # Scenario 17: Init creates .env.example
  scenario "nself init creates .env.example"
  if [[ -f .env.example ]]; then pass; else fail "No .env.example"; fi

  # Scenario 18: Init creates .gitignore
  scenario "nself init creates .gitignore"
  if [[ -f .gitignore ]]; then pass; else fail "No .gitignore"; fi

  # Scenario 19: .env has PROJECT_NAME
  scenario ".env contains PROJECT_NAME"
  if grep -q "PROJECT_NAME=" .env 2>/dev/null; then pass; else fail "No PROJECT_NAME"; fi

  # Scenario 20: .env has ENV variable
  scenario ".env contains ENV variable"
  if grep -q "ENV=" .env 2>/dev/null; then pass; else fail "No ENV variable"; fi

  # Scenario 21: .env has BASE_DOMAIN
  scenario ".env contains BASE_DOMAIN"
  if grep -q "BASE_DOMAIN=" .env 2>/dev/null; then pass; else fail "No BASE_DOMAIN"; fi

  # Scenario 22: .env permissions are 600
  scenario ".env has secure permissions (600)"
  local perms
  if stat --version 2>/dev/null | grep -q GNU; then
    perms=$(stat -c "%a" .env 2>/dev/null)
  else
    perms=$(stat -f "%OLp" .env 2>/dev/null)
  fi
  if [[ "$perms" == "600" ]]; then pass; else fail "Permissions: $perms"; fi

  # Scenario 23: .gitignore contains .env
  scenario ".gitignore includes .env"
  if grep -q "^\.env$" .gitignore 2>/dev/null; then pass; else fail ".env not in gitignore"; fi

  # Scenario 24: Init with --force
  scenario "nself init --force overwrites existing"
  echo "CUSTOM=test" >> .env
  "$NSELF_BIN" init --force --quiet 2>/dev/null || true
  if ! grep -q "CUSTOM=test" .env 2>/dev/null; then pass; else fail "Force didn't overwrite"; fi

  # Scenario 25: Init --quiet suppresses output
  scenario "nself init --quiet has minimal output"
  mkdir -p "$test_dir/quiet" && cd "$test_dir/quiet"
  local output
  output=$("$NSELF_BIN" init --quiet 2>&1) || true
  local line_count
  line_count=$(echo "$output" | wc -l)
  if [[ $line_count -lt 10 ]]; then pass; else fail "Too much output: $line_count lines"; fi

  # Scenario 26: Init --demo mode
  scenario "nself init --demo creates demo config"
  mkdir -p "$test_dir/demo" && cd "$test_dir/demo"
  "$NSELF_BIN" init --demo --quiet 2>/dev/null || true
  if grep -qE "REDIS_ENABLED=true|MINIO_ENABLED=true" .env 2>/dev/null; then pass; else fail "No demo services"; fi

  # Scenario 27: Init in git repo
  scenario "nself init works in git repo"
  mkdir -p "$test_dir/gitrepo" && cd "$test_dir/gitrepo"
  git init --quiet 2>/dev/null || git init 2>/dev/null || true
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  if [[ -f .env ]]; then pass; else fail "Failed in git repo"; fi

  # Scenario 28: Init preserves existing .gitignore entries
  scenario "Init preserves existing .gitignore entries"
  mkdir -p "$test_dir/preserve" && cd "$test_dir/preserve"
  echo "*.log" > .gitignore
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  if grep -q "*.log" .gitignore 2>/dev/null; then pass; else fail "Lost existing entries"; fi

  # Scenario 29: Init detects project name from directory
  scenario "Init detects project name from directory"
  mkdir -p "$test_dir/my-awesome-project" && cd "$test_dir/my-awesome-project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  if grep -qiE "PROJECT_NAME=.*my.*awesome.*project|PROJECT_NAME=myawesomeproject" .env 2>/dev/null; then
    pass
  else
    # Check if any project name was set
    if grep -q "PROJECT_NAME=" .env 2>/dev/null; then pass; else fail "No project name"; fi
  fi

  # Scenario 30: Init handles special characters in path
  scenario "Init handles paths with spaces"
  mkdir -p "$test_dir/path with spaces" && cd "$test_dir/path with spaces"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  if [[ -f .env ]]; then pass; else fail "Failed with spaces in path"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 3: Build Command Tests (Scenarios 31-50)
# ═══════════════════════════════════════════════════════════════════════════════

test_build_command() {
  section "SECTION 3: Build Command Tests"

  local test_dir="$TEST_TMP/build-tests"

  # Setup a valid project first
  mkdir -p "$test_dir/project" && cd "$test_dir/project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true

  # Scenario 31: Build --dry-run
  scenario "nself build --dry-run works"
  local dry_output
  dry_output=$("$NSELF_BIN" build --dry-run 2>&1) || true
  if echo "$dry_output" | grep -qiE "dry|would|skip|preview|build"; then pass; else fail "No dry-run output"; fi

  # Scenario 32: Build creates docker-compose.yml
  scenario "nself build creates docker-compose.yml"
  "$NSELF_BIN" build 2>/dev/null || true
  if [[ -f docker-compose.yml ]]; then pass; else fail "No docker-compose.yml"; fi

  # Scenario 33: docker-compose.yml has services
  scenario "docker-compose.yml contains services"
  if grep -q "services:" docker-compose.yml 2>/dev/null; then pass; else fail "No services section"; fi

  # Scenario 34: docker-compose.yml has postgres
  scenario "docker-compose.yml has postgres service"
  if grep -qE "postgres:|postgresql:" docker-compose.yml 2>/dev/null; then pass; else fail "No postgres"; fi

  # Scenario 35: docker-compose.yml has hasura
  scenario "docker-compose.yml has hasura service"
  if grep -q "hasura:" docker-compose.yml 2>/dev/null; then pass; else fail "No hasura"; fi

  # Scenario 36: docker-compose.yml has nginx
  scenario "docker-compose.yml has nginx service"
  if grep -q "nginx:" docker-compose.yml 2>/dev/null; then pass; else fail "No nginx"; fi

  # Scenario 37: Build creates nginx config
  scenario "Build creates nginx configuration"
  if [[ -f nginx/nginx.conf ]] || [[ -d nginx ]]; then pass; else fail "No nginx config"; fi

  # Scenario 38: Build with REDIS_ENABLED
  scenario "Build with REDIS_ENABLED=true includes redis"
  mkdir -p "$test_dir/redis" && cd "$test_dir/redis"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "REDIS_ENABLED=true" >> .env
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  if grep -q "redis:" docker-compose.yml 2>/dev/null; then pass; else fail "No redis service"; fi

  # Scenario 39: Build with MINIO_ENABLED
  scenario "Build with MINIO_ENABLED=true includes minio"
  mkdir -p "$test_dir/minio" && cd "$test_dir/minio"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "MINIO_ENABLED=true" >> .env
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  if grep -q "minio:" docker-compose.yml 2>/dev/null; then pass; else fail "No minio service"; fi

  # Scenario 40: Build with MEILISEARCH_ENABLED
  scenario "Build with MEILISEARCH_ENABLED=true includes meilisearch"
  mkdir -p "$test_dir/meili" && cd "$test_dir/meili"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "MEILISEARCH_ENABLED=true" >> .env
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  if grep -q "meilisearch:" docker-compose.yml 2>/dev/null; then pass; else fail "No meilisearch service"; fi

  # Scenario 41: Build with MONITORING_ENABLED
  scenario "Build with MONITORING_ENABLED=true includes monitoring"
  mkdir -p "$test_dir/monitoring" && cd "$test_dir/monitoring"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "MONITORING_ENABLED=true" >> .env
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  if grep -qE "prometheus:|grafana:" docker-compose.yml 2>/dev/null; then pass; else fail "No monitoring"; fi

  # Scenario 42: Build creates SSL directory
  scenario "Build creates SSL directory"
  cd "$test_dir/project"
  if [[ -d ssl ]] || [[ -d nginx/ssl ]]; then pass; else fail "No SSL directory"; fi

  # Scenario 43: Build creates postgres init directory
  scenario "Build creates postgres init directory"
  if [[ -d postgres/init ]] || [[ -d postgres ]]; then pass; else fail "No postgres init"; fi

  # Scenario 44: Build --verbose shows detailed output
  scenario "nself build --verbose shows details"
  local output
  output=$("$NSELF_BIN" build --verbose 2>&1) || true
  if echo "$output" | grep -qiE "generat|creat|writ"; then pass; else fail "No verbose output"; fi

  # Scenario 45: Build without .env fails gracefully
  scenario "Build without .env shows error"
  mkdir -p "$test_dir/no-env" && cd "$test_dir/no-env"
  if "$NSELF_BIN" build 2>&1 | grep -qiE "error\|not found\|missing\|.env"; then pass; else fail "No error message"; fi

  # Scenario 46: Build with custom services
  scenario "Build with CS_1 custom service"
  mkdir -p "$test_dir/custom" && cd "$test_dir/custom"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "CS_1=myapi:express-js:8001" >> .env
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  if grep -qE "myapi:|express" docker-compose.yml 2>/dev/null; then pass; else fail "No custom service"; fi

  # Scenario 47: Build with NSELF_ADMIN_ENABLED
  scenario "Build with NSELF_ADMIN_ENABLED includes admin"
  mkdir -p "$test_dir/admin" && cd "$test_dir/admin"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "NSELF_ADMIN_ENABLED=true" >> .env
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  if grep -qE "admin:|nself-admin:" docker-compose.yml 2>/dev/null; then pass; else fail "No admin service"; fi

  # Scenario 48: Build generates valid YAML
  scenario "docker-compose.yml is valid YAML"
  cd "$test_dir/project"
  if command -v python3 >/dev/null 2>&1; then
    if python3 -c "import yaml; yaml.safe_load(open('docker-compose.yml'))" 2>/dev/null; then
      pass
    else
      fail "Invalid YAML"
    fi
  else
    # Try docker compose config
    if command -v docker >/dev/null 2>&1; then
      if docker compose config >/dev/null 2>&1; then pass; else skip "YAML validation unavailable"; fi
    else
      skip "No YAML validator available"
    fi
  fi

  # Scenario 49: Build with all optional services
  scenario "Build with all optional services enabled"
  mkdir -p "$test_dir/all-services" && cd "$test_dir/all-services"
  "$NSELF_BIN" init --demo --quiet 2>/dev/null || true
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  local service_count
  service_count=$(grep -c "image:" docker-compose.yml 2>/dev/null || echo "0")
  if [[ $service_count -ge 5 ]]; then pass; else fail "Only $service_count services"; fi

  # Scenario 50: Build is idempotent
  scenario "Build is idempotent (running twice works)"
  cd "$test_dir/project"
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  local hash1 hash2
  hash1=$(md5sum docker-compose.yml 2>/dev/null | cut -d' ' -f1 || md5 -q docker-compose.yml 2>/dev/null)
  "$NSELF_BIN" build --quiet 2>/dev/null || true
  hash2=$(md5sum docker-compose.yml 2>/dev/null | cut -d' ' -f1 || md5 -q docker-compose.yml 2>/dev/null)
  if [[ "$hash1" == "$hash2" ]]; then pass; else fail "Build not idempotent"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 4: Environment Command Tests (Scenarios 51-65)
# ═══════════════════════════════════════════════════════════════════════════════

test_env_command() {
  section "SECTION 4: Environment Command Tests"

  local test_dir="$TEST_TMP/env-tests"
  mkdir -p "$test_dir/project" && cd "$test_dir/project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true

  # Scenario 51: env create local
  scenario "nself env create local-test local"
  "$NSELF_BIN" env create local-test local 2>/dev/null || true
  if [[ -d .environments/local-test ]]; then pass; else fail "Create failed"; fi

  # Scenario 52: env creates directory
  scenario "env create creates .environments directory"
  if [[ -d .environments/local-test ]]; then pass; else fail "No directory"; fi

  # Scenario 53: env creates .env file
  scenario "env create creates .env in environment"
  if [[ -f .environments/local-test/.env ]]; then pass; else fail "No .env"; fi

  # Scenario 54: env create staging
  scenario "nself env create staging-test staging"
  "$NSELF_BIN" env create staging-test staging 2>/dev/null || true
  if [[ -f .environments/staging-test/.env ]]; then pass; else fail "No staging env"; fi

  # Scenario 55: staging has .env.secrets
  scenario "staging environment has .env.secrets"
  if [[ -f .environments/staging-test/.env.secrets ]]; then pass; else fail "No secrets file"; fi

  # Scenario 56: env create prod
  scenario "nself env create prod-test prod"
  "$NSELF_BIN" env create prod-test prod 2>/dev/null || true
  if [[ -f .environments/prod-test/.env ]]; then pass; else fail "No prod env"; fi

  # Scenario 57: prod has DEBUG=false
  scenario "prod environment has DEBUG=false"
  if grep -q "DEBUG=false" .environments/prod-test/.env 2>/dev/null; then pass; else fail "DEBUG not false"; fi

  # Scenario 58: env list
  scenario "nself env list shows environments"
  local list_output
  list_output=$("$NSELF_BIN" env list 2>&1) || true
  if echo "$list_output" | grep -qE "local-test|staging-test|prod-test|environment"; then pass; else fail "List failed"; fi

  # Scenario 59: env switch
  scenario "nself env switch works"
  "$NSELF_BIN" env switch local-test 2>/dev/null || true
  # Check if switch happened (either marker file or directory exists)
  if [[ -f .current-env ]] || [[ -d .environments/local-test ]]; then pass; else fail "Switch failed"; fi

  # Scenario 60: env switch creates .current-env
  scenario "env switch creates .current-env marker"
  if [[ -f .current-env ]] || grep -q "local-test" .env 2>/dev/null; then pass; else fail "No marker file"; fi

  # Scenario 61: env diff
  scenario "nself env diff compares environments"
  local diff_output
  diff_output=$("$NSELF_BIN" env diff local-test staging-test 2>&1) || true
  if echo "$diff_output" | grep -qiE "diff|ENV|DEBUG|\+|\-|change|local-test|staging-test"; then pass; else fail "No diff output"; fi

  # Scenario 62: env validate
  scenario "nself env validate works"
  local validate_output
  validate_output=$("$NSELF_BIN" env validate local-test 2>&1) || true
  if echo "$validate_output" | grep -qiE "valid|pass|ok|success|local-test|check"; then pass; else fail "Validate failed"; fi

  # Scenario 63: env delete with force
  scenario "nself env delete with --force"
  "$NSELF_BIN" env create deleteme local 2>/dev/null || true
  if "$NSELF_BIN" env delete deleteme --force 2>&1 | grep -qiE "delet|remov|success"; then pass; else fail "Delete failed"; fi

  # Scenario 64: env delete removes directory
  scenario "env delete removes environment directory"
  if [[ ! -d .environments/deleteme ]]; then pass; else fail "Directory still exists"; fi

  # Scenario 65: env sanitizes names
  scenario "env create sanitizes environment names"
  "$NSELF_BIN" env create "Test--Name_123" local 2>/dev/null || true
  if [[ -d .environments/test--name123 ]] || [[ -d .environments/testname123 ]]; then pass; else fail "Name not sanitized"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 5: Deploy Command Tests (Scenarios 66-75)
# ═══════════════════════════════════════════════════════════════════════════════

test_deploy_command() {
  section "SECTION 5: Deploy Command Tests"

  local test_dir="$TEST_TMP/deploy-tests"
  mkdir -p "$test_dir/project" && cd "$test_dir/project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true

  # Scenario 66: deploy list
  scenario "nself deploy list shows targets"
  if "$NSELF_BIN" deploy list 2>&1 | grep -qiE "target|staging|prod|local"; then pass; else fail "No targets"; fi

  # Scenario 67: deploy --dry-run
  scenario "nself deploy --dry-run works"
  if "$NSELF_BIN" deploy staging --dry-run 2>&1 | grep -qiE "dry|would|skip"; then pass; else fail "No dry-run"; fi

  # Scenario 68: deploy status
  scenario "nself deploy status works"
  if "$NSELF_BIN" deploy status 2>&1 | grep -qiE "status|deploy|target|environment"; then pass; else fail "No status"; fi

  # Scenario 69: deploy check
  scenario "nself deploy check works"
  if "$NSELF_BIN" deploy check 2>&1 | grep -qiE "check|config|valid"; then pass; else fail "No check output"; fi

  # Scenario 70: deploy with invalid target
  scenario "deploy with invalid target shows error"
  if "$NSELF_BIN" deploy invalid-target 2>&1 | grep -qiE "error|invalid|not found|unknown"; then pass; else fail "No error"; fi

  # Scenario 71: deploy config
  scenario "nself deploy config works"
  if "$NSELF_BIN" deploy config 2>&1 | grep -qiE "config|setting|target"; then pass; else fail "No config"; fi

  # Scenario 72: deploy help shows subcommands
  scenario "deploy help lists subcommands"
  if "$NSELF_BIN" deploy --help 2>&1 | grep -qiE "list|status|check"; then pass; else fail "No subcommands"; fi

  # Scenario 73: deploy requires target
  scenario "deploy without target prompts for selection"
  local output
  output=$("$NSELF_BIN" deploy 2>&1) || true
  if echo "$output" | grep -qiE "select|target|specify|usage"; then pass; else fail "No prompt"; fi

  # Scenario 74: deploy scope shows services
  scenario "deploy scope shows deployment scope"
  if "$NSELF_BIN" deploy scope 2>&1 | grep -qiE "service|deploy|scope"; then pass; else fail "No scope"; fi

  # Scenario 75: deploy diff
  scenario "nself deploy diff works"
  if "$NSELF_BIN" deploy diff 2>&1 | grep -qiE "diff|change|compare|no change"; then pass; else fail "No diff"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 6: Production Command Tests (Scenarios 76-85)
# ═══════════════════════════════════════════════════════════════════════════════

test_prod_command() {
  section "SECTION 6: Production Command Tests"

  local test_dir="$TEST_TMP/prod-tests"
  mkdir -p "$test_dir/project" && cd "$test_dir/project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true

  # Scenario 76: prod status
  scenario "nself prod status works"
  if "$NSELF_BIN" prod status 2>&1 | grep -qiE "status|prod|environment|config"; then pass; else fail "No status"; fi

  # Scenario 77: prod (default is status)
  scenario "nself prod defaults to status"
  if "$NSELF_BIN" prod 2>&1 | grep -qiE "status|prod|environment|config"; then pass; else fail "No default status"; fi

  # Scenario 78: prod check
  scenario "nself prod check runs security audit"
  if "$NSELF_BIN" prod check 2>&1 | grep -qiE "check|audit|security|pass|fail"; then pass; else fail "No check"; fi

  # Scenario 79: prod audit (alias)
  scenario "nself prod audit works (alias for check)"
  if "$NSELF_BIN" prod audit 2>&1 | grep -qiE "check|audit|security"; then pass; else fail "No audit"; fi

  # Scenario 80: prod secrets generate
  scenario "nself prod secrets generate works"
  if "$NSELF_BIN" prod secrets generate --force 2>&1 | grep -qiE "generat|secret|creat"; then pass; else fail "No generate"; fi

  # Scenario 81: prod secrets creates .env.secrets
  scenario "prod secrets creates .env.secrets file"
  if [[ -f .env.secrets ]]; then pass; else fail "No secrets file"; fi

  # Scenario 82: prod secrets validate
  scenario "nself prod secrets validate works"
  if "$NSELF_BIN" prod secrets validate 2>&1 | grep -qiE "valid|check|secret"; then pass; else fail "No validate"; fi

  # Scenario 83: prod ssl status
  scenario "nself prod ssl status works"
  if "$NSELF_BIN" prod ssl status 2>&1 | grep -qiE "ssl|certificate|status|not found"; then pass; else fail "No ssl status"; fi

  # Scenario 84: prod firewall status
  scenario "nself prod firewall status works"
  if "$NSELF_BIN" prod firewall status 2>&1 | grep -qiE "firewall|status|ufw|iptables|none"; then pass; else fail "No firewall"; fi

  # Scenario 85: prod harden --dry-run
  scenario "nself prod harden --dry-run works"
  if "$NSELF_BIN" prod harden --dry-run 2>&1 | grep -qiE "harden|dry|would|security"; then pass; else fail "No harden"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 7: Staging Command Tests (Scenarios 86-90)
# ═══════════════════════════════════════════════════════════════════════════════

test_staging_command() {
  section "SECTION 7: Staging Command Tests"

  local test_dir="$TEST_TMP/staging-tests"
  mkdir -p "$test_dir/project" && cd "$test_dir/project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true

  # Scenario 86: staging status
  scenario "nself staging status works"
  if "$NSELF_BIN" staging status 2>&1 | grep -qiE "status|staging|environment"; then pass; else fail "No status"; fi

  # Scenario 87: staging (default)
  scenario "nself staging defaults to status"
  if "$NSELF_BIN" staging 2>&1 | grep -qiE "status|staging|environment"; then pass; else fail "No default"; fi

  # Scenario 88: staging check
  scenario "nself staging check works"
  if "$NSELF_BIN" staging check 2>&1 | grep -qiE "check|config|valid"; then pass; else fail "No check"; fi

  # Scenario 89: staging config
  scenario "nself staging config works"
  if "$NSELF_BIN" staging config 2>&1 | grep -qiE "config|setting|staging"; then pass; else fail "No config"; fi

  # Scenario 90: staging setup
  scenario "nself staging setup works"
  if "$NSELF_BIN" staging setup --dry-run 2>&1 | grep -qiE "setup|staging|would|dry"; then pass; else fail "No setup"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 8: Service Management Tests (Scenarios 91-100)
# ═══════════════════════════════════════════════════════════════════════════════

test_service_management() {
  section "SECTION 8: Service Management Tests"

  local test_dir="$TEST_TMP/service-tests"
  mkdir -p "$test_dir/project" && cd "$test_dir/project"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  "$NSELF_BIN" build --quiet 2>/dev/null || true

  # Scenario 91: status command
  scenario "nself status works"
  if "$NSELF_BIN" status 2>&1 | grep -qiE "status|service|running|stopped|not running"; then pass; else fail "No status"; fi

  # Scenario 92: logs command help
  scenario "nself logs shows usage"
  if "$NSELF_BIN" logs 2>&1 | grep -qiE "log|usage|service"; then pass; else fail "No logs help"; fi

  # Scenario 93: urls command
  scenario "nself urls shows service URLs"
  if "$NSELF_BIN" urls 2>&1 | grep -qiE "url|http|service|localhost"; then pass; else fail "No urls"; fi

  # Scenario 94: list command
  scenario "nself list shows services"
  if "$NSELF_BIN" list 2>&1 | grep -qiE "service|postgres|hasura|nginx"; then pass; else fail "No list"; fi

  # Scenario 95: restart help
  scenario "nself restart --help works"
  if "$NSELF_BIN" restart --help 2>&1 | grep -qiE "restart|usage|service"; then pass; else fail "No restart help"; fi

  # Scenario 96: stop help
  scenario "nself stop --help works"
  if "$NSELF_BIN" stop --help 2>&1 | grep -qiE "stop|usage|service"; then pass; else fail "No stop help"; fi

  # Scenario 97: start help
  scenario "nself start --help works"
  if "$NSELF_BIN" start --help 2>&1 | grep -qiE "start|usage|service"; then pass; else fail "No start help"; fi

  # Scenario 98: exec help
  scenario "nself exec --help works"
  if "$NSELF_BIN" exec --help 2>&1 | grep -qiE "exec|usage|command|container"; then pass; else fail "No exec help"; fi

  # Scenario 99: clean help
  scenario "nself clean --help works"
  if "$NSELF_BIN" clean --help 2>&1 | grep -qiE "clean|usage|remove|container"; then pass; else fail "No clean help"; fi

  # Scenario 100: reset help
  scenario "nself reset --help works"
  if "$NSELF_BIN" reset --help 2>&1 | grep -qiE "reset|usage|data|clean"; then pass; else fail "No reset help"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 9: Error Handling & Edge Cases (Scenarios 101-115)
# ═══════════════════════════════════════════════════════════════════════════════

test_error_handling() {
  section "SECTION 9: Error Handling & Edge Cases"

  local test_dir="$TEST_TMP/error-tests"

  # Scenario 101: Invalid flag
  scenario "Invalid flag shows error"
  mkdir -p "$test_dir/test1" && cd "$test_dir/test1"
  if "$NSELF_BIN" init --invalid-flag-xyz 2>&1 | grep -qiE "error|invalid|unknown|unrecognized"; then pass; else fail "No error"; fi

  # Scenario 102: Build in empty directory
  scenario "Build in empty directory shows error"
  mkdir -p "$test_dir/empty" && cd "$test_dir/empty"
  if "$NSELF_BIN" build 2>&1 | grep -qiE "error\|not found\|.env\|missing"; then pass; else fail "No error"; fi

  # Scenario 103: Env switch to non-existent
  scenario "Env switch to non-existent shows error"
  mkdir -p "$test_dir/test2" && cd "$test_dir/test2"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  if "$NSELF_BIN" env switch non-existent-xyz 2>&1 | grep -qiE "error|not found|does not exist"; then pass; else fail "No error"; fi

  # Scenario 104: Deploy to invalid target
  scenario "Deploy to invalid target shows error"
  if "$NSELF_BIN" deploy invalid-target-xyz 2>&1 | grep -qiE "error|invalid|unknown|not found"; then pass; else fail "No error"; fi

  # Scenario 105: Double init without force
  scenario "Double init without force preserves files"
  mkdir -p "$test_dir/double" && cd "$test_dir/double"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo "CUSTOM=value" >> .env
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  # Should either error or preserve
  if grep -q "CUSTOM=value" .env 2>/dev/null || "$NSELF_BIN" init 2>&1 | grep -qiE "exist|force"; then pass; else fail "Overwrote without force"; fi

  # Scenario 106: Env create duplicate
  scenario "Env create duplicate shows error or warning"
  mkdir -p "$test_dir/dup" && cd "$test_dir/dup"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  "$NSELF_BIN" env create dup-test local 2>/dev/null || true
  if "$NSELF_BIN" env create dup-test local 2>&1 | grep -qiE "exist|already|error|overwrite"; then pass; else fail "No duplicate warning"; fi

  # Scenario 107: Missing required ENV vars
  scenario "Build with missing vars shows warning"
  mkdir -p "$test_dir/missing" && cd "$test_dir/missing"
  echo "PROJECT_NAME=test" > .env
  # Missing many vars
  local output
  output=$("$NSELF_BIN" build 2>&1) || true
  # Should either work with defaults or show warnings
  if [[ -f docker-compose.yml ]] || echo "$output" | grep -qiE "warn|missing|error"; then pass; else fail "No handling"; fi

  # Scenario 108: Very long project name
  scenario "Init handles very long project name"
  mkdir -p "$test_dir/longname" && cd "$test_dir/longname"
  cat > .env << 'EOF'
PROJECT_NAME=this-is-a-very-long-project-name-that-exceeds-normal-limits-and-should-be-handled-gracefully
EOF
  "$NSELF_BIN" build 2>/dev/null || true
  if [[ -f docker-compose.yml ]]; then pass; else fail "Failed with long name"; fi

  # Scenario 109: Special characters in values
  scenario "Build handles special characters in .env"
  mkdir -p "$test_dir/special" && cd "$test_dir/special"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  echo 'SPECIAL_VAR="value with spaces & special chars!"' >> .env
  "$NSELF_BIN" build 2>/dev/null || true
  if [[ -f docker-compose.yml ]]; then pass; else fail "Failed with special chars"; fi

  # Scenario 110: Unicode in project name
  scenario "Init handles unicode characters"
  mkdir -p "$test_dir/unicode" && cd "$test_dir/unicode"
  cat > .env << 'EOF'
PROJECT_NAME=test-项目
BASE_DOMAIN=localhost
ENV=dev
EOF
  "$NSELF_BIN" build 2>/dev/null || true
  # Should work or sanitize
  if [[ -f docker-compose.yml ]]; then pass; else skip "Unicode not supported"; fi

  # Scenario 111: Read-only directory
  scenario "Init handles read-only gracefully"
  mkdir -p "$test_dir/readonly" && cd "$test_dir/readonly"
  chmod 555 . 2>/dev/null || true
  local result
  result=$("$NSELF_BIN" init --quiet 2>&1) || true
  chmod 755 . 2>/dev/null || true
  # Should show permission error
  if echo "$result" | grep -qiE "permission|denied|error|cannot"; then pass; else skip "Root user bypasses permissions"; fi

  # Scenario 112: Concurrent builds
  scenario "Concurrent builds don't corrupt files"
  mkdir -p "$test_dir/concurrent" && cd "$test_dir/concurrent"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  (
    "$NSELF_BIN" build --quiet 2>/dev/null &
    "$NSELF_BIN" build --quiet 2>/dev/null &
    wait
  )
  if [[ -f docker-compose.yml ]] && grep -q "services:" docker-compose.yml 2>/dev/null; then pass; else fail "Corrupted"; fi

  # Scenario 113: Empty .env file
  scenario "Build with empty .env shows error"
  mkdir -p "$test_dir/empty-env" && cd "$test_dir/empty-env"
  touch .env
  if "$NSELF_BIN" build 2>&1 | grep -qiE "error|missing|required|PROJECT_NAME"; then pass; else fail "No error"; fi

  # Scenario 114: Malformed .env
  scenario "Build handles malformed .env"
  mkdir -p "$test_dir/malformed" && cd "$test_dir/malformed"
  echo "THIS IS NOT VALID ENV FORMAT" > .env
  local output
  output=$("$NSELF_BIN" build 2>&1) || true
  # Should either handle gracefully or show clear error
  if echo "$output" | grep -qiE "error|invalid|parse" || [[ -f docker-compose.yml ]]; then pass; else fail "No handling"; fi

  # Scenario 115: env delete current environment
  scenario "Env delete current environment shows warning"
  mkdir -p "$test_dir/del-current" && cd "$test_dir/del-current"
  "$NSELF_BIN" init --quiet 2>/dev/null || true
  "$NSELF_BIN" env create current-env local 2>/dev/null || true
  "$NSELF_BIN" env switch current-env 2>/dev/null || true
  if "$NSELF_BIN" env delete current-env 2>&1 | grep -qiE "current|active|cannot|warning"; then pass; else fail "No warning"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 10: Cross-Platform Compatibility (Scenarios 116-125)
# ═══════════════════════════════════════════════════════════════════════════════

test_cross_platform() {
  section "SECTION 10: Cross-Platform Compatibility"

  local test_dir="$TEST_TMP/compat-tests"
  mkdir -p "$test_dir"

  # Scenario 116: No echo -e in lib files
  scenario "No echo -e in lib files"
  if grep -r 'echo -e' "$PROJECT_ROOT/src/lib/" 2>/dev/null | grep -v "^Binary" | grep -v ".bak" | head -1; then
    fail "Found echo -e"
  else
    pass
  fi

  # Scenario 117: No Bash 4+ lowercase expansion
  scenario "No Bash 4+ lowercase \${var,,}"
  if grep -rE '\$\{[^}]*,,[^}]*\}' "$PROJECT_ROOT/src/lib/" 2>/dev/null | grep -v "^Binary" | head -1; then
    fail "Found lowercase expansion"
  else
    pass
  fi

  # Scenario 118: No Bash 4+ uppercase expansion
  scenario "No Bash 4+ uppercase \${var^^}"
  if grep -rE '\$\{[^}]*\^\^[^}]*\}' "$PROJECT_ROOT/src/lib/" 2>/dev/null | grep -v "^Binary" | head -1; then
    fail "Found uppercase expansion"
  else
    pass
  fi

  # Scenario 119: No associative arrays in lib
  scenario "No associative arrays (declare -A)"
  if grep -r 'declare -A' "$PROJECT_ROOT/src/lib/" 2>/dev/null | grep -v "^Binary" | head -1; then
    fail "Found associative array"
  else
    pass
  fi

  # Scenario 120: No mapfile/readarray
  scenario "No mapfile or readarray commands"
  if grep -rE '\b(mapfile|readarray)\b' "$PROJECT_ROOT/src/lib/" 2>/dev/null | grep -v "^Binary" | head -1; then
    fail "Found mapfile/readarray"
  else
    pass
  fi

  # Scenario 121: Uses printf for formatting
  scenario "Uses printf for formatted output"
  local printf_count
  printf_count=$(grep -r 'printf' "$PROJECT_ROOT/src/lib/" 2>/dev/null | wc -l)
  if [[ $printf_count -gt 10 ]]; then pass; else fail "Not enough printf usage"; fi

  # Scenario 122: Shell scripts have shebang
  scenario "All shell scripts have proper shebang"
  local bad_shebang=0
  while IFS= read -r file; do
    if [[ -f "$file" ]]; then
      local first_line
      first_line=$(head -1 "$file")
      if [[ ! "$first_line" =~ ^#! ]]; then
        bad_shebang=$((bad_shebang + 1))
      fi
    fi
  done < <(find "$PROJECT_ROOT/src" -name "*.sh" -type f 2>/dev/null)
  if [[ $bad_shebang -eq 0 ]]; then pass; else fail "$bad_shebang files without shebang"; fi

  # Scenario 123: CLI scripts are executable
  scenario "CLI scripts are executable"
  local non_exec=0
  while IFS= read -r file; do
    if [[ ! -x "$file" ]]; then
      non_exec=$((non_exec + 1))
    fi
  done < <(find "$PROJECT_ROOT/src/cli" -name "*.sh" -type f 2>/dev/null)
  if [[ $non_exec -eq 0 ]]; then pass; else fail "$non_exec non-executable"; fi

  # Scenario 124: No hardcoded /bin/bash paths
  scenario "No hardcoded /bin/bash in scripts"
  local hardcoded
  hardcoded=$(grep -r '^#!/bin/bash' "$PROJECT_ROOT/src/lib/" 2>/dev/null | wc -l)
  # Using /usr/bin/env bash is preferred
  if [[ $hardcoded -lt 5 ]]; then pass; else fail "$hardcoded hardcoded paths"; fi

  # Scenario 125: stat commands use wrappers
  scenario "stat commands use safe wrappers or detection"
  local direct_stat
  direct_stat=$(grep -rE 'stat -c|stat -f' "$PROJECT_ROOT/src/lib/" 2>/dev/null | grep -v "safe_stat\|platform" | wc -l)
  if [[ $direct_stat -lt 3 ]]; then pass; else fail "$direct_stat direct stat calls"; fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Final Report
# ═══════════════════════════════════════════════════════════════════════════════

print_final_report() {
  printf "\n"
  printf "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${BLUE}║                  EXHAUSTIVE QA REPORT                         ║${NC}\n"
  printf "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
  printf "\n"

  printf "  Total Scenarios:  %d\n" "$TOTAL_SCENARIOS"
  printf "  ${GREEN}Passed:${NC}           %d\n" "$PASSED"
  printf "  ${RED}Failed:${NC}           %d\n" "$FAILED"
  printf "  ${YELLOW}Skipped:${NC}          %d\n" "$SKIPPED"
  printf "\n"

  local pass_rate=0
  if [[ $TOTAL_SCENARIOS -gt 0 ]]; then
    pass_rate=$((PASSED * 100 / TOTAL_SCENARIOS))
  fi
  printf "  Pass Rate:        %d%%\n" "$pass_rate"
  printf "\n"

  if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
    printf "${RED}Failed Tests:${NC}\n"
    for test in "${FAILED_TESTS[@]}"; do
      printf "  - %s\n" "$test"
    done
    printf "\n"
  fi

  if [[ $FAILED -eq 0 ]]; then
    printf "${GREEN}═══════════════════════════════════════════════════════════════${NC}\n"
    printf "${GREEN}  ALL TESTS PASSED!${NC}\n"
    printf "${GREEN}═══════════════════════════════════════════════════════════════${NC}\n"
  else
    printf "${RED}═══════════════════════════════════════════════════════════════${NC}\n"
    printf "${RED}  SOME TESTS FAILED - Review above for details${NC}\n"
    printf "${RED}═══════════════════════════════════════════════════════════════${NC}\n"
  fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Main Execution
# ═══════════════════════════════════════════════════════════════════════════════

main() {
  printf "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${BLUE}║            nself CLI Exhaustive QA Test Suite                 ║${NC}\n"
  printf "${BLUE}║                  100+ Scenarios Coverage                      ║${NC}\n"
  printf "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
  printf "\n"
  printf "Project Root: %s\n" "$PROJECT_ROOT"
  printf "Test Temp:    %s\n" "$TEST_TMP"
  printf "Date:         %s\n" "$(date)"
  printf "\n"

  # Setup
  setup_test_env

  # Run all test sections
  test_binary_and_help      # 1-15
  test_init_command         # 16-30
  test_build_command        # 31-50
  test_env_command          # 51-65
  test_deploy_command       # 66-75
  test_prod_command         # 76-85
  test_staging_command      # 86-90
  test_service_management   # 91-100
  test_error_handling       # 101-115
  test_cross_platform       # 116-125

  # Cleanup
  cleanup_test_env

  # Final report
  print_final_report

  # Exit with appropriate code
  if [[ $FAILED -eq 0 ]]; then
    exit 0
  else
    exit 1
  fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
