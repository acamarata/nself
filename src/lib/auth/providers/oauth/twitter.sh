#!/usr/bin/env bash
# twitter.sh - Twitter OAuth 2.0 provider (OAUTH-007)
# Part of nself v0.6.0 - Phase 1 Sprint 1

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source OAuth base
if [[ -f "$SCRIPT_DIR/oauth-base.sh" ]]; then
  source "$SCRIPT_DIR/oauth-base.sh"
fi

# Twitter OAuth endpoints
readonly TWITTER_AUTH_ENDPOINT="https://twitter.com/i/oauth2/authorize"
readonly TWITTER_TOKEN_ENDPOINT="https://api.twitter.com/2/oauth2/token"
readonly TWITTER_USERINFO_ENDPOINT="https://api.twitter.com/2/users/me"

# Default scopes
readonly TWITTER_DEFAULT_SCOPES="tweet.read users.read"

# Get Twitter authorization URL
# Usage: twitter_get_auth_url <client_id> <redirect_uri> [scopes]
twitter_get_auth_url() {
  local client_id="$1"
  local redirect_uri="$2"
  local scopes="${3:-$TWITTER_DEFAULT_SCOPES}"

  local state
  state=$(oauth_generate_state)

  oauth_store_state "$state" "{\"provider\": \"twitter\"}"

  # Twitter requires code_challenge for PKCE
  local code_challenge
  code_challenge=$(openssl rand -hex 32 | base64)

  oauth_build_auth_url "$TWITTER_AUTH_ENDPOINT" "$client_id" "$redirect_uri" "$scopes" "$state"
}

# Exchange Twitter authorization code for tokens
# Usage: twitter_exchange_code <client_id> <client_secret> <code> <redirect_uri>
twitter_exchange_code() {
  local client_id="$1"
  local client_secret="$2"
  local code="$3"
  local redirect_uri="$4"

  # Create base64 encoded client credentials
  local encoded_credentials
  encoded_credentials=$(echo -n "${client_id}:${client_secret}" | base64)

  # Make POST request to token endpoint
  local response
  response=$(curl -s -X POST "$TWITTER_TOKEN_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Authorization: Basic ${encoded_credentials}" \
    -d "grant_type=authorization_code" \
    -d "client_id=${client_id}" \
    -d "code=${code}" \
    -d "redirect_uri=${redirect_uri}")

  echo "$response"
}

# Get Twitter user info
# Usage: twitter_get_user_info <access_token>
twitter_get_user_info() {
  local access_token="$1"

  # Twitter API v2 requires explicit fields parameter
  local response
  response=$(curl -s -X GET "${TWITTER_USERINFO_ENDPOINT}?user.fields=id,name,username,email,created_at" \
    -H "Authorization: Bearer ${access_token}")

  echo "$response"
}

# Refresh Twitter access token
# Usage: twitter_refresh_token <client_id> <client_secret> <refresh_token>
twitter_refresh_token() {
  local client_id="$1"
  local client_secret="$2"
  local refresh_token="$3"

  # Create base64 encoded client credentials
  local encoded_credentials
  encoded_credentials=$(echo -n "${client_id}:${client_secret}" | base64)

  # Make POST request to token endpoint
  local response
  response=$(curl -s -X POST "$TWITTER_TOKEN_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Authorization: Basic ${encoded_credentials}" \
    -d "grant_type=refresh_token" \
    -d "refresh_token=${refresh_token}")

  echo "$response"
}

# Revoke Twitter token
# Usage: twitter_revoke_token <client_id> <client_secret> <token>
twitter_revoke_token() {
  local client_id="$1"
  local client_secret="$2"
  local token="$3"

  # Create base64 encoded client credentials
  local encoded_credentials
  encoded_credentials=$(echo -n "${client_id}:${client_secret}" | base64)

  curl -s -X POST "https://api.twitter.com/2/oauth2/revoke" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Authorization: Basic ${encoded_credentials}" \
    -d "token=${token}" \
    -d "token_type_hint=access_token" \
    >/dev/null 2>&1
}

# Export functions
export -f twitter_get_auth_url
export -f twitter_exchange_code
export -f twitter_get_user_info
export -f twitter_refresh_token
export -f twitter_revoke_token
