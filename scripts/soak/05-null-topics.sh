#!/usr/bin/env bash
# T05 — zero NULL topic_id rows in nclaw.conversation_topics.
# Shells out to psql via the prod proxy. Fails if count > 0.

set -euo pipefail

PG_CONN="${PG_CONN:-${DATABASE_URL:-}}"
if [ -z "$PG_CONN" ]; then
  printf "FAIL T05: PG_CONN / DATABASE_URL not set\n"
  exit 2
fi

query="SELECT count(*) FROM nclaw.conversation_topics WHERE topic_id IS NULL;"

count="$(psql "$PG_CONN" -tAc "$query" 2>/dev/null || echo FAIL)"

if [ "$count" = "FAIL" ]; then
  printf "FAIL T05: psql could not execute query (check PG_CONN + nclaw schema exists)\n"
  exit 1
fi

if [ "$count" != "0" ]; then
  # Also list the offending conversation ids for debugging.
  printf "FAIL T05: %s conversation_topics rows have NULL topic_id\n" "$count"
  psql "$PG_CONN" -tAc "SELECT conversation_id FROM nclaw.conversation_topics WHERE topic_id IS NULL LIMIT 20;" 2>/dev/null | sed 's/^/  null_topic_conv_id=/'
  exit 1
fi

printf "OK T05: 0 NULL topics in nclaw.conversation_topics\n"
