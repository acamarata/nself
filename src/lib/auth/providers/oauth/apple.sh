#!/usr/bin/env bash
# apple.sh - Apple OAuth 2.0 provider (OAUTH-005)
# Part of nself v0.6.0 - Phase 1 Sprint 1

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source OAuth base
if [[ -f "$SCRIPT_DIR/oauth-base.sh" ]]; then
  source "$SCRIPT_DIR/oauth-base.sh"
fi

# Apple OAuth endpoints
readonly APPLE_AUTH_ENDPOINT="https://appleid.apple.com/auth/authorize"
readonly APPLE_TOKEN_ENDPOINT="https://appleid.apple.com/auth/token"
readonly APPLE_REVOKE_ENDPOINT="https://appleid.apple.com/auth/revoke"

# Default scopes
readonly APPLE_DEFAULT_SCOPES="name email"

# Get Apple authorization URL
# Usage: apple_get_auth_url <client_id> <redirect_uri> [scopes]
apple_get_auth_url() {
  local client_id="$1"
  local redirect_uri="$2"
  local scopes="${3:-$APPLE_DEFAULT_SCOPES}"

  local state
  state=$(oauth_generate_state)

  oauth_store_state "$state" "{\"provider\": \"apple\"}"

  oauth_build_auth_url "$APPLE_AUTH_ENDPOINT" "$client_id" "$redirect_uri" "$scopes" "$state"
}

# Exchange Apple authorization code for tokens
# Usage: apple_exchange_code <client_id> <client_secret> <code> <redirect_uri>
apple_exchange_code() {
  local client_id="$1"
  local client_secret="$2"
  local code="$3"
  local redirect_uri="$4"

  oauth_exchange_code "$APPLE_TOKEN_ENDPOINT" "$client_id" "$client_secret" "$code" "$redirect_uri"
}

# Get Apple user info
# Usage: apple_get_user_info <access_token>
apple_get_user_info() {
  local access_token="$1"

  # Apple doesn't have a dedicated userinfo endpoint
  # User info is returned in the token response during initial exchange
  # This function is for interface compatibility
  echo "{}"
}

# Refresh Apple access token
# Usage: apple_refresh_token <client_id> <client_secret> <refresh_token>
apple_refresh_token() {
  local client_id="$1"
  local client_secret="$2"
  local refresh_token="$3"

  oauth_refresh_token "$APPLE_TOKEN_ENDPOINT" "$client_id" "$client_secret" "$refresh_token"
}

# Revoke Apple token
# Usage: apple_revoke_token <client_id> <client_secret> <token>
apple_revoke_token() {
  local client_id="$1"
  local client_secret="$2"
  local token="$3"

  curl -s -X POST "$APPLE_REVOKE_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "client_id=${client_id}" \
    -d "client_secret=${client_secret}" \
    -d "token=${token}" \
    -d "token_type_hint=access_token" \
    >/dev/null 2>&1
}

# Export functions
export -f apple_get_auth_url
export -f apple_exchange_code
export -f apple_get_user_info
export -f apple_refresh_token
export -f apple_revoke_token
