#!/usr/bin/env bash
# T04 — 100% mux classification rate.
# Queries Prometheus for mux_classify_success_total / mux_classify_total.
# Threshold: >= 0.999 (allow 1-in-1000 for timing-race transients inside scrape window).

set -euo pipefail

PROM_URL="${PROM_URL:-https://prometheus.nself.org/api/v1/query}"
THRESHOLD="${THRESHOLD:-0.999}"
WINDOW="${WINDOW:-10m}"

query="sum(rate(mux_classify_success_total[${WINDOW}])) / sum(rate(mux_classify_total[${WINDOW}]))"

# Prometheus alert rule (reference) — ship in prometheus/alerts/p93-soak.yml:
#   - alert: MuxClassificationBelowThreshold
#     expr: sum(rate(mux_classify_success_total[10m])) / sum(rate(mux_classify_total[10m])) < 0.999
#     for: 5m
#     labels: { severity: critical, soak_gate: T04 }
#     annotations:
#       summary: "mux classification rate {{ $value }} < 0.999"

resp="$(curl -fsSL --max-time 15 --data-urlencode "query=${query}" "$PROM_URL" 2>/dev/null || echo '{}')"
val="$(printf '%s' "$resp" | python3 -c "import sys,json
try:
  d = json.load(sys.stdin)
  r = d.get('data',{}).get('result',[])
  print(r[0]['value'][1] if r else 'nan')
except Exception:
  print('nan')")"

if [ "$val" = "nan" ]; then
  printf "FAIL T04: no data from Prometheus (ensure mux plugin is exporting metrics)\n"
  exit 1
fi

awk -v v="$val" -v t="$THRESHOLD" 'BEGIN {
  if (v + 0 >= t + 0) { printf "OK T04: mux classification = %s (>= %s)\n", v, t; exit 0 }
  else { printf "FAIL T04: mux classification = %s (< %s)\n", v, t; exit 1 }
}'
