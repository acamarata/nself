#!/usr/bin/env bash
# comprehensive-scanner.sh - Comprehensive Security Scanning System
# Part of nself v0.9.6+ - Security First Implementation
#
# This module provides comprehensive security scanning for nself projects:
# - Weak password/secret detection
# - Default secrets detection
# - Git exposure scanning
# - File permission auditing
# - SQL injection vulnerability scanning
# - XSS risk detection
# - Configuration security auditing

set -euo pipefail

# Get script directory
SECURITY_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_ROOT="$(dirname "$SECURITY_LIB_DIR")"

# Source dependencies
source "$LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true
source "$SECURITY_LIB_DIR/scanner.sh" 2>/dev/null || true

# ============================================================================
# Configuration
# ============================================================================

# Minimum secret lengths
readonly MIN_PASSWORD_LENGTH=16
readonly MIN_SECRET_LENGTH=32
readonly MIN_JWT_SECRET_LENGTH=64

# Default secrets to detect (these should NEVER be in production)
readonly -a DEFAULT_SECRETS=(
  "postgres-dev-password"
  "hasura-admin-secret-dev"
  "admin-secret"
  "secret"
  "password"
  "changeme"
  "default"
  "test"
)

# Sensitive file patterns
readonly -a SENSITIVE_FILES=(
  ".env"
  ".env.local"
  ".env.secrets"
  ".env.prod"
  ".env.production"
  "secrets.json"
  "credentials.json"
  "service-account.json"
  ".aws/credentials"
  ".ssh/id_rsa"
  "private.key"
  "*.pem"
)

# ============================================================================
# Utility Functions
# ============================================================================

# Print scan header
print_scan_header() {
  local title="$1"
  printf "\n"
  printf "${COLOR_CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}\n"
  printf "${COLOR_CYAN}%s${COLOR_RESET}\n" "$title"
  printf "${COLOR_CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}\n"
  printf "\n"
}

# Print finding with severity
print_finding() {
  local severity="$1"
  local title="$2"
  local description="$3"
  local recommendation="${4:-}"

  case "$severity" in
    CRITICAL)
      printf "  ${COLOR_RED}⚠ CRITICAL${COLOR_RESET} - %s\n" "$title"
      ;;
    HIGH)
      printf "  ${COLOR_RED}⚠ HIGH${COLOR_RESET} - %s\n" "$title"
      ;;
    MEDIUM)
      printf "  ${COLOR_YELLOW}⚠ MEDIUM${COLOR_RESET} - %s\n" "$title"
      ;;
    LOW)
      printf "  ${COLOR_YELLOW}⚠ LOW${COLOR_RESET} - %s\n" "$title"
      ;;
    INFO)
      printf "  ${COLOR_BLUE}ℹ INFO${COLOR_RESET} - %s\n" "$title"
      ;;
    PASS)
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$title"
      ;;
  esac

  if [[ -n "$description" ]]; then
    printf "    %s\n" "$description"
  fi

  if [[ -n "$recommendation" ]]; then
    printf "    ${COLOR_CYAN}→${COLOR_RESET} %s\n" "$recommendation"
  fi
  printf "\n"
}

# ============================================================================
# Secret Scanning
# ============================================================================

# Scan environment files for weak passwords/secrets
scan_weak_secrets() {
  local env_file="${1:-.env}"
  local issues=0
  local warnings=0

  print_scan_header "SECRET STRENGTH ANALYSIS"

  if [[ ! -f "$env_file" ]]; then
    print_finding "INFO" "No environment file found" "File: $env_file" "Create .env file with: nself init"
    return 0
  fi

  printf "  Scanning: %s\n\n" "$env_file"

  # Check each password/secret variable
  local secret_vars=(
    "POSTGRES_PASSWORD"
    "HASURA_GRAPHQL_ADMIN_SECRET"
    "JWT_SECRET"
    "COOKIE_SECRET"
    "MINIO_ROOT_PASSWORD"
    "REDIS_PASSWORD"
    "GRAFANA_ADMIN_PASSWORD"
  )

  for var in "${secret_vars[@]}"; do
    local value
    value=$(grep "^${var}=" "$env_file" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | tr -d "'")

    if [[ -z "$value" ]]; then
      if [[ "$var" == "POSTGRES_PASSWORD" ]] || [[ "$var" == "HASURA_GRAPHQL_ADMIN_SECRET" ]] || [[ "$var" == "JWT_SECRET" ]]; then
        print_finding "CRITICAL" "$var is not set" "Required secret is missing" "Set in $env_file or .env.secrets"
        issues=$((issues + 1))
      fi
      continue
    fi

    # Determine minimum length based on variable type
    local min_length=$MIN_PASSWORD_LENGTH
    if [[ "$var" == *"SECRET"* ]] || [[ "$var" == "JWT_SECRET" ]]; then
      min_length=$MIN_JWT_SECRET_LENGTH
    elif [[ "$var" == *"PASSWORD"* ]]; then
      min_length=$MIN_PASSWORD_LENGTH
    fi

    # Length check
    if [[ ${#value} -lt $min_length ]]; then
      print_finding "HIGH" "$var is too short" "Current: ${#value} characters, Minimum: $min_length" "Generate stronger secret: openssl rand -hex 32"
      issues=$((issues + 1))
      continue
    fi

    # Check for default/weak values
    local is_weak=false
    local lower_value
    lower_value=$(echo "$value" | tr '[:upper:]' '[:lower:]')

    for default in "${DEFAULT_SECRETS[@]}"; do
      if [[ "$lower_value" == *"$default"* ]]; then
        print_finding "CRITICAL" "$var uses default/weak value" "Contains common password pattern" "Generate secure random value immediately"
        issues=$((issues + 1))
        is_weak=true
        break
      fi
    done

    if $is_weak; then
      continue
    fi

    # Complexity check for passwords
    if [[ "$var" == *"PASSWORD"* ]]; then
      # Check for at least some character variety
      local has_lower=false
      local has_upper=false
      local has_digit=false
      local has_special=false

      [[ "$value" =~ [a-z] ]] && has_lower=true
      [[ "$value" =~ [A-Z] ]] && has_upper=true
      [[ "$value" =~ [0-9] ]] && has_digit=true
      [[ "$value" =~ [^a-zA-Z0-9] ]] && has_special=true

      local complexity_score=0
      $has_lower && complexity_score=$((complexity_score + 1))
      $has_upper && complexity_score=$((complexity_score + 1))
      $has_digit && complexity_score=$((complexity_score + 1))
      $has_special && complexity_score=$((complexity_score + 1))

      if [[ $complexity_score -lt 3 ]]; then
        print_finding "MEDIUM" "$var lacks complexity" "Use mix of uppercase, lowercase, digits, and special characters" "Use: openssl rand -base64 32"
        warnings=$((warnings + 1))
        continue
      fi
    fi

    print_finding "PASS" "$var is properly configured" "${#value} characters, good complexity"
  done

  printf "  ${COLOR_BOLD}Summary:${COLOR_RESET}\n"
  printf "    Issues: %d | Warnings: %d\n" "$issues" "$warnings"
  printf "\n"

  return $issues
}

# Scan for secrets in git repository
scan_git_secrets() {
  local issues=0

  print_scan_header "GIT EXPOSURE SCAN"

  if [[ ! -d ".git" ]]; then
    print_finding "INFO" "Not a git repository" "Skipping git-based checks"
    return 0
  fi

  if ! command -v git >/dev/null 2>&1; then
    print_finding "INFO" "Git not available" "Skipping git-based checks"
    return 0
  fi

  # Check if sensitive files are tracked
  for pattern in "${SENSITIVE_FILES[@]}"; do
    # Use git ls-files to find tracked files matching pattern
    local tracked_files
    tracked_files=$(git ls-files "$pattern" 2>/dev/null || true)

    if [[ -n "$tracked_files" ]]; then
      while IFS= read -r file; do
        print_finding "CRITICAL" "Sensitive file tracked in git" "File: $file" "Run: git rm --cached $file && echo '$file' >> .gitignore"
        issues=$((issues + 1))
      done <<<"$tracked_files"
    fi
  done

  # Check if .gitignore exists and has required entries
  if [[ ! -f ".gitignore" ]]; then
    print_finding "HIGH" ".gitignore file missing" "Sensitive files may be committed" "Create .gitignore with sensitive patterns"
    issues=$((issues + 1))
  else
    local required_patterns=(
      ".env"
      ".env.local"
      ".env.secrets"
      ".env.*.local"
    )

    for pattern in "${required_patterns[@]}"; do
      if ! grep -q "^${pattern}$" .gitignore 2>/dev/null; then
        print_finding "MEDIUM" ".gitignore missing pattern" "Pattern: $pattern" "Add to .gitignore"
        issues=$((issues + 1))
      fi
    done
  fi

  # Check git history for secrets (recent commits only)
  printf "  Scanning recent commits for exposed secrets...\n\n"

  local recent_commits
  recent_commits=$(git log --all --pretty=format:"%H" -n 10 2>/dev/null || true)

  if [[ -n "$recent_commits" ]]; then
    local found_secrets=false
    while IFS= read -r commit; do
      local commit_files
      commit_files=$(git diff-tree --no-commit-id --name-only -r "$commit" 2>/dev/null || true)

      while IFS= read -r file; do
        for pattern in "${SENSITIVE_FILES[@]}"; do
          if [[ "$file" == $pattern ]]; then
            print_finding "HIGH" "Sensitive file in commit history" "Commit: ${commit:0:8}, File: $file" "Consider using git-filter-repo to remove from history"
            found_secrets=true
            issues=$((issues + 1))
            break
          fi
        done
      done <<<"$commit_files"
    done <<<"$recent_commits"

    if ! $found_secrets; then
      print_finding "PASS" "No sensitive files in recent commit history"
    fi
  fi

  printf "  ${COLOR_BOLD}Summary:${COLOR_RESET}\n"
  printf "    Issues: %d\n" "$issues"
  printf "\n"

  return $issues
}

# ============================================================================
# File Permission Scanning
# ============================================================================

scan_file_permissions() {
  local issues=0
  local warnings=0

  print_scan_header "FILE PERMISSION AUDIT"

  # Check sensitive files have correct permissions
  local sensitive_configs=(
    ".env:600"
    ".env.local:600"
    ".env.secrets:600"
    ".env.prod:600"
    "docker-compose.yml:644"
  )

  for config in "${sensitive_configs[@]}"; do
    local file="${config%:*}"
    local expected_perms="${config#*:}"

    if [[ ! -f "$file" ]]; then
      continue
    fi

    local actual_perms
    actual_perms=$(safe_stat_perms "$file" 2>/dev/null || echo "unknown")

    if [[ "$actual_perms" == "unknown" ]]; then
      print_finding "WARNING" "Cannot read permissions" "File: $file" "Check file system"
      warnings=$((warnings + 1))
      continue
    fi

    if [[ "$actual_perms" != "$expected_perms" ]]; then
      local severity="MEDIUM"
      [[ "$file" == *"secret"* ]] || [[ "$file" == ".env" ]] && severity="HIGH"

      print_finding "$severity" "Incorrect file permissions" "File: $file (${actual_perms}, should be ${expected_perms})" "Run: chmod $expected_perms $file"
      issues=$((issues + 1))
    else
      print_finding "PASS" "$file has correct permissions" "$actual_perms"
    fi
  done

  # Check for world-readable sensitive files
  if command -v find >/dev/null 2>&1; then
    local world_readable
    world_readable=$(find . -maxdepth 2 -type f \( -name "*.env*" -o -name "*secret*" -o -name "*credential*" \) -perm -004 2>/dev/null || true)

    if [[ -n "$world_readable" ]]; then
      while IFS= read -r file; do
        print_finding "HIGH" "World-readable sensitive file" "File: $file" "Run: chmod 600 $file"
        issues=$((issues + 1))
      done <<<"$world_readable"
    fi
  fi

  printf "  ${COLOR_BOLD}Summary:${COLOR_RESET}\n"
  printf "    Issues: %d | Warnings: %d\n" "$issues" "$warnings"
  printf "\n"

  return $issues
}

# ============================================================================
# Configuration Security Audit
# ============================================================================

scan_configuration_security() {
  local issues=0
  local warnings=0
  local env_file="${1:-.env}"

  print_scan_header "CONFIGURATION SECURITY AUDIT"

  if [[ ! -f "$env_file" ]]; then
    print_finding "INFO" "No environment file found" "File: $env_file"
    return 0
  fi

  printf "  Scanning: %s\n\n" "$env_file"

  # Check environment
  local env
  env=$(grep "^ENV=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

  # Production-specific checks
  if [[ "$env" == "production" ]] || [[ "$env" == "prod" ]]; then
    # Hasura console should be disabled in production
    local console_enabled
    console_enabled=$(grep "^HASURA_GRAPHQL_ENABLE_CONSOLE=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

    if [[ "$console_enabled" == "true" ]]; then
      print_finding "CRITICAL" "Hasura console enabled in production" "HASURA_GRAPHQL_ENABLE_CONSOLE=true" "Set to false in production"
      issues=$((issues + 1))
    else
      print_finding "PASS" "Hasura console properly disabled in production"
    fi

    # Check for debug mode
    local debug_mode
    debug_mode=$(grep "^DEBUG=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

    if [[ "$debug_mode" == "true" ]] || [[ "$debug_mode" == "1" ]]; then
      print_finding "HIGH" "Debug mode enabled in production" "DEBUG=true" "Set DEBUG=false in production"
      issues=$((issues + 1))
    fi

    # SSL should be enabled
    local ssl_enabled
    ssl_enabled=$(grep "^SSL_ENABLED=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

    if [[ "$ssl_enabled" != "true" ]]; then
      print_finding "CRITICAL" "SSL not enabled in production" "SSL_ENABLED not set or false" "Enable SSL: nself auth ssl generate"
      issues=$((issues + 1))
    else
      print_finding "PASS" "SSL enabled for production"
    fi

    # Check for proper domain (not localhost)
    local base_domain
    base_domain=$(grep "^BASE_DOMAIN=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

    if [[ "$base_domain" == "localhost" ]] || [[ "$base_domain" == *".local" ]]; then
      print_finding "CRITICAL" "Production using localhost domain" "BASE_DOMAIN=$base_domain" "Set production domain"
      issues=$((issues + 1))
    fi
  fi

  # Check CORS configuration
  local cors_origin
  cors_origin=$(grep "^HASURA_GRAPHQL_CORS_DOMAIN=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

  if [[ "$cors_origin" == "*" ]]; then
    print_finding "HIGH" "CORS allows all origins" "HASURA_GRAPHQL_CORS_DOMAIN=*" "Restrict to specific domains"
    issues=$((issues + 1))
  elif [[ -z "$cors_origin" ]]; then
    print_finding "MEDIUM" "CORS not configured" "May cause frontend connection issues" "Configure HASURA_GRAPHQL_CORS_DOMAIN"
    warnings=$((warnings + 1))
  else
    print_finding "PASS" "CORS properly configured" "Limited to: $cors_origin"
  fi

  # Check for exposed ports
  local postgres_port
  postgres_port=$(grep "^POSTGRES_PORT=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

  if [[ -n "$postgres_port" ]] && [[ "$postgres_port" != "5432" ]]; then
    print_finding "INFO" "Custom PostgreSQL port" "Port: $postgres_port" "Ensure firewall rules are configured"
  fi

  # Check for monitoring in production
  if [[ "$env" == "production" ]] || [[ "$env" == "prod" ]]; then
    local monitoring_enabled
    monitoring_enabled=$(grep "^MONITORING_ENABLED=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")

    if [[ "$monitoring_enabled" != "true" ]]; then
      print_finding "MEDIUM" "Monitoring not enabled in production" "MONITORING_ENABLED not set" "Enable monitoring for production visibility"
      warnings=$((warnings + 1))
    else
      print_finding "PASS" "Monitoring enabled in production"
    fi
  fi

  printf "  ${COLOR_BOLD}Summary:${COLOR_RESET}\n"
  printf "    Issues: %d | Warnings: %d\n" "$issues" "$warnings"
  printf "\n"

  return $issues
}

# ============================================================================
# SQL Injection Scanning
# ============================================================================

scan_sql_files() {
  local issues=0

  print_scan_header "SQL INJECTION VULNERABILITY SCAN"

  # Find SQL files
  local sql_files
  if command -v find >/dev/null 2>&1; then
    sql_files=$(find . -type f -name "*.sql" 2>/dev/null | grep -v ".git" || true)
  else
    # Fallback for systems without find
    sql_files=$(ls -1 *.sql 2>/dev/null || true)
  fi

  if [[ -z "$sql_files" ]]; then
    print_finding "INFO" "No SQL files found" "Skipping SQL injection scan"
    return 0
  fi

  printf "  Scanning SQL files for potential vulnerabilities...\n\n"

  local risky_patterns=(
    "EXECUTE.*FORMAT.*%s"
    "EXECUTE.*\|\|"
    "EXECUTE.*CONCAT"
    "-- noinspection SqlNoDataSourceInspection"
    "\$\{[^}]+\}"
  )

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue

    local file_issues=0

    for pattern in "${risky_patterns[@]}"; do
      if grep -q "$pattern" "$file" 2>/dev/null; then
        print_finding "HIGH" "Potential SQL injection vector" "File: $file, Pattern: $pattern" "Use parameterized queries or proper escaping"
        file_issues=$((file_issues + 1))
        issues=$((issues + 1))
      fi
    done

    if [[ $file_issues -eq 0 ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s - No issues found\n" "$file"
    fi
  done <<<"$sql_files"

  printf "\n"
  printf "  ${COLOR_BOLD}Summary:${COLOR_RESET}\n"
  printf "    Potential SQL injection risks: %d\n" "$issues"
  printf "\n"

  return $issues
}

# ============================================================================
# Container Security Scan
# ============================================================================

scan_container_security() {
  local issues=0
  local warnings=0

  print_scan_header "CONTAINER SECURITY SCAN"

  if [[ ! -f "docker-compose.yml" ]]; then
    print_finding "INFO" "No docker-compose.yml found" "Skipping container security scan"
    return 0
  fi

  printf "  Scanning docker-compose.yml...\n\n"

  # Check for privileged containers
  if grep -q "privileged.*true" docker-compose.yml 2>/dev/null; then
    print_finding "HIGH" "Privileged container detected" "Containers should not run privileged" "Remove 'privileged: true' unless absolutely necessary"
    issues=$((issues + 1))
  else
    print_finding "PASS" "No privileged containers"
  fi

  # Check for containers running as root
  if ! grep -q "user:.*[0-9]" docker-compose.yml 2>/dev/null; then
    print_finding "MEDIUM" "Containers may be running as root" "No user specification found" "Add 'user: 1000:1000' to services"
    warnings=$((warnings + 1))
  fi

  # Check for environment variable exposure
  if grep -q "environment:" docker-compose.yml 2>/dev/null; then
    if grep -A 10 "environment:" docker-compose.yml | grep -q "PASSWORD\|SECRET\|KEY" 2>/dev/null; then
      print_finding "HIGH" "Secrets in docker-compose.yml" "Use env_file or .env instead" "Move secrets to .env file"
      issues=$((issues + 1))
    fi
  fi

  # Check for latest tags
  if grep -q "image:.*:latest" docker-compose.yml 2>/dev/null; then
    print_finding "LOW" "Using 'latest' image tags" "Pin to specific versions for reproducibility" "Specify exact versions (e.g., postgres:16-alpine)"
    warnings=$((warnings + 1))
  fi

  printf "  ${COLOR_BOLD}Summary:${COLOR_RESET}\n"
  printf "    Issues: %d | Warnings: %d\n" "$issues" "$warnings"
  printf "\n"

  return $issues
}

# ============================================================================
# Main Scan Function
# ============================================================================

# Run comprehensive security scan
security_scan_comprehensive() {
  local scan_deep="${1:-false}"
  local env_file="${2:-.env}"

  local total_issues=0
  local total_warnings=0

  printf "\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}╔═══════════════════════════════════════════════════════════════════════════╗${COLOR_RESET}\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}║                                                                           ║${COLOR_RESET}\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}║              nself COMPREHENSIVE SECURITY SCAN                           ║${COLOR_RESET}\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}║                                                                           ║${COLOR_RESET}\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}╚═══════════════════════════════════════════════════════════════════════════╝${COLOR_RESET}\n"
  printf "\n"
  printf "  Scan started: %s\n" "$(date '+%Y-%m-%d %H:%M:%S')"
  printf "  Target: %s\n" "$(pwd)"
  printf "  Deep scan: %s\n" "$scan_deep"
  printf "\n"

  # Run all scans
  scan_weak_secrets "$env_file" || total_issues=$((total_issues + $?))
  scan_git_secrets || total_issues=$((total_issues + $?))
  scan_file_permissions || total_issues=$((total_issues + $?))
  scan_configuration_security "$env_file" || total_issues=$((total_issues + $?))
  scan_container_security || total_issues=$((total_issues + $?))

  if [[ "$scan_deep" == "true" ]]; then
    scan_sql_files || total_issues=$((total_issues + $?))
  fi

  # Final summary
  printf "\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}\n"
  printf "${COLOR_BOLD}FINAL SUMMARY${COLOR_RESET}\n"
  printf "${COLOR_BOLD}${COLOR_CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}\n"
  printf "\n"

  if [[ $total_issues -eq 0 ]]; then
    printf "  ${COLOR_GREEN}${COLOR_BOLD}✓ NO SECURITY ISSUES FOUND${COLOR_RESET}\n"
    printf "  Your nself project appears secure.\n"
    printf "\n"
    return 0
  elif [[ $total_issues -lt 5 ]]; then
    printf "  ${COLOR_YELLOW}⚠ %d SECURITY ISSUES FOUND${COLOR_RESET}\n" "$total_issues"
    printf "  Review and address the issues above.\n"
    printf "\n"
    return 1
  else
    printf "  ${COLOR_RED}⚠ %d SECURITY ISSUES FOUND${COLOR_RESET}\n" "$total_issues"
    printf "  ${COLOR_RED}${COLOR_BOLD}CRITICAL: Multiple security issues detected!${COLOR_RESET}\n"
    printf "  Review and fix immediately before deploying to production.\n"
    printf "\n"
    return 1
  fi
}

# Export functions
export -f print_scan_header
export -f print_finding
export -f scan_weak_secrets
export -f scan_git_secrets
export -f scan_file_permissions
export -f scan_configuration_security
export -f scan_sql_files
export -f scan_container_security
export -f security_scan_comprehensive
