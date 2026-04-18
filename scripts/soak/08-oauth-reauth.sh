#!/usr/bin/env bash
# T08 — OAuth re-auth flow end-to-end.
# Uses a long-lived refresh token stored in vault (SOAK_GOOGLE_REFRESH_TOKEN).
# Exchanges refresh for access, calls userinfo, validates response.

set -u

CLIENT_ID="${GOOGLE_OAUTH_CLIENT_ID:-}"
CLIENT_SECRET="${GOOGLE_OAUTH_CLIENT_SECRET:-}"
REFRESH_TOKEN="${SOAK_GOOGLE_REFRESH_TOKEN:-}"

if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ] || [ -z "$REFRESH_TOKEN" ]; then
  printf "FAIL T08: missing OAuth env (GOOGLE_OAUTH_CLIENT_ID/SECRET + SOAK_GOOGLE_REFRESH_TOKEN)\n"
  exit 2
fi

# 1. Refresh access token.
token_resp="$(curl -fsSL --max-time 15 \
  -d "client_id=${CLIENT_ID}" \
  -d "client_secret=${CLIENT_SECRET}" \
  -d "refresh_token=${REFRESH_TOKEN}" \
  -d "grant_type=refresh_token" \
  "https://oauth2.googleapis.com/token" 2>/dev/null || echo '{}')"

access_token="$(printf '%s' "$token_resp" | python3 -c "import sys,json
try: print(json.load(sys.stdin).get('access_token',''))
except Exception: print('')" 2>/dev/null)"

if [ -z "$access_token" ]; then
  printf "FAIL T08: refresh token exchange failed. resp=%s\n" "$token_resp"
  exit 1
fi

# 2. Call userinfo with fresh token.
userinfo_code="$(curl -fsSL -o /dev/null -w '%{http_code}' --max-time 10 \
  -H "Authorization: Bearer ${access_token}" \
  "https://openidconnect.googleapis.com/v1/userinfo" 2>/dev/null || echo 000)"

if [ "$userinfo_code" != "200" ]; then
  printf "FAIL T08: userinfo returned %s with refreshed access token\n" "$userinfo_code"
  exit 1
fi

printf "OK T08: OAuth refresh + userinfo 200\n"
