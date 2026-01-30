#!/usr/bin/env bash
# Unit tests for email templates system
# Tests: Template creation, variable substitution, validation, multi-language, tenant isolation

set -euo pipefail

# Test framework colors (only define if not already set)
: ${GREEN:='\033[0;32m'}
: ${RED:='\033[0;31m'}
: ${YELLOW:='\033[1;33m'}
: ${NC:='\033[0m'}

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Setup test environment
setup_test_env() {
  export PROJECT_ROOT="/tmp/nself-email-test-$$"
  export BRAND_NAME="TestBrand"
  export BASE_DOMAIN="test.local"

  mkdir -p "$PROJECT_ROOT"

  # Get the absolute path to the script directory
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  # Source the email templates library
  if [[ -f "${script_dir}/../../lib/whitelabel/email-templates.sh" ]]; then
    source "${script_dir}/../../lib/whitelabel/email-templates.sh"
  else
    # Try from current directory (if running from project root)
    if [[ -f "src/lib/whitelabel/email-templates.sh" ]]; then
      source "src/lib/whitelabel/email-templates.sh"
    else
      printf "${RED}Failed to source email-templates.sh${NC}\n" >&2
      exit 1
    fi
  fi
}

# Cleanup test environment
cleanup_test_env() {
  rm -rf "$PROJECT_ROOT"
}

# Test assertion helpers
assert_equals() {
  local expected="$1"
  local actual="$2"
  local test_name="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$expected" == "$actual" ]]; then
    printf "${GREEN}✓${NC} %s\n" "$test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "${RED}✗${NC} %s\n" "$test_name"
    printf "  Expected: %s\n" "$expected"
    printf "  Actual: %s\n" "$actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local test_name="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if printf "%s" "$haystack" | grep -q "$needle"; then
    printf "${GREEN}✓${NC} %s\n" "$test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "${RED}✗${NC} %s\n" "$test_name"
    printf "  String does not contain: %s\n" "$needle"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

assert_file_exists() {
  local file="$1"
  local test_name="$2"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$file" ]]; then
    printf "${GREEN}✓${NC} %s\n" "$test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "${RED}✗${NC} %s\n" "$test_name"
    printf "  File not found: %s\n" "$file"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

assert_success() {
  local command="$1"
  local test_name="$2"

  TESTS_RUN=$((TESTS_RUN + 1))

  if eval "$command" >/dev/null 2>&1; then
    printf "${GREEN}✓${NC} %s\n" "$test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "${RED}✗${NC} %s\n" "$test_name"
    printf "  Command failed: %s\n" "$command"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

# ============================================================================
# Test Cases
# ============================================================================

test_html_escape() {
  local input='<script>alert("XSS")</script>'
  local expected='&lt;script&gt;alert(&quot;XSS&quot;)&lt;/script&gt;'
  local actual
  actual=$(html_escape "$input")

  assert_equals "$expected" "$actual" "HTML escape prevents XSS"
}

test_sanitize_variable_name() {
  # Test that special characters are stripped
  local input='USER$COMMAND_NAME'
  local expected='USERCOMMAND_NAME'
  local actual
  actual=$(sanitize_variable_name "$input")

  assert_equals "$expected" "$actual" "Variable name sanitization removes $ characters"

  # Test that only A-Z, 0-9, and _ remain
  local input2='USER-NAME@123!TEST'
  local expected2='USERNAME123TEST'
  local actual2
  actual2=$(sanitize_variable_name "$input2")

  assert_equals "$expected2" "$actual2" "Variable name sanitization removes dangerous characters"
}

test_template_initialization() {
  initialize_email_templates >/dev/null 2>&1

  assert_file_exists "$TEMPLATES_DIR/VARIABLES.md" "Variables reference created"
  assert_file_exists "${TEMPLATES_LANG_DIR}/en/welcome.html" "Welcome template HTML created"
  assert_file_exists "${TEMPLATES_LANG_DIR}/en/welcome.txt" "Welcome template TXT created"
  assert_file_exists "${TEMPLATES_LANG_DIR}/en/welcome.json" "Welcome template metadata created"
}

test_template_variable_substitution() {
  local template='Hello {{USER_NAME}}, welcome to {{BRAND_NAME}}!'
  local vars=("USER_NAME=John Doe" "BRAND_NAME=TestApp")
  local result
  result=$(substitute_template_variables "$template" "${vars[@]}")

  assert_contains "$result" "John Doe" "Variable USER_NAME substituted"
  assert_contains "$result" "TestApp" "Variable BRAND_NAME substituted"
}

test_template_validation() {
  initialize_email_templates >/dev/null 2>&1

  local template_file="${TEMPLATES_LANG_DIR}/en/welcome.html"
  assert_success "validate_template_content '$template_file'" "Valid template passes validation"

  # Create invalid template with dangerous code
  local bad_template="/tmp/bad-template-$$.html"
  printf '<!DOCTYPE html><html><body>$(whoami)</body></html>' >"$bad_template"

  TESTS_RUN=$((TESTS_RUN + 1))
  if validate_template_content "$bad_template" 2>/dev/null; then
    printf "${RED}✗${NC} Invalid template should fail validation\n"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  else
    printf "${GREEN}✓${NC} Invalid template correctly rejected\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  fi

  rm -f "$bad_template"
}

test_render_template() {
  initialize_email_templates >/dev/null 2>&1

  local vars=("USER_NAME=Test User" "APP_URL=https://test.com")
  local result
  result=$(render_template "welcome" "en" "html" "${vars[@]}")

  assert_contains "$result" "Test User" "User name rendered in template"
  assert_contains "$result" "TestBrand" "Brand name from env rendered"
}

test_multi_language_support() {
  initialize_email_templates >/dev/null 2>&1
  set_email_language "es" >/dev/null 2>&1

  assert_file_exists "${TEMPLATES_LANG_DIR}/es/welcome.html" "Spanish template created"
  assert_file_exists "${TEMPLATES_LANG_DIR}/es/password-reset.html" "Spanish password reset created"
}

test_tenant_isolation() {
  initialize_email_templates >/dev/null 2>&1

  local tenant_id="tenant123"
  initialize_tenant_templates "$tenant_id" >/dev/null 2>&1

  local tenant_dir
  tenant_dir=$(get_tenant_templates_dir "$tenant_id")

  assert_file_exists "$tenant_dir/languages/en/welcome.html" "Tenant template created"
  assert_file_exists "$tenant_dir/VARIABLES.md" "Tenant variables reference created"
}

test_tenant_directory_traversal_protection() {
  local bad_tenant_id="../../../etc/passwd"
  local tenant_dir
  tenant_dir=$(get_tenant_templates_dir "$bad_tenant_id" 2>/dev/null)

  # Should sanitize and not contain ../ or /etc
  TESTS_RUN=$((TESTS_RUN + 1))
  if [[ "$tenant_dir" =~ \.\. ]] || [[ "$tenant_dir" =~ /etc/ ]]; then
    printf "${RED}✗${NC} Tenant ID directory traversal not prevented\n"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  else
    printf "${GREEN}✓${NC} Tenant ID directory traversal prevented\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  fi
}

test_template_backup_on_edit() {
  initialize_email_templates >/dev/null 2>&1

  local template_file="${TEMPLATES_LANG_DIR}/en/welcome.html"
  local original_content
  original_content=$(cat "$template_file")

  # Create a fake editor script
  local fake_editor="/tmp/fake-editor-$$.sh"
  cat >"$fake_editor" <<'EOF'
#!/usr/bin/env bash
printf 'Modified content' > "$1"
EOF
  chmod +x "$fake_editor"

  # Set editor and edit
  export EDITOR="$fake_editor"

  # Edit should create backup
  edit_email_template "welcome" "en" "html" >/dev/null 2>&1

  # Check if backup exists
  TESTS_RUN=$((TESTS_RUN + 1))
  local backup_count
  backup_count=$(find "${TEMPLATES_LANG_DIR}/en/" -name "welcome.html.backup-*" 2>/dev/null | wc -l)

  if [[ "$backup_count" -gt 0 ]]; then
    printf "${GREEN}✓${NC} Template backup created on edit\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    printf "${RED}✗${NC} Template backup not created\n"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi

  # Cleanup
  unset EDITOR
  rm -f "$fake_editor"
}

test_custom_template_upload() {
  initialize_email_templates >/dev/null 2>&1

  # Create custom template
  local custom_html="/tmp/custom-$$.html"
  cat >"$custom_html" <<'EOF'
<!DOCTYPE html>
<html>
<body>
  <h1>Custom Template: {{TITLE}}</h1>
  <p>{{MESSAGE}}</p>
</body>
</html>
EOF

  upload_custom_template "custom-notification" "$custom_html" "" "en" >/dev/null 2>&1

  assert_file_exists "${TEMPLATES_LANG_DIR}/en/custom-notification.html" "Custom template uploaded"

  rm -f "$custom_html"
}

test_validate_all_templates() {
  initialize_email_templates >/dev/null 2>&1

  assert_success "validate_all_templates 'en'" "All default templates validate successfully"
}

test_template_export_import() {
  initialize_email_templates >/dev/null 2>&1

  local export_dir="/tmp/template-export-$$"
  export_all_templates "en" "$export_dir" >/dev/null 2>&1

  assert_file_exists "$export_dir/welcome.html" "Template exported"
  assert_file_exists "$export_dir/VARIABLES.md" "Variables reference exported"

  # Test import
  local new_project_root="/tmp/nself-import-test-$$"
  mkdir -p "$new_project_root"

  export PROJECT_ROOT="$new_project_root"
  initialize_email_templates >/dev/null 2>&1

  import_all_templates "$export_dir" "en" >/dev/null 2>&1

  assert_file_exists "${TEMPLATES_LANG_DIR}/en/welcome.html" "Template imported"

  rm -rf "$export_dir" "$new_project_root"
}

test_subject_line_rendering() {
  initialize_email_templates >/dev/null 2>&1

  local vars=("BRAND_NAME=MyApp")
  local subject
  subject=$(get_template_subject "welcome" "en" "${vars[@]}")

  assert_contains "$subject" "MyApp" "Subject line variable substitution works"
}

test_xss_prevention_in_variables() {
  initialize_email_templates >/dev/null 2>&1

  local vars=("USER_NAME=<script>alert('xss')</script>")
  local result
  result=$(render_template "welcome" "en" "html" "${vars[@]}")

  # Should contain escaped version, not raw script tag
  TESTS_RUN=$((TESTS_RUN + 1))
  if printf "%s" "$result" | grep -q "&lt;script&gt;" && ! printf "%s" "$result" | grep -q "<script>"; then
    printf "${GREEN}✓${NC} XSS prevention in template variables\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    printf "${RED}✗${NC} XSS not prevented in template variables\n"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# ============================================================================
# Run Tests
# ============================================================================

run_all_tests() {
  printf "\n${YELLOW}Running Email Templates Unit Tests${NC}\n"
  printf "%s\n\n" "$(printf '%.s=' {1..60})"

  setup_test_env

  # Security tests
  printf "\n${YELLOW}Security Tests:${NC}\n"
  test_html_escape
  test_sanitize_variable_name
  test_template_validation
  test_xss_prevention_in_variables
  test_tenant_directory_traversal_protection

  # Core functionality tests
  printf "\n${YELLOW}Core Functionality Tests:${NC}\n"
  test_template_initialization
  test_template_variable_substitution
  test_render_template
  test_subject_line_rendering

  # Multi-language tests
  printf "\n${YELLOW}Multi-Language Tests:${NC}\n"
  test_multi_language_support

  # Tenant isolation tests
  printf "\n${YELLOW}Tenant Isolation Tests:${NC}\n"
  test_tenant_isolation

  # Management tests
  printf "\n${YELLOW}Management Tests:${NC}\n"
  test_template_backup_on_edit
  test_custom_template_upload
  test_validate_all_templates
  test_template_export_import

  cleanup_test_env

  # Summary
  printf "\n%s\n" "$(printf '%.s=' {1..60})"
  printf "Test Results:\n"
  printf "  Total: %s\n" "$TESTS_RUN"
  printf "  ${GREEN}Passed: %s${NC}\n" "$TESTS_PASSED"
  printf "  ${RED}Failed: %s${NC}\n" "$TESTS_FAILED"
  printf "\n"

  if [[ $TESTS_FAILED -eq 0 ]]; then
    printf "${GREEN}All tests passed!${NC}\n\n"
    return 0
  else
    printf "${RED}Some tests failed!${NC}\n\n"
    return 1
  fi
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  run_all_tests
fi
