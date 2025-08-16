#!/usr/bin/env bash

# base.sh - Base error handling framework

# Error registry - track all errors found
# On macOS, the default /bin/bash is 3.2 which lacks associative arrays.
# Provide a compatibility shim by detecting support and falling back to
# delimited-string maps if needed.

_HAS_ASSOC_ARRAYS=true
if ! (declare -A __test 2>/dev/null); then
  _HAS_ASSOC_ARRAYS=false
fi

if [[ "$_HAS_ASSOC_ARRAYS" == "true" ]]; then
  declare -A ERROR_REGISTRY
  declare -A ERROR_FIXES
  declare -A ERROR_ACTIONS
else
  # Emulate simple key->value maps using newline-delimited key=value pairs
  ERROR_REGISTRY=""
  ERROR_FIXES=""
  ERROR_ACTIONS=""
fi

# Error severity levels
readonly ERROR_CRITICAL=3 # Cannot continue
readonly ERROR_MAJOR=2    # Can try to fix
readonly ERROR_MINOR=1    # Can work around
readonly ERROR_WARNING=0  # Just informational

# Initialize error handling
init_error_handling() {
  if [[ "$_HAS_ASSOC_ARRAYS" == "true" ]]; then
    ERROR_REGISTRY=()
    ERROR_FIXES=()
    ERROR_ACTIONS=()
  else
    ERROR_REGISTRY=""
    ERROR_FIXES=""
    ERROR_ACTIONS=""
  fi
  export ERROR_COUNT=0
  export CRITICAL_ERRORS=0
  export FIXABLE_ERRORS=0
  export FIXED_ERRORS=0
}

# Register an error
register_error() {
  local error_code="$1"
  local error_msg="$2"
  local severity="${3:-$ERROR_MAJOR}"
  local fix_available="${4:-false}"
  local fix_function="${5:-}"
  if [[ "$_HAS_ASSOC_ARRAYS" == "true" ]]; then
    ERROR_REGISTRY["$error_code"]="$error_msg"
    ERROR_REGISTRY["${error_code}_severity"]="$severity"
    ERROR_REGISTRY["${error_code}_fixable"]="$fix_available"
    if [[ "$fix_available" == "true" ]] && [[ -n "$fix_function" ]]; then
      ERROR_FIXES["$error_code"]="$fix_function"
      ((FIXABLE_ERRORS++))
    fi
  else
    ERROR_REGISTRY+="$error_code=$error_msg
${error_code}_severity=$severity
${error_code}_fixable=$fix_available
"
    if [[ "$fix_available" == "true" ]] && [[ -n "$fix_function" ]]; then
      ERROR_FIXES+="$error_code=$fix_function
"
      ((FIXABLE_ERRORS++))
    fi
  fi

  ((ERROR_COUNT++))

  if [[ $severity -eq $ERROR_CRITICAL ]]; then
    ((CRITICAL_ERRORS++))
  fi
}

# Display error with context
display_error() {
  local error_code="$1"
  local context="${2:-}"
  local error_msg
  local severity
  local fixable
  if [[ "$_HAS_ASSOC_ARRAYS" == "true" ]]; then
    error_msg="${ERROR_REGISTRY[$error_code]}"
    severity="${ERROR_REGISTRY[${error_code}_severity]}"
    fixable="${ERROR_REGISTRY[${error_code}_fixable]}"
  else
    error_msg=$(echo "$ERROR_REGISTRY" | awk -F= -v k="$error_code" '$1==k{print substr($0,index($0,$2)) }' | head -1)
    severity=$(echo "$ERROR_REGISTRY" | awk -F= -v k="${error_code}_severity" '$1==k{print $2}' | head -1)
    fixable=$(echo "$ERROR_REGISTRY" | awk -F= -v k="${error_code}_fixable" '$1==k{print $2}' | head -1)
  fi

  case $severity in
  $ERROR_CRITICAL)
    log_error "[CRITICAL] $error_msg"
    ;;
  $ERROR_MAJOR)
    log_error "$error_msg"
    ;;
  $ERROR_MINOR)
    log_warning "$error_msg"
    ;;
  $ERROR_WARNING)
    log_info "$error_msg"
    ;;
  esac

  if [[ -n "$context" ]]; then
    echo "  Context: $context"
  fi

  if [[ "$fixable" == "true" ]]; then
    log_info "  ✓ This issue can be fixed automatically"
  fi
}

# Attempt to fix an error
attempt_fix() {
  local error_code="$1"
  local fix_function
  if [[ "$_HAS_ASSOC_ARRAYS" == "true" ]]; then
    fix_function="${ERROR_FIXES[$error_code]}"
  else
    fix_function=$(echo "$ERROR_FIXES" | awk -F= -v k="$error_code" '$1==k{print $2}' | head -1)
  fi

  if [[ -z "$fix_function" ]]; then
    log_debug "No fix available for error: $error_code"
    return 1
  fi

  log_info "Attempting to fix: ${ERROR_REGISTRY[$error_code]}"

  if $fix_function; then
    ((FIXED_ERRORS++))
    log_success "✓ Fixed successfully"
    return 0
  else
    log_error "✗ Fix failed"
    return 1
  fi
}

# Run all registered fixes
run_auto_fixes() {
  local interactive="${1:-true}"
  local fixed_count=0

  if [[ $FIXABLE_ERRORS -eq 0 ]]; then
    log_info "No auto-fixable errors found"
    return 0
  fi

  log_info "Found $FIXABLE_ERRORS fixable error(s)"

  if [[ "$interactive" == "true" ]]; then
    echo ""
    read -p "Attempt to fix automatically? [Y/n]: " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]] && [[ -n $REPLY ]]; then
      log_info "Skipping auto-fixes"
      return 0
    fi
  fi

  echo ""
  log_info "Running auto-fixes..."

  if [[ "$_HAS_ASSOC_ARRAYS" == "true" ]]; then
    for error_code in "${!ERROR_FIXES[@]}"; do
      if attempt_fix "$error_code"; then
        ((fixed_count++))
      fi
    done
  else
    # Iterate over newline-delimited list
    while IFS='=' read -r k v; do
      [[ -z "$k" ]] && continue
      if attempt_fix "$k"; then
        ((fixed_count++))
      fi
    done <<<"$ERROR_FIXES"
  fi

  echo ""
  if [[ $fixed_count -eq $FIXABLE_ERRORS ]]; then
    log_success "All fixable errors resolved!"
    return 0
  elif [[ $fixed_count -gt 0 ]]; then
    log_warning "Fixed $fixed_count of $FIXABLE_ERRORS errors"
    return 1
  else
    log_error "Could not fix any errors automatically"
    return 1
  fi
}

# Display error summary
show_error_summary() {
  if [[ $ERROR_COUNT -eq 0 ]]; then
    return 0
  fi

  echo ""
  log_header "Error Summary"

  echo "Total errors found: $ERROR_COUNT"

  if [[ $CRITICAL_ERRORS -gt 0 ]]; then
    echo "  • Critical errors: $CRITICAL_ERRORS (must be fixed)"
  fi

  if [[ $FIXABLE_ERRORS -gt 0 ]]; then
    echo "  • Fixable errors: $FIXABLE_ERRORS (can be fixed automatically)"
  fi

  if [[ $FIXED_ERRORS -gt 0 ]]; then
    echo "  • Fixed errors: $FIXED_ERRORS"
  fi

  local unfixed=$((ERROR_COUNT - FIXED_ERRORS))
  if [[ $unfixed -gt 0 ]]; then
    echo ""
    log_warning "$unfixed error(s) remain unresolved"
  fi
}

# Check if we should continue despite errors
should_continue() {
  if [[ $CRITICAL_ERRORS -gt 0 ]]; then
    log_error "Cannot continue due to critical errors"
    return 1
  fi

  local unfixed=$((ERROR_COUNT - FIXED_ERRORS))
  if [[ $unfixed -gt 0 ]]; then
    log_warning "There are $unfixed unresolved error(s)"
    read -p "Continue anyway? [y/N]: " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      return 1
    fi
  fi

  return 0
}

# Export functions
export -f init_error_handling
export -f register_error
export -f display_error
export -f attempt_fix
export -f run_auto_fixes
export -f show_error_summary
export -f should_continue
