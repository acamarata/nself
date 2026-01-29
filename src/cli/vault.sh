#!/usr/bin/env bash
# vault.sh - Secrets vault management CLI (SEC-007)
# Part of nself v0.6.0 - Phase 1 Sprint 4
#
# Complete CLI interface for encrypted secrets vault

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

if [[ -f "$LIB_DIR/secrets/vault.sh" ]]; then
  source "$LIB_DIR/secrets/vault.sh"
fi
if [[ -f "$LIB_DIR/secrets/encryption.sh" ]]; then
  source "$LIB_DIR/secrets/encryption.sh"
fi
if [[ -f "$LIB_DIR/secrets/audit.sh" ]]; then
  source "$LIB_DIR/secrets/audit.sh"
fi
if [[ -f "$LIB_DIR/secrets/environment.sh" ]]; then
  source "$LIB_DIR/secrets/environment.sh"
fi

# ============================================================================
# CLI Main
# ============================================================================

cmd_vault() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    init)
      cmd_vault_init "$@"
      ;;
    set)
      cmd_vault_set "$@"
      ;;
    get)
      cmd_vault_get "$@"
      ;;
    delete|rm)
      cmd_vault_delete "$@"
      ;;
    list|ls)
      cmd_vault_list "$@"
      ;;
    rotate)
      cmd_vault_rotate "$@"
      ;;
    versions)
      cmd_vault_versions "$@"
      ;;
    rollback)
      cmd_vault_rollback "$@"
      ;;
    audit)
      cmd_vault_audit "$@"
      ;;
    compare)
      cmd_vault_compare "$@"
      ;;
    sync)
      cmd_vault_sync "$@"
      ;;
    promote)
      cmd_vault_promote "$@"
      ;;
    status)
      cmd_vault_status "$@"
      ;;
    keys)
      cmd_vault_keys "$@"
      ;;
    help|--help|-h)
      cmd_vault_help
      ;;
    *)
      echo "ERROR: Unknown command: $subcommand"
      echo "Run 'nself vault help' for usage information"
      return 1
      ;;
  esac
}

# ============================================================================
# Subcommands
# ============================================================================

# Initialize secrets vault
cmd_vault_init() {
  echo "Initializing secrets vault..."

  # Initialize encryption
  if ! encryption_init; then
    echo "ERROR: Failed to initialize encryption" >&2
    return 1
  fi

  # Initialize vault
  if ! vault_init; then
    echo "ERROR: Failed to initialize vault" >&2
    return 1
  fi

  # Initialize audit
  if ! audit_init; then
    echo "ERROR: Failed to initialize audit" >&2
    return 1
  fi

  printf "\n✓ Secrets vault initialized successfully\n"
  return 0
}

# Set secret
cmd_vault_set() {
  local key_name=""
  local value=""
  local environment="default"
  local description=""
  local expires=""
  local from_stdin=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --value)
        value="$2"
        shift 2
        ;;
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --description|--desc)
        description="$2"
        shift 2
        ;;
      --expires)
        expires="$2"
        shift 2
        ;;
      --stdin)
        from_stdin=true
        shift
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        elif [[ -z "$value" ]]; then
          value="$1"
        fi
        shift
        ;;
    esac
  done

  # Read from stdin if requested
  if [[ "$from_stdin" == "true" ]]; then
    value=$(cat)
  fi

  if [[ -z "$key_name" ]]; then
    echo "ERROR: Key name required"
    echo "Usage: nself vault set --key <name> --value <value> [--env <environment>]"
    return 1
  fi

  if [[ -z "$value" ]]; then
    echo "ERROR: Value required"
    echo "Use --value <value> or --stdin to read from stdin"
    return 1
  fi

  # Set secret
  local secret_id
  secret_id=$(env_set_secret "$key_name" "$value" "$environment" "$description" "$expires")

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to set secret"
    return 1
  fi

  printf "✓ Secret set: %s (environment: %s)\n" "$key_name" "$environment"
  printf "  ID: %s\n" "$secret_id"
  return 0
}

# Get secret
cmd_vault_get() {
  local key_name=""
  local environment="default"
  local version=""
  local format="plain"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --version)
        version="$2"
        shift 2
        ;;
      --format)
        format="$2"
        shift 2
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$key_name" ]]; then
    echo "ERROR: Key name required"
    echo "Usage: nself vault get --key <name> [--env <environment>]"
    return 1
  fi

  # Get secret
  local value
  if [[ -n "$version" ]]; then
    value=$(vault_get "$key_name" "$environment" "$version")
  else
    value=$(env_get_secret "$key_name" "$environment")
  fi

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Secret not found: $key_name (environment: $environment)"
    return 1
  fi

  # Output based on format
  case "$format" in
    plain)
      echo "$value"
      ;;
    json)
      jq -n --arg key "$key_name" --arg value "$value" --arg env "$environment" \
        '{key: $key, value: $value, environment: $env}'
      ;;
    *)
      echo "$value"
      ;;
  esac

  return 0
}

# Delete secret
cmd_vault_delete() {
  local key_name=""
  local environment="default"
  local confirm=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --yes|-y)
        confirm=true
        shift
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$key_name" ]]; then
    echo "ERROR: Key name required"
    echo "Usage: nself vault delete --key <name> [--env <environment>]"
    return 1
  fi

  # Confirm deletion
  if [[ "$confirm" == "false" ]]; then
    printf "Delete secret '%s' in environment '%s'? (y/N): " "$key_name" "$environment"
    read -r response
    response=$(echo "$response" | tr '[:upper:]' '[:lower:]')

    if [[ "$response" != "y" ]] && [[ "$response" != "yes" ]]; then
      echo "Cancelled"
      return 0
    fi
  fi

  # Delete secret
  if ! env_delete_secret "$key_name" "$environment"; then
    echo "ERROR: Failed to delete secret"
    return 1
  fi

  printf "✓ Secret deleted: %s (environment: %s)\n" "$key_name" "$environment"
  return 0
}

# List secrets
cmd_vault_list() {
  local environment=""
  local format="table"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --format)
        format="$2"
        shift 2
        ;;
      --json)
        format="json"
        shift
        ;;
      *)
        if [[ -z "$environment" ]]; then
          environment="$1"
        fi
        shift
        ;;
    esac
  done

  # List secrets
  local secrets
  if [[ -n "$environment" ]]; then
    secrets=$(vault_list "$environment")
  else
    secrets=$(vault_list)
  fi

  if [[ "$secrets" == "[]" ]]; then
    echo "No secrets found"
    return 0
  fi

  # Output based on format
  case "$format" in
    json)
      echo "$secrets" | jq '.'
      ;;
    table)
      echo "$secrets" | jq -r '["KEY", "ENVIRONMENT", "VERSION", "CREATED"],
        (.[] | [.key_name, .environment, .version, .created_at]) | @tsv' | column -t
      ;;
    *)
      echo "$secrets" | jq -r '.[] | "\(.key_name) (\(.environment))"'
      ;;
  esac

  return 0
}

# Rotate secret
cmd_vault_rotate() {
  local key_name=""
  local environment="default"
  local rotate_all=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --all)
        rotate_all=true
        shift
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ "$rotate_all" == "true" ]]; then
    # Rotate all secrets
    if ! vault_rotate_all; then
      echo "ERROR: Failed to rotate secrets"
      return 1
    fi

    printf "✓ All secrets rotated\n"
    return 0
  fi

  if [[ -z "$key_name" ]]; then
    echo "ERROR: Key name required (or use --all)"
    echo "Usage: nself vault rotate --key <name> [--env <environment>]"
    echo "       nself vault rotate --all"
    return 1
  fi

  # Rotate single secret
  if ! vault_rotate "$key_name" "$environment"; then
    echo "ERROR: Failed to rotate secret"
    return 1
  fi

  printf "✓ Secret rotated: %s (environment: %s)\n" "$key_name" "$environment"
  return 0
}

# Show version history
cmd_vault_versions() {
  local key_name=""
  local environment="default"
  local format="table"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --format)
        format="$2"
        shift 2
        ;;
      --json)
        format="json"
        shift
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$key_name" ]]; then
    echo "ERROR: Key name required"
    echo "Usage: nself vault versions --key <name> [--env <environment>]"
    return 1
  fi

  # Get versions
  local versions
  versions=$(vault_get_versions "$key_name" "$environment")

  if [[ "$versions" == "[]" ]]; then
    echo "No version history found"
    return 0
  fi

  # Output based on format
  case "$format" in
    json)
      echo "$versions" | jq '.'
      ;;
    table)
      echo "$versions" | jq -r '["VERSION", "CHANGED_AT"],
        (.[] | [.version, .changed_at]) | @tsv' | column -t
      ;;
    *)
      echo "$versions" | jq -r '.[] | "v\(.version) - \(.changed_at)"'
      ;;
  esac

  return 0
}

# Rollback to previous version
cmd_vault_rollback() {
  local key_name=""
  local version=""
  local environment="default"
  local confirm=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --version)
        version="$2"
        shift 2
        ;;
      --env|--environment)
        environment="$2"
        shift 2
        ;;
      --yes|-y)
        confirm=true
        shift
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        elif [[ -z "$version" ]]; then
          version="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$key_name" ]] || [[ -z "$version" ]]; then
    echo "ERROR: Key name and version required"
    echo "Usage: nself vault rollback --key <name> --version <version> [--env <environment>]"
    return 1
  fi

  # Confirm rollback
  if [[ "$confirm" == "false" ]]; then
    printf "Rollback '%s' to version %s? (y/N): " "$key_name" "$version"
    read -r response
    response=$(echo "$response" | tr '[:upper:]' '[:lower:]')

    if [[ "$response" != "y" ]] && [[ "$response" != "yes" ]]; then
      echo "Cancelled"
      return 0
    fi
  fi

  # Rollback
  if ! vault_rollback "$key_name" "$version" "$environment"; then
    echo "ERROR: Failed to rollback secret"
    return 1
  fi

  printf "✓ Secret rolled back: %s to version %s\n" "$key_name" "$version"
  return 0
}

# Show audit logs
cmd_vault_audit() {
  local key_name=""
  local environment=""
  local action=""
  local limit=50
  local format="table"
  local subcommand="${1:-logs}"

  shift || true

  case "$subcommand" in
    logs)
      # Parse options
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --key|--name)
            key_name="$2"
            shift 2
            ;;
          --env|--environment)
            environment="$2"
            shift 2
            ;;
          --action)
            action="$2"
            shift 2
            ;;
          --limit)
            limit="$2"
            shift 2
            ;;
          --format)
            format="$2"
            shift 2
            ;;
          --json)
            format="json"
            shift
            ;;
          *)
            shift
            ;;
        esac
      done

      # Get logs
      local logs
      logs=$(audit_get_logs "$key_name" "$environment" "$action" "$limit")

      if [[ "$logs" == "[]" ]]; then
        echo "No audit logs found"
        return 0
      fi

      # Output
      case "$format" in
        json)
          echo "$logs" | jq '.'
          ;;
        table)
          echo "$logs" | jq -r '["KEY", "ACTION", "RESULT", "TIME"],
            (.[] | [.key_name, .action, .result, .accessed_at]) | @tsv' | column -t
          ;;
        *)
          echo "$logs" | jq -r '.[] | "\(.accessed_at) - \(.key_name) (\(.action)): \(.result)"'
          ;;
      esac
      ;;

    summary)
      # Get summary for a secret
      local key="${1:-}"
      local env="${2:-default}"

      if [[ -z "$key" ]]; then
        echo "ERROR: Key name required"
        return 1
      fi

      local summary
      summary=$(audit_get_summary "$key" "$env")

      echo "$summary" | jq '.'
      ;;

    failures)
      # Show failed attempts
      local key="${1:-}"
      local limit="${2:-50}"

      local failures
      failures=$(audit_get_failures "$key" "$limit")

      if [[ "$failures" == "[]" ]]; then
        echo "No failures found"
        return 0
      fi

      echo "$failures" | jq '.'
      ;;

    suspicious)
      # Detect suspicious activity
      local threshold="${1:-10}"

      local suspicious
      suspicious=$(audit_detect_suspicious "$threshold")

      if [[ "$suspicious" == "[]" ]]; then
        echo "No suspicious activity detected"
        return 0
      fi

      echo "$suspicious" | jq '.'
      ;;

    *)
      echo "ERROR: Unknown audit subcommand: $subcommand"
      return 1
      ;;
  esac

  return 0
}

# Compare secrets across environments
cmd_vault_compare() {
  local env1="${1:-}"
  local env2="${2:-}"

  if [[ -z "$env1" ]] || [[ -z "$env2" ]]; then
    echo "ERROR: Two environments required"
    echo "Usage: nself vault compare <env1> <env2>"
    return 1
  fi

  # Compare
  local comparison
  comparison=$(env_compare "$env1" "$env2")

  echo "$comparison" | jq '.'
  return 0
}

# Sync secrets between environments
cmd_vault_sync() {
  local key_name=""
  local source_env=""
  local target_env=""
  local sync_all=false
  local overwrite=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --from|--source)
        source_env="$2"
        shift 2
        ;;
      --to|--target)
        target_env="$2"
        shift 2
        ;;
      --all)
        sync_all=true
        shift
        ;;
      --overwrite)
        overwrite=true
        shift
        ;;
      *)
        if [[ -z "$source_env" ]]; then
          source_env="$1"
        elif [[ -z "$target_env" ]]; then
          target_env="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$source_env" ]] || [[ -z "$target_env" ]]; then
    echo "ERROR: Source and target environments required"
    echo "Usage: nself vault sync --from <source> --to <target> [--key <name> | --all]"
    return 1
  fi

  if [[ "$sync_all" == "true" ]]; then
    # Sync all secrets
    if ! env_sync_all "$source_env" "$target_env" "$overwrite"; then
      echo "ERROR: Failed to sync secrets"
      return 1
    fi

    printf "✓ Synced all secrets from %s to %s\n" "$source_env" "$target_env"
    return 0
  fi

  if [[ -z "$key_name" ]]; then
    echo "ERROR: Key name required (or use --all)"
    return 1
  fi

  # Sync single secret
  if ! env_sync_secret "$key_name" "$source_env" "$target_env"; then
    echo "ERROR: Failed to sync secret"
    return 1
  fi

  printf "✓ Synced %s from %s to %s\n" "$key_name" "$source_env" "$target_env"
  return 0
}

# Promote secret through environments
cmd_vault_promote() {
  local key_name=""
  local from_env=""
  local to_env=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --key|--name)
        key_name="$2"
        shift 2
        ;;
      --from)
        from_env="$2"
        shift 2
        ;;
      --to)
        to_env="$2"
        shift 2
        ;;
      *)
        if [[ -z "$key_name" ]]; then
          key_name="$1"
        elif [[ -z "$from_env" ]]; then
          from_env="$1"
        elif [[ -z "$to_env" ]]; then
          to_env="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$key_name" ]] || [[ -z "$from_env" ]] || [[ -z "$to_env" ]]; then
    echo "ERROR: Key name, source, and target environment required"
    echo "Usage: nself vault promote --key <name> --from <env> --to <env>"
    return 1
  fi

  # Promote
  if ! env_promote "$key_name" "$from_env" "$to_env"; then
    echo "ERROR: Failed to promote secret"
    return 1
  fi

  printf "✓ Promoted %s from %s to %s\n" "$key_name" "$from_env" "$to_env"
  return 0
}

# Show environment status
cmd_vault_status() {
  local environment="${1:-}"

  if [[ -n "$environment" ]]; then
    # Single environment status
    local status
    status=$(env_get_status "$environment")

    echo "$status" | jq '.'
  else
    # All environments
    for env in default dev staging prod; do
      printf "\n=== %s ===\n" "$env"
      local status
      status=$(env_get_status "$env")
      echo "$status" | jq '.'
    done
  fi

  return 0
}

# Manage encryption keys
cmd_vault_keys() {
  local subcommand="${1:-list}"
  shift || true

  case "$subcommand" in
    list|ls)
      local keys
      keys=$(encryption_list_keys)

      if [[ "$keys" == "[]" ]]; then
        echo "No encryption keys found"
        return 0
      fi

      echo "$keys" | jq -r '["ID", "ALGORITHM", "ACTIVE", "CREATED"],
        (.[] | [.id, .algorithm, .is_active, .created_at]) | @tsv' | column -t
      ;;

    rotate)
      if ! encryption_rotate_key; then
        echo "ERROR: Failed to rotate encryption key"
        return 1
      fi

      printf "✓ Encryption key rotated\n"
      printf "  Run 'nself vault rotate --all' to re-encrypt all secrets\n"
      ;;

    check)
      if encryption_check_rotation; then
        printf "⚠ Encryption key rotation recommended\n"
        return 0
      else
        printf "✓ Encryption key is current\n"
        return 0
      fi
      ;;

    *)
      echo "ERROR: Unknown keys subcommand: $subcommand"
      echo "Available: list, rotate, check"
      return 1
      ;;
  esac

  return 0
}

# ============================================================================
# Help
# ============================================================================

cmd_vault_help() {
  cat <<'EOF'
nself vault - Encrypted secrets vault management

USAGE:
  nself vault <command> [options]

COMMANDS:
  init                 Initialize secrets vault
  set                  Set a secret value
  get                  Get a secret value
  delete               Delete a secret
  list                 List all secrets
  rotate               Rotate secret encryption
  versions             Show secret version history
  rollback             Rollback to previous version
  audit                View audit logs
  compare              Compare secrets across environments
  sync                 Sync secrets between environments
  promote              Promote secret through environments
  status               Show environment status
  keys                 Manage encryption keys

EXAMPLES:
  # Initialize vault
  nself vault init

  # Set a secret
  nself vault set --key DB_PASSWORD --value "secret123" --env prod
  echo "secret" | nself vault set --key API_KEY --stdin --env dev

  # Get a secret
  nself vault get --key DB_PASSWORD --env prod
  nself vault get DB_PASSWORD prod --format json

  # List secrets
  nself vault list --env prod
  nself vault list --json

  # Rotate secrets
  nself vault rotate --key DB_PASSWORD --env prod
  nself vault rotate --all

  # Version management
  nself vault versions --key DB_PASSWORD
  nself vault rollback --key DB_PASSWORD --version 2

  # Audit logs
  nself vault audit logs --key DB_PASSWORD --limit 20
  nself vault audit summary DB_PASSWORD prod
  nself vault audit failures

  # Environment management
  nself vault compare dev prod
  nself vault sync --key DB_PASSWORD --from staging --to prod
  nself vault sync --all --from dev --to staging
  nself vault promote --key DB_PASSWORD --from dev --to staging

  # Encryption keys
  nself vault keys list
  nself vault keys rotate
  nself vault keys check

For more information: https://docs.nself.org/vault
EOF
}

# ============================================================================
# Export
# ============================================================================

export -f cmd_vault

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_vault "$@"
fi
