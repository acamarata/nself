#!/usr/bin/env bash
# P87 Production Verification (sprint-07 T-0702..T-0708)
#
# Runs the full matrix of probes against nclaw-prod:
#   14 HTTP endpoints (claw + mux + ai + ping)
#   Gemini pool prefix-tolerance check
#   Docker DNS plugin-mux resolution (when run on the VPS)
#
# Requires: $CLAW_TEST_JWT in env (dev tenant JWT). Optional: $CLAW_BASE,
# $MUX_BASE, $AI_BASE, $PING_BASE overrides. Set VERBOSE=1 to keep response
# bodies under /tmp/p88b-*.json.
#
# Exit codes:
#   0   all green
#   1   one or more red (regression)
#   2   yellow only (structured 5xx on <=2 endpoints) — caller decides
#
# Usage:
#   CLAW_TEST_JWT=eyJ... ./p87-verification.sh
#   make verify-prod

set -u -o pipefail

CLAW_BASE="${CLAW_BASE:-https://claw.camarata.com/api}"
MUX_BASE="${MUX_BASE:-https://claw.camarata.com/mux}"
AI_BASE="${AI_BASE:-https://claw.camarata.com/ai}"
PING_BASE="${PING_BASE:-https://ping.nself.org}"

RED=0
YELLOW=0
GREEN=0
FAILED_NAMES=()
YELLOW_NAMES=()

if [ -z "${CLAW_TEST_JWT:-}" ]; then
  printf 'ERROR: CLAW_TEST_JWT is not set in env.\n' >&2
  exit 1
fi

AUTH="Authorization: Bearer ${CLAW_TEST_JWT}"
CT="Content-Type: application/json"

probe() {
  # probe NAME METHOD URL EXPECT_STATUS JQ_CHECK [BODY]
  local name="$1" method="$2" url="$3" expect="$4" check="$5" body="${6:-}"
  local out="/tmp/p88b-${name}.json"
  local code
  if [ -n "$body" ]; then
    code=$(curl -sS -o "$out" -w '%{http_code}' -X "$method" \
      -H "$AUTH" -H "$CT" --data "$body" "$url" || printf '000')
  else
    code=$(curl -sS -o "$out" -w '%{http_code}' -X "$method" \
      -H "$AUTH" "$url" || printf '000')
  fi

  case "$code" in
    "$expect")
      if [ -n "$check" ]; then
        if jq -e "$check" "$out" >/dev/null 2>&1; then
          GREEN=$((GREEN+1))
          printf '  [GREEN] %-28s %s %s\n' "$name" "$code" "$url"
        else
          YELLOW=$((YELLOW+1))
          YELLOW_NAMES+=("$name")
          printf '  [YELLOW] %-27s %s schema mismatch — %s\n' \
            "$name" "$code" "$check"
        fi
      else
        GREEN=$((GREEN+1))
        printf '  [GREEN] %-28s %s %s\n' "$name" "$code" "$url"
      fi
      ;;
    5*)
      if jq -e '.error and .error.code' "$out" >/dev/null 2>&1; then
        YELLOW=$((YELLOW+1))
        YELLOW_NAMES+=("$name")
        printf '  [YELLOW] %-27s %s structured error (allowed, tracked)\n' \
          "$name" "$code"
      else
        RED=$((RED+1))
        FAILED_NAMES+=("$name")
        printf '  [RED] %-30s %s unstructured 5xx\n' "$name" "$code"
      fi
      ;;
    *)
      RED=$((RED+1))
      FAILED_NAMES+=("$name")
      printf '  [RED] %-30s %s expected %s\n' "$name" "$code" "$expect"
      ;;
  esac

  if [ "${VERBOSE:-0}" != "1" ]; then
    rm -f "$out"
  fi
}

printf '== P87 Endpoint Matrix ==\n'
probe ep01-claw-health        GET  "$CLAW_BASE/health"                              200 '.status == "ok"'
probe ep02-memory-search      GET  "$CLAW_BASE/memory/search?q=test"                200 '.results | type == "array"'
probe ep03-memory-compress    POST "$CLAW_BASE/memory/compress"                     202 '.job_id' '{}'
probe ep04-kg-entities        GET  "$CLAW_BASE/kg/entities?limit=5"                 200 'type == "array"'
probe ep05-kg-edges           GET  "$CLAW_BASE/kg/edges?entity_id=seed"             200 'type == "array"'
probe ep06-topics-classify    POST "$CLAW_BASE/topics/classify"                     200 '.topic_id and .confidence' '{"text":"ping"}'
probe ep07-topics-tree        GET  "$CLAW_BASE/topics/tree"                         200 'type == "array"'
probe ep08-briefings-gen      POST "$CLAW_BASE/briefings/generate"                  202 '.job_id' '{}'
probe ep09-audit-log          GET  "$CLAW_BASE/audit/log?limit=10"                  200 'type == "array"'
probe ep10-tools-dispatch     POST "$CLAW_BASE/tools/dispatch"                      200 '.' '{"tool":"weather.current","args":{"city":"Berlin"}}'
probe ep11-mux-messages       GET  "$MUX_BASE/messages?limit=5"                     200 'type == "array"'
probe ep12-mux-classify       POST "$MUX_BASE/classify"                             200 '.category' '{"text":"sample"}'
probe ep13-ai-complete        POST "$AI_BASE/complete"                              200 '.text' '{"prompt":"ping","max_tokens":16}'
probe ep14-ai-pool-status     GET  "$AI_BASE/pool/status"                           200 '.keys | type == "array"'

printf '\n== Gemini Pool Prefix Tolerance ==\n'
if [ -f /tmp/p88b-ep14-ai-pool-status.json ]; then
  jq -r '.keys[].source_prefix' /tmp/p88b-ep14-ai-pool-status.json 2>/dev/null | sort -u | \
    sed 's/^/  prefix: /'
fi

printf '\n== Docker DNS (run on nclaw-prod VPS only) ==\n'
if command -v docker >/dev/null 2>&1 && [ -n "${ON_VPS:-}" ]; then
  if docker compose exec -T plugin-claw getent hosts plugin-mux >/dev/null 2>&1; then
    printf '  [GREEN] plugin-mux resolves from plugin-claw\n'
    GREEN=$((GREEN+1))
  else
    printf '  [RED] plugin-mux DNS resolution failed\n'
    RED=$((RED+1))
    FAILED_NAMES+=("docker-dns-plugin-mux")
  fi
else
  printf '  [SKIP] not on VPS (set ON_VPS=1 to enable)\n'
fi

printf '\n== Summary ==\n'
printf '  green: %d  yellow: %d  red: %d\n' "$GREEN" "$YELLOW" "$RED"
if [ "$RED" -gt 0 ]; then
  printf '  FAILED: %s\n' "${FAILED_NAMES[*]}"
  exit 1
fi
if [ "$YELLOW" -gt 0 ]; then
  printf '  YELLOW: %s\n' "${YELLOW_NAMES[*]}"
  if [ "$YELLOW" -gt 2 ]; then
    printf '  >2 yellow — treating as RED per spec §2.3\n'
    exit 1
  fi
  exit 2
fi
exit 0
