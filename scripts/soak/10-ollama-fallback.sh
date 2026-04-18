#!/usr/bin/env bash
# T10 — Ollama fallback verified under simulated cloud-provider outage.
# Uses the mux plugin's fault-injection mode (FAULT_INJECT_DROP=gemini,claude,openai)
# to force routing to Ollama, then issues a classify request and asserts
# provider=ollama in the response.

set -u

MUX_BASE="${MUX_BASE:-https://api.nself.org/mux}"
FAULT_HDR="x-fault-inject-drop: gemini,claude,openai"

resp="$(curl -fsSL --max-time 30 \
  -H "$FAULT_HDR" \
  -H 'content-type: application/json' \
  -d '{"text":"Schedule a meeting with Bob at 3pm tomorrow about the Q3 roadmap.","task":"classify"}' \
  "$MUX_BASE/classify" 2>/dev/null || echo '{}')"

provider="$(printf '%s' "$resp" | python3 -c "import sys,json
try: print(json.load(sys.stdin).get('provider',''))
except Exception: print('')")"

if [ "$provider" != "ollama" ]; then
  printf "FAIL T10: expected provider=ollama under cloud-outage simulation, got=%s resp=%s\n" "$provider" "$resp"
  exit 1
fi

# Response must still include a valid classification.
topic="$(printf '%s' "$resp" | python3 -c "import sys,json
try: print(json.load(sys.stdin).get('topic',''))
except Exception: print('')")"
if [ -z "$topic" ]; then
  printf "FAIL T10: Ollama responded but classification was empty. resp=%s\n" "$resp"
  exit 1
fi

printf "OK T10: Ollama fallback served classification (topic=%s)\n" "$topic"
