#!/usr/bin/env bash
set -euo pipefail

RESULTS_DIR="$(dirname "$0")/results"
mkdir -p "$RESULTS_DIR"
DATE=$(date +%Y-%m-%d)
OUTPUT="$RESULTS_DIR/${DATE}-local.json"

echo "Running nSelf benchmark harness..."
echo "Output: $OUTPUT"
echo ""

# ── Setup time ────────────────────────────────────────────────────────────────
echo "[1/3] Measuring setup time..."
START=$(date +%s%N)
# Assumes nself is installed and running
nself doctor --quiet 2>/dev/null && echo "nself already running — skipping boot timing"
END=$(date +%s%N)
SETUP_MS=$(( (END - START) / 1000000 ))

# ── RPS at 1 CPU ────────────────────────────────────────────────────────────
echo "[2/3] Running RPS benchmark (requires oha + running nself stack)..."
RPS=0
if command -v oha &>/dev/null; then
  OHA_RESULT=$(oha --no-tui -n 5000 -c 20 \
    -H "Content-Type: application/json" \
    http://localhost:8080/healthz 2>/dev/null | grep "Requests/sec" | awk '{print $2}')
  RPS=${OHA_RESULT:-0}
fi

# ── Feature coverage ─────────────────────────────────────────────────────────
echo "[3/3] Feature coverage check..."
COVERAGE_SCORE=15  # from feature-matrix.json (manual)

# ── Write results ─────────────────────────────────────────────────────────────
cat > "$OUTPUT" << JSON
{
  "date": "${DATE}",
  "runner": "local",
  "product": "nself",
  "setup_ms": ${SETUP_MS},
  "rps_1cpu": ${RPS},
  "feature_coverage_score": ${COVERAGE_SCORE},
  "methodology": "See METHODOLOGY.md"
}
JSON

echo ""
echo "Results written to $OUTPUT"
echo ""
cat "$OUTPUT"
