#!/usr/bin/env bash
# secrets-gen.sh - Strong Secret Generation for nself init
# Part of nself v0.9.6+ - Security First Implementation
#
# This module generates strong random secrets during init
# and replaces default weak values with secure ones

set -euo pipefail

# Get module directory
INIT_MODULE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECURITY_LIB_DIR="$INIT_MODULE_DIR/../security"

# Source secret generation functions
if [[ -f "$SECURITY_LIB_DIR/secrets.sh" ]]; then
  source "$SECURITY_LIB_DIR/secrets.sh"
fi

# ============================================================================
# Secret Generation for Init
# ============================================================================

# Generate strong random secret (fallback if secrets.sh not available)
generate_random_secret() {
  local length="${1:-32}"
  local type="${2:-hex}"

  # Try to use secrets::generate_random if available
  if command -v secrets::generate_random >/dev/null 2>&1; then
    secrets::generate_random "$length" "$type"
    return $?
  fi

  # Fallback implementation
  case "$type" in
    hex)
      if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex "$((length / 2))" | head -c "$length"
      elif [[ -f /dev/urandom ]]; then
        head -c "$((length / 2))" /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c "$length"
      else
        # Last resort: date + process ID
        printf "%s" "$(date +%s%N)$$" | sha256sum | cut -c1-"$length"
      fi
      ;;
    alphanumeric)
      if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 "$((length * 2))" | tr -dc 'a-zA-Z0-9' | head -c "$length"
      elif [[ -f /dev/urandom ]]; then
        head -c "$((length * 2))" /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c "$length"
      else
        # Fallback
        printf "%s%s" "$(date +%s%N)" "$$" | sha256sum | tr -dc 'a-zA-Z0-9' | head -c "$length"
      fi
      ;;
    *)
      # Default to hex
      generate_random_secret "$length" "hex"
      ;;
  esac
}

# Replace default secrets in environment file
replace_default_secrets_in_file() {
  local env_file="$1"
  local skip_replacement="${2:-false}"

  if [[ ! -f "$env_file" ]]; then
    return 0
  fi

  # Skip replacement if flag is set (for --keep-defaults)
  if [[ "$skip_replacement" == "true" ]]; then
    return 0
  fi

  local temp_file
  temp_file=$(mktemp)

  # Define default secrets to replace
  local -A secret_replacements=(
    ["POSTGRES_PASSWORD"]="postgres-dev-password"
    ["HASURA_GRAPHQL_ADMIN_SECRET"]="hasura-admin-secret-dev"
    ["HASURA_JWT_KEY"]="development-secret-key-minimum-32-characters-long"
    ["MINIO_ROOT_PASSWORD"]="minioadmin"
    ["S3_SECRET_KEY"]="storage-secret-key-dev"
    ["S3_ACCESS_KEY"]="storage-access-key-dev"
  )

  # Track if we made any replacements
  local replaced=false

  # Process each line
  while IFS= read -r line || [[ -n "$line" ]]; do
    local modified_line="$line"
    local line_modified=false

    # Check each secret pattern
    for var_name in "${!secret_replacements[@]}"; do
      local default_value="${secret_replacements[$var_name]}"

      # Check if this line sets this variable with the default value
      if [[ "$line" =~ ^[[:space:]]*${var_name}=[[:space:]]*${default_value}[[:space:]]*$ ]]; then
        # Generate strong replacement
        local new_value

        case "$var_name" in
          *PASSWORD*)
            # Passwords: 32 char alphanumeric
            new_value=$(generate_random_secret 32 alphanumeric)
            ;;
          *SECRET*|*KEY*)
            # Secrets/keys: 64 char hex
            new_value=$(generate_random_secret 64 hex)
            ;;
          *)
            # Default: 32 char hex
            new_value=$(generate_random_secret 32 hex)
            ;;
        esac

        modified_line="${var_name}=${new_value}"
        line_modified=true
        replaced=true
        break
      fi
    done

    printf "%s\n" "$modified_line" >>"$temp_file"
  done <"$env_file"

  # Only replace file if we made changes
  if [[ "$replaced" == "true" ]]; then
    mv "$temp_file" "$env_file"
    chmod 600 "$env_file"
    return 0
  else
    rm -f "$temp_file"
    return 1
  fi
}

# Generate secrets section for new .env file
generate_secrets_section() {
  local env_type="${1:-dev}"

  cat <<EOF

#####################################
# 🔐 Security Secrets
#####################################
# IMPORTANT: These are randomly generated secure values
# DO NOT commit these to version control
# For production, use: nself auth security rotate <SECRET_NAME>

EOF

  # Generate based on environment type
  if [[ "$env_type" == "production" ]] || [[ "$env_type" == "prod" ]]; then
    # Production: Ultra-strong secrets
    printf "POSTGRES_PASSWORD=%s\n" "$(generate_random_secret 48 alphanumeric)"
    printf "HASURA_GRAPHQL_ADMIN_SECRET=%s\n" "$(generate_random_secret 96 hex)"
    printf "HASURA_JWT_KEY=%s\n" "$(generate_random_secret 96 hex)"
    printf "MINIO_ROOT_PASSWORD=%s\n" "$(generate_random_secret 48 alphanumeric)"
    printf "S3_SECRET_KEY=%s\n" "$(generate_random_secret 64 hex)"
    printf "S3_ACCESS_KEY=%s\n" "$(generate_random_secret 32 alphanumeric)"
  else
    # Development: Strong but shorter
    printf "POSTGRES_PASSWORD=%s\n" "$(generate_random_secret 32 alphanumeric)"
    printf "HASURA_GRAPHQL_ADMIN_SECRET=%s\n" "$(generate_random_secret 64 hex)"
    printf "HASURA_JWT_KEY=%s\n" "$(generate_random_secret 64 hex)"
    printf "MINIO_ROOT_PASSWORD=%s\n" "$(generate_random_secret 32 alphanumeric)"
    printf "S3_SECRET_KEY=%s\n" "$(generate_random_secret 48 hex)"
    printf "S3_ACCESS_KEY=%s\n" "$(generate_random_secret 24 alphanumeric)"
  fi

  printf "\n"
}

# Add strong secrets to environment file if they're missing or weak
enhance_env_file_security() {
  local env_file="$1"
  local force="${2:-false}"

  if [[ ! -f "$env_file" ]]; then
    return 0
  fi

  local needs_enhancement=false

  # Check for weak default secrets
  local weak_patterns=(
    "postgres-dev-password"
    "hasura-admin-secret-dev"
    "development-secret-key"
    "minioadmin"
    "storage-secret-key-dev"
    "storage-access-key-dev"
    "admin"
    "password"
    "secret"
  )

  for pattern in "${weak_patterns[@]}"; do
    if grep -qi "$pattern" "$env_file" 2>/dev/null; then
      needs_enhancement=true
      break
    fi
  done

  if [[ "$needs_enhancement" == "true" ]] || [[ "$force" == "true" ]]; then
    replace_default_secrets_in_file "$env_file" "false"
    return $?
  fi

  return 1
}

# Export functions
export -f generate_random_secret
export -f replace_default_secrets_in_file
export -f generate_secrets_section
export -f enhance_env_file_security
