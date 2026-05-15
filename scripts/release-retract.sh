#!/usr/bin/env bash
# release-retract.sh
#
# Retract a previously published nself CLI release.
# Created under P101 S8.T05 to pair with the release denylist + override flow.
#
# Behaviors (in order):
#   1. Mark the GitHub release as draft (gh release edit <tag> --draft)
#   2. Prepend a RETRACTED notice to the release notes
#   3. Optionally delete the git tag locally and on remote (--delete-tag)
#   4. Append a retraction record to audit/release-retractions.log
#   5. Emit a cascade-rollback hint listing inboxes that should receive a PCI
#
# Usage:
#   bash scripts/release-retract.sh <tag> [--delete-tag] [--reason "<text>"]
#
# Examples:
#   bash scripts/release-retract.sh v1.0.13 --reason "checksums mismatch in darwin-arm64 tarball"
#   bash scripts/release-retract.sh v1.0.14 --delete-tag --reason "leaked credential in dist artifact"
#
# Notes:
#   - Always asks for double confirmation before any destructive action.
#   - Tag deletion (--delete-tag) is GUARDED — falls under the destructive
#     deny-list and the operator must type DELETE to confirm.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
AUDIT_DIR="${REPO_ROOT}/audit"
AUDIT_LOG="${AUDIT_DIR}/release-retractions.log"

# ── Argument parsing ─────────────────────────────────────────────────────────
if [ $# -lt 1 ]; then
  echo "ERROR: usage: $0 <tag> [--delete-tag] [--reason \"<text>\"]" >&2
  exit 2
fi

TAG="$1"
shift

DELETE_TAG=0
REASON=""
while [ $# -gt 0 ]; do
  case "$1" in
    --delete-tag) DELETE_TAG=1; shift ;;
    --reason)     REASON="${2:-}"; shift 2 ;;
    *)            echo "ERROR: unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [ -z "${REASON}" ]; then
  if [ -t 0 ]; then
    printf "Reason for retraction (required, will be logged): " >&2
    read -r REASON
  fi
fi
if [ -z "${REASON}" ]; then
  echo "ERROR: --reason is required (or set RELEASE_RETRACT_REASON env var)" >&2
  exit 2
fi

# ── Pre-checks ───────────────────────────────────────────────────────────────
if ! command -v gh >/dev/null 2>&1; then
  echo "ERROR: gh CLI not found" >&2
  exit 2
fi

if ! gh release view "${TAG}" >/dev/null 2>&1; then
  echo "ERROR: no GitHub release found for tag ${TAG}" >&2
  exit 2
fi

log_info() { printf "[release-retract] %s\n" "$*"; }
log_warn() { printf "[release-retract] WARN: %s\n" "$*" >&2; }

# ── Confirmation ─────────────────────────────────────────────────────────────
log_warn "About to retract release: ${TAG}"
log_warn "Reason: ${REASON}"
log_warn "Delete tag (local + remote): $([ "${DELETE_TAG}" -eq 1 ] && echo YES || echo NO)"

if [ -t 0 ]; then
  printf "Type RETRACT to confirm: " >&2
  read -r CONFIRM
  if [ "${CONFIRM}" != "RETRACT" ]; then
    echo "Aborted by operator." >&2
    exit 1
  fi
  if [ "${DELETE_TAG}" -eq 1 ]; then
    printf "Tag deletion is destructive. Type DELETE to confirm: " >&2
    read -r CONFIRM2
    if [ "${CONFIRM2}" != "DELETE" ]; then
      echo "Tag deletion aborted; proceeding with draft + note update only." >&2
      DELETE_TAG=0
    fi
  fi
fi

# ── Step 1: Mark release as draft ────────────────────────────────────────────
log_info "marking release ${TAG} as draft"
gh release edit "${TAG}" --draft

# ── Step 2: Prepend RETRACTED notice to release notes ────────────────────────
log_info "fetching current release notes"
CURRENT_NOTES=$(gh release view "${TAG}" --json body -q '.body' 2>/dev/null || echo "")
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
NEW_NOTES=$(printf "## RETRACTED %s\n\n**Reason:** %s\n\n---\n\n%s" \
  "${TIMESTAMP}" "${REASON}" "${CURRENT_NOTES}")
log_info "prepending RETRACTED notice"
printf "%s" "${NEW_NOTES}" | gh release edit "${TAG}" --notes-file -

# ── Step 3: Optional tag deletion ────────────────────────────────────────────
if [ "${DELETE_TAG}" -eq 1 ]; then
  log_warn "deleting tag ${TAG} locally"
  git -C "${REPO_ROOT}" tag -d "${TAG}" || true
  log_warn "deleting tag ${TAG} on origin"
  git -C "${REPO_ROOT}" push origin ":refs/tags/${TAG}" || log_warn "remote tag delete failed (may already be gone)"
fi

# ── Step 4: Audit log ────────────────────────────────────────────────────────
mkdir -p "${AUDIT_DIR}"
GIT_SHA=$(git -C "${REPO_ROOT}" rev-parse HEAD)
OPERATOR="${GITHUB_ACTOR:-${USER:-unknown}}"
printf "%s | sha=%s | tag=%s | operator=%s | deleted_tag=%s | reason=%s\n" \
  "${TIMESTAMP}" "${GIT_SHA}" "${TAG}" "${OPERATOR}" "${DELETE_TAG}" "${REASON}" \
  >> "${AUDIT_LOG}"
log_info "retraction logged to ${AUDIT_LOG}"

# ── Step 5: Cascade-rollback hint ────────────────────────────────────────────
cat <<EOF

==============================================================================
RETRACTION COMPLETE: ${TAG}
==============================================================================
Next steps (manual — file PCIs to each affected inbox):

  1. homebrew-nself:  ~/Sites/nself/.claude/inbox/ (slug: hotfix-revert-formula-${TAG})
     → Roll formula back to prior version
  2. plugins-pro:     ~/Sites/nself/.claude/inbox/ (slug: hotfix-pin-prior-cli-${TAG})
     → Update integration tests + pinned CLI version
  3. admin:           ~/Sites/nself/.claude/inbox/ (slug: hotfix-cli-lockstep-${TAG})
     → Roll admin Docker image tag back if lockstep was bumped
  4. web/docs:        ~/Sites/nself/.claude/inbox/ (slug: hotfix-docs-retract-${TAG})
     → Add deprecation notice + redirect installer to prior version
  5. web/org:         ~/Sites/nself/.claude/inbox/ (slug: hotfix-changelog-retract-${TAG})
     → Mark changelog entry RETRACTED + reason
  6. ping_api:        Confirm license-validate route still resolves prior CLI

Use: pci-send nself <slug> high hotfix "..."
==============================================================================
EOF
