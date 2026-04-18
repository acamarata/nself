#!/usr/bin/env bash
# T07 — 4 pool keys healthy: Gemini / Claude / OpenAI / Ollama.
# Each provider exposes /health through the AI plugin router.

set -u

POOL_BASE="${POOL_BASE:-https://api.nself.org/ai/pool}"

PROVIDERS="gemini claude openai ollama"

pass=0
fail=0
for p in $PROVIDERS; do
  url="${POOL_BASE}/${p}/health"
  code="$(curl -fsSL -o /tmp/soak-pool-body.$$ -w '%{http_code}' --max-time 10 "$url" 2>/dev/null || echo 000)"
  body="$(cat /tmp/soak-pool-body.$$ 2>/dev/null | head -1)"
  rm -f /tmp/soak-pool-body.$$
  if [ "$code" = "200" ]; then
    printf "  OK   pool=%-7s http=%s body=%s\n" "$p" "$code" "$body"
    pass=$((pass + 1))
  else
    printf "  FAIL pool=%-7s http=%s body=%s\n" "$p" "$code" "$body"
    fail=$((fail + 1))
  fi
done

printf "\nResult: %s/4 pool keys healthy\n" "$pass"
[ "$fail" -eq 0 ] || exit 1
printf "OK T07: all 4 pool keys (Gemini/Claude/OpenAI/Ollama) healthy\n"
