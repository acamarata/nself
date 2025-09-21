#!/usr/bin/env bash
# hosts.sh - Manage /etc/hosts entries for local development

# Check if a hosts entry exists
hosts_entry_exists() {
  local domain="$1"
  grep -q "127\.0\.0\.1.*[[:space:]]${domain}[[:space:]]*" /etc/hosts 2>/dev/null || \
  grep -q "127\.0\.0\.1.*[[:space:]]${domain}$" /etc/hosts 2>/dev/null
}

# Add entry to /etc/hosts (requires sudo)
add_hosts_entry() {
  local domain="$1"

  if hosts_entry_exists "$domain"; then
    return 0
  fi

  echo "127.0.0.1 $domain" | sudo tee -a /etc/hosts >/dev/null
  return $?
}

# Check and add all required hosts entries
ensure_hosts_entries() {
  local base_domain="${1:-localhost}"
  local project_name="${2:-nself}"
  local needs_update=false
  local missing_domains=()

  # Domains we need based on the base domain - initialize as array
  local required_domains
  required_domains=()

  if [[ "$base_domain" == "localhost" ]]; then
    # For localhost, we need these subdomains
    required_domains=(
      "api.localhost"
      "auth.localhost"
      "storage.localhost"
      "console.localhost"
      "functions.localhost"
      "${project_name}.localhost"
    )

    # Add frontend app domains if configured
    if [[ -n "${FRONTEND_APP_COUNT:-}" ]] && [[ "${FRONTEND_APP_COUNT}" -gt 0 ]]; then
      for ((i=1; i<=FRONTEND_APP_COUNT; i++)); do
        local app_domain_var="FRONTEND_APP_${i}_DOMAIN"
        local app_domain="${!app_domain_var:-}"
        if [[ -n "$app_domain" ]] && [[ "$app_domain" == *".localhost" ]]; then
          required_domains+=("$app_domain")
        fi
      done
    fi
  elif [[ "$base_domain" == "local.nself.org" ]]; then
    # local.nself.org uses wildcard DNS, no hosts entries needed
    return 0
  else
    # For custom domains, check if they resolve
    if ! host "$base_domain" >/dev/null 2>&1; then
      required_domains+=("$base_domain" "api.$base_domain" "auth.$base_domain" "storage.$base_domain")
    fi
  fi

  # Check which domains are missing
  if [[ ${#required_domains[@]} -gt 0 ]]; then
    for domain in "${required_domains[@]}"; do
      if ! hosts_entry_exists "$domain"; then
        missing_domains+=("$domain")
        needs_update=true
      fi
    done
  fi

  if [[ "$needs_update" == "false" ]]; then
    return 0
  fi

  # Inform user about missing entries
  echo ""
  echo "⚠️  Some domains need to be added to /etc/hosts for local development:"
  if [[ ${#missing_domains[@]} -gt 0 ]]; then
    for domain in "${missing_domains[@]}"; do
      echo "   - $domain"
    done
  fi
  echo ""

  # Check if we can use sudo without password
  local can_sudo_nopass=false
  if sudo -n true 2>/dev/null; then
    can_sudo_nopass=true
  fi

  # Check if we're in an interactive terminal
  local is_interactive=false
  if [[ -t 0 ]]; then
    is_interactive=true
  fi

  # Determine the best approach
  if [[ "$can_sudo_nopass" == "true" ]]; then
    # Can sudo without password, just do it
    echo "Adding entries to /etc/hosts..."

    # Build the entries string
    local entries=""
    if [[ ${#missing_domains[@]} -gt 0 ]]; then
      for domain in "${missing_domains[@]}"; do
        entries+="127.0.0.1 $domain\n"
      done
    fi

    # Add all entries at once
    if echo -e "$entries" | sudo tee -a /etc/hosts >/dev/null; then
      echo "✅ Successfully added ${#missing_domains[@]} entries to /etc/hosts"
      return 0
    else
      echo "❌ Failed to update /etc/hosts"
    fi
  elif [[ "$is_interactive" == "true" ]]; then
    # Interactive terminal, ask for permission
    echo "Would you like nself to add these entries automatically? (requires sudo)"
    echo -n "Add entries to /etc/hosts? [Y/n]: "
    read -r response

    if [[ -z "$response" ]] || [[ "$response" =~ ^[Yy] ]]; then
      echo ""
      echo "Adding entries to /etc/hosts (you may be prompted for your password)..."

      # Build the entries string
      local entries=""
      if [[ ${#missing_domains[@]} -gt 0 ]]; then
        for domain in "${missing_domains[@]}"; do
          entries+="127.0.0.1 $domain\n"
        done
      fi

      # Try to add entries with sudo
      if echo -e "$entries" | sudo tee -a /etc/hosts >/dev/null 2>&1; then
        echo "✅ Successfully added ${#missing_domains[@]} entries to /etc/hosts"
        return 0
      else
        echo "❌ Failed to update /etc/hosts"
      fi
    else
      echo ""
      echo "ℹ️  Skipped /etc/hosts update."
    fi
  else
    # Non-interactive or can't sudo
    echo "⚠️  Cannot automatically update /etc/hosts (non-interactive terminal)"
  fi

  # Show manual instructions if we couldn't update
  if [[ "$can_sudo_nopass" != "true" ]] || [[ "$is_interactive" != "true" ]]; then
    echo ""
    echo "You can manually add these lines to /etc/hosts:"
    if [[ ${#missing_domains[@]} -gt 0 ]]; then
      for domain in "${missing_domains[@]}"; do
        echo "127.0.0.1 $domain"
      done
    fi
    echo ""
    echo "Or run with sudo: sudo nself start"
    echo "Or use BASE_DOMAIN=local.nself.org which doesn't require /etc/hosts changes."
  fi

  return 0
}

# Remove nself entries from /etc/hosts (cleanup)
remove_hosts_entries() {
  local base_domain="${1:-localhost}"

  if [[ "$base_domain" == "local.nself.org" ]]; then
    return 0  # No entries to remove for wildcard domain
  fi

  echo "Removing nself entries from /etc/hosts (requires sudo)..."

  # Create a pattern to match our entries
  local pattern="127\.0\.0\.1.*\.\(localhost\|${base_domain}\)"

  # Remove matching lines
  sudo sed -i.bak "/$pattern/d" /etc/hosts

  echo "✅ Cleaned up /etc/hosts entries"
}

# Check if we can resolve a domain
can_resolve_domain() {
  local domain="$1"

  # First check if it's in /etc/hosts
  if hosts_entry_exists "$domain"; then
    return 0
  fi

  # Then check DNS resolution
  if host "$domain" >/dev/null 2>&1; then
    return 0
  fi

  # Check if it resolves to localhost via ping
  if ping -c 1 -W 1 "$domain" 2>/dev/null | grep -q "127.0.0.1"; then
    return 0
  fi

  return 1
}

# Export functions
export -f hosts_entry_exists
export -f add_hosts_entry
export -f ensure_hosts_entries
export -f remove_hosts_entries
export -f can_resolve_domain