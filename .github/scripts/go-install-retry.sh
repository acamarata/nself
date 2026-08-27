#!/usr/bin/env bash
# go-install-retry.sh — install a pinned Go tool, retrying transient failures.
#
# Purpose: `go install <tool>@<version>` in CI depends on proxy.golang.org and
#   sum.golang.org being reachable. Both are outside our control and both fail
#   intermittently. On 2026-08-27 four separate jobs died on the same thing:
#
#     go: github.com/google/go-licenses@v1.6.0: version constraints conflict:
#       ... verifying go.mod: reading https://sum.golang.org/tile/8/0/x002/691:
#       stream error: stream ID 483; INTERNAL_ERROR; received from peer
#
#   Different tiles, different modules, different jobs — the tool never changed.
#   One of those took 8 minutes to fail, so a red gate cost 8 minutes and told
#   nobody anything true.
#
# Inputs:  $1 = module@version (must be pinned; see below). $2 = attempts (default 3).
# Outputs: the tool on $GOBIN/$GOPATH/bin, or a non-zero exit after N attempts.
# Constraints: checksum verification stays ON. This retries the download; it does
#   NOT set GOFLAGS=-mod=mod, GONOSUMDB, GONOSUMCHECK or GOPRIVATE, because
#   skipping verification to dodge a flaky verifier is how a supply-chain gate
#   becomes decorative. A genuine checksum mismatch still fails on attempt 1 and
#   on every retry.
set -euo pipefail

TOOL="${1:?usage: go-install-retry.sh <module@version> [attempts]}"
ATTEMPTS="${2:-3}"

# Refuse @latest and unpinned specs: retrying an unpinned install can silently
# install a different version on attempt 2 than attempt 1 resolved.
case "$TOOL" in
  *@latest|*@master|*@main) echo "refusing to install unpinned tool: $TOOL" >&2; exit 2 ;;
  *@*) : ;;
  *) echo "tool must be pinned as module@version, got: $TOOL" >&2; exit 2 ;;
esac

for i in $(seq 1 "$ATTEMPTS"); do
  if go install "$TOOL"; then
    [ "$i" -gt 1 ] && echo "::notice::installed $TOOL on attempt $i"
    exit 0
  fi
  if [ "$i" -lt "$ATTEMPTS" ]; then
    delay=$(( i * 15 ))
    echo "::warning::go install $TOOL failed (attempt $i/$ATTEMPTS); retrying in ${delay}s"
    sleep "$delay"
  fi
done

echo "::error::go install $TOOL failed after $ATTEMPTS attempts." >&2
echo "If the error mentions sum.golang.org or proxy.golang.org, it is upstream." >&2
echo "Checksum verification was NOT disabled — a real mismatch must not be retried away." >&2
exit 1
