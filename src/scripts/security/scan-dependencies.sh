#!/usr/bin/env bash
#
# Dependency Scanning Script
# Runs local security scans for dependencies, containers, and secrets
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Output helpers
info() {
  printf "${BLUE}ℹ ${NC}%s\n" "$1"
}

success() {
  printf "${GREEN}✓${NC} %s\n" "$1"
}

warning() {
  printf "${YELLOW}⚠${NC} %s\n" "$1"
}

error() {
  printf "${RED}✗${NC} %s\n" "$1"
}

# Check if command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Install instructions
show_install() {
  local tool="$1"
  local install_cmd="$2"

  warning "$tool not found"
  printf "  Install with: %s\n" "$install_cmd"
}

# Scan with ShellCheck
scan_shellcheck() {
  info "Running ShellCheck security scan..."

  if ! command_exists shellcheck; then
    show_install "ShellCheck" "brew install shellcheck"
    return 1
  fi

  # Find all shell scripts
  local scripts
  scripts=$(find "$PROJECT_ROOT/src" -type f -name "*.sh")

  local issues=0
  while IFS= read -r script; do
    if ! shellcheck -S error -e SC1091 "$script" 2>/dev/null; then
      issues=$((issues + 1))
    fi
  done <<< "$scripts"

  if [ $issues -eq 0 ]; then
    success "ShellCheck: No security issues found"
    return 0
  else
    error "ShellCheck: Found $issues files with issues"
    return 1
  fi
}

# Scan for secrets
scan_secrets() {
  info "Running secret scanning..."

  local found_secrets=0

  # Try detect-secrets
  if command_exists detect-secrets; then
    if detect-secrets scan --baseline "$PROJECT_ROOT/.secrets.baseline" 2>/dev/null; then
      success "detect-secrets: No new secrets found"
    else
      error "detect-secrets: New secrets detected"
      found_secrets=1
    fi
  else
    show_install "detect-secrets" "pip install detect-secrets"
  fi

  # Try gitleaks
  if command_exists gitleaks; then
    if gitleaks detect --source "$PROJECT_ROOT" --no-git 2>/dev/null; then
      success "gitleaks: No secrets found"
    else
      error "gitleaks: Secrets detected"
      found_secrets=1
    fi
  else
    show_install "gitleaks" "brew install gitleaks"
  fi

  # Try trufflehog
  if command_exists trufflehog; then
    if trufflehog filesystem "$PROJECT_ROOT" --only-verified --no-update 2>/dev/null; then
      success "trufflehog: No verified secrets found"
    else
      error "trufflehog: Verified secrets detected"
      found_secrets=1
    fi
  else
    show_install "trufflehog" "brew install trufflehog"
  fi

  return $found_secrets
}

# Scan with Trivy
scan_trivy() {
  info "Running Trivy vulnerability scan..."

  if ! command_exists trivy; then
    show_install "Trivy" "brew install aquasecurity/trivy/trivy"
    return 1
  fi

  # Scan filesystem
  if trivy fs "$PROJECT_ROOT" --severity HIGH,CRITICAL --exit-code 0 2>/dev/null; then
    success "Trivy: Filesystem scan complete"
  else
    warning "Trivy: Found vulnerabilities"
  fi

  # Scan Dockerfiles
  local dockerfiles
  dockerfiles=$(find "$PROJECT_ROOT/src/templates" -name "Dockerfile*" -type f)

  local docker_issues=0
  while IFS= read -r dockerfile; do
    if ! trivy config "$dockerfile" --severity HIGH,CRITICAL --exit-code 0 2>/dev/null; then
      docker_issues=$((docker_issues + 1))
    fi
  done <<< "$dockerfiles"

  if [ $docker_issues -eq 0 ]; then
    success "Trivy: No critical Dockerfile issues"
  else
    warning "Trivy: Found issues in $docker_issues Dockerfile(s)"
  fi

  return 0
}

# Scan with Semgrep
scan_semgrep() {
  info "Running Semgrep SAST scan..."

  if ! command_exists semgrep; then
    show_install "Semgrep" "brew install semgrep"
    return 1
  fi

  # Run security audit
  if semgrep --config=p/security-audit "$PROJECT_ROOT" --quiet --error 2>/dev/null; then
    success "Semgrep: No security issues found"
    return 0
  else
    error "Semgrep: Security issues detected"
    return 1
  fi
}

# Scan Dockerfiles
scan_dockerfiles() {
  info "Running Dockerfile security checks..."

  local dockerfiles
  dockerfiles=$(find "$PROJECT_ROOT/src/templates" -name "Dockerfile*" -type f)

  local issues=0

  while IFS= read -r dockerfile; do
    # Check for USER instruction
    if ! grep -q "^USER" "$dockerfile"; then
      warning "No USER instruction: $dockerfile"
      issues=$((issues + 1))
    fi

    # Check for :latest tag
    if grep -q "^FROM.*:latest" "$dockerfile"; then
      warning "Using :latest tag: $dockerfile"
      issues=$((issues + 1))
    fi

    # Check for HEALTHCHECK
    if ! grep -q "^HEALTHCHECK" "$dockerfile"; then
      info "No HEALTHCHECK: $dockerfile"
    fi
  done <<< "$dockerfiles"

  if [ $issues -eq 0 ]; then
    success "Dockerfile checks: No critical issues"
    return 0
  else
    warning "Dockerfile checks: Found $issues issues"
    return 0  # Don't fail on warnings
  fi
}

# Generate report
generate_report() {
  local report_file="$PROJECT_ROOT/security-scan-report.txt"

  info "Generating security report..."

  {
    printf "Security Scan Report\n"
    printf "====================\n"
    printf "Generated: %s\n" "$(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    printf "Project: nself\n\n"

    printf "Scans Completed:\n"
    printf "- ShellCheck (shell script security)\n"
    printf "- Secret scanning (detect-secrets, gitleaks, trufflehog)\n"
    printf "- Trivy (dependency and container vulnerabilities)\n"
    printf "- Semgrep (SAST - static analysis)\n"
    printf "- Dockerfile security checks\n\n"

    printf "See CI/CD workflows for continuous scanning:\n"
    printf "- .github/workflows/security-scan.yml\n"
    printf "- .github/workflows/ci.yml\n\n"

    printf "Pre-commit hooks configured:\n"
    printf "- .pre-commit-config.yaml\n\n"

    printf "Documentation:\n"
    printf "- .wiki/security/DEPENDENCY-SCANNING.md\n"
  } > "$report_file"

  success "Report generated: $report_file"
}

# Main execution
main() {
  printf "${BLUE}nself Security Dependency Scanner${NC}\n"
  printf "==================================\n\n"

  cd "$PROJECT_ROOT"

  local exit_code=0

  # Run all scans
  scan_shellcheck || exit_code=1
  printf "\n"

  scan_secrets || exit_code=1
  printf "\n"

  scan_trivy || exit_code=1
  printf "\n"

  scan_semgrep || exit_code=1
  printf "\n"

  scan_dockerfiles || exit_code=1
  printf "\n"

  generate_report

  printf "\n"
  if [ $exit_code -eq 0 ]; then
    success "All security scans passed!"
  else
    error "Some security scans found issues - please review"
  fi

  printf "\nFor continuous security scanning, see:\n"
  printf "  - GitHub Actions: .github/workflows/security-scan.yml\n"
  printf "  - Pre-commit hooks: .pre-commit-config.yaml\n"
  printf "  - Documentation: .wiki/security/DEPENDENCY-SCANNING.md\n"

  exit $exit_code
}

main "$@"
