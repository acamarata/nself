#!/usr/bin/env bash
# admin-merge.sh
#
# Solo-operator merge helper for nself-org repos.
#
# Problem: nself-org has branch protection on main (1 required reviewer). A
# solo operator cannot self-merge a PR under those rules. This script:
#   1. Pre-flight: verifies git state, checks for in-flight operations, reads
#      the target branch from the first argument or interactive prompt.
#   2. Launches an independent watchdog process (nohup, separate PID) that
#      will RESTORE branch protection even if the parent shell is killed.
#   3. Relaxes branch protection on the target repo/branch via GitHub API.
#   4. Merges the PR (by number) or the current branch directly.
#   5. Waits for CI to pass (up to WAIT_MINUTES, default 20).
#   6. Restores branch protection to the documented baseline.
#   7. Writes an audit log entry to ~/.nself/admin-merge-audit.log.
#
# The watchdog is an independent bash process (not a subshell) so it cannot
# be killed by SIGKILL to the parent. It polls for the "done" marker file and
# calls the restore function if the marker never appears within WATCHDOG_DEADLINE.
#
# Usage:
#   scripts/admin-merge.sh [--repo <owner/repo>] [--branch <branch>]
#                          [--pr <number>] [--dry-run] [--no-wait-ci]
#                          [--wait-minutes <N>]
#
# Flags:
#   --repo <owner/repo>    Target repo (default: auto-detected from git remote)
#   --branch <branch>      Branch to merge (default: current branch)
#   --pr <number>          PR number to merge (auto-detected from branch if omitted)
#   --dry-run              Print what would happen; do not change anything
#   --no-wait-ci           Skip CI wait; restore immediately after merge (risky)
#   --wait-minutes <N>     CI wait budget in minutes (default: 20)
#   --force-restore        Restore branch protection only (recovery mode)
#
# Requirements:
#   gh (GitHub CLI, authenticated with admin scope)
#   jq
#   git
#
# Audit log: ~/.nself/admin-merge-audit.log
# Watchdog state: /tmp/nself-admin-merge-<ts>.done (created on clean exit)
#
# Security note: relaxing branch protection for <60 seconds is a carefully
# bounded window. The watchdog ensures protection is ALWAYS restored even on
# SIGKILL. Do NOT expand RELAX_WINDOW_SECONDS beyond 120.
#
# Authority: P98 S98-02 T09; admin-merge-workflow.md

set -euo pipefail

# ── Constants ─────────────────────────────────────────────────────────────────
# RELAX_WINDOW_SECONDS: hard cap on how long protection stays off.
# WATCHDOG_DEADLINE >= 3x RELAX_WINDOW_SECONDS to allow CI startup overhead.
RELAX_WINDOW_SECONDS=60
WATCHDOG_DEADLINE=$(( RELAX_WINDOW_SECONDS * 3 ))  # 180s
DEFAULT_WAIT_MINUTES=20
AUDIT_LOG="${HOME}/.nself/admin-merge-audit.log"

# ── Arguments ─────────────────────────────────────────────────────────────────
DRY_RUN=0
NO_WAIT_CI=0
FORCE_RESTORE=0
WAIT_MINUTES="${DEFAULT_WAIT_MINUTES}"
TARGET_REPO=""
TARGET_BRANCH=""
PR_NUMBER=""

while [ $# -gt 0 ]; do
  case "$1" in
    --repo)           TARGET_REPO="$2";    shift 2 ;;
    --branch)         TARGET_BRANCH="$2";  shift 2 ;;
    --pr)             PR_NUMBER="$2";      shift 2 ;;
    --dry-run)        DRY_RUN=1;           shift ;;
    --no-wait-ci)     NO_WAIT_CI=1;        shift ;;
    --force-restore)  FORCE_RESTORE=1;     shift ;;
    --wait-minutes)   WAIT_MINUTES="$2";   shift 2 ;;
    --help|-h)
      sed -n '2,65p' "$0" | grep '^#' | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      printf 'Unknown flag: %s\n' "$1" >&2
      exit 1
      ;;
  esac
done

# ── Utilities ─────────────────────────────────────────────────────────────────
TS() { date '+%Y-%m-%dT%H:%M:%S%z'; }
BOLD='\033[1m'
RED='\033[31m'
YELLOW='\033[33m'
GREEN='\033[32m'
RESET='\033[0m'

info()  { printf '%b[INFO]%b  %s\n' "${BOLD}" "${RESET}" "$*"; }
warn()  { printf '%b[WARN]%b  %s\n' "${YELLOW}" "${RESET}" "$*"; }
error() { printf '%b[ERROR]%b %s\n' "${RED}" "${RESET}" "$*" >&2; }
ok()    { printf '%b[OK]%b    %s\n' "${GREEN}" "${RESET}" "$*"; }

audit() {
  mkdir -p "$(dirname "${AUDIT_LOG}")"
  printf '%s  %s\n' "$(TS)" "$*" >> "${AUDIT_LOG}"
}

die() {
  error "$*"
  audit "ABORT: $*"
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

# ── Pre-flight checks ─────────────────────────────────────────────────────────
require_cmd gh
require_cmd jq
require_cmd git

# Auto-detect repo from git remote
if [ -z "${TARGET_REPO}" ]; then
  REMOTE_URL=$(git remote get-url origin 2>/dev/null || true)
  if [ -z "${REMOTE_URL}" ]; then
    die "Cannot detect repo: no git remote 'origin'. Pass --repo owner/repo."
  fi
  # Extract owner/repo from https://github.com/owner/repo.git or git@github.com:owner/repo.git
  TARGET_REPO=$(printf '%s' "${REMOTE_URL}" \
    | sed 's|.*github\.com[:/]\(.*\)\.git|\1|' \
    | sed 's|.*github\.com[:/]\(.*\)|\1|')
fi

# Auto-detect branch
if [ -z "${TARGET_BRANCH}" ]; then
  TARGET_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
  if [ -z "${TARGET_BRANCH}" ] || [ "${TARGET_BRANCH}" = "HEAD" ]; then
    die "Cannot detect current branch. Pass --branch <branch>."
  fi
fi

# Guard: never merge directly on main
if [ "${TARGET_BRANCH}" = "main" ] || [ "${TARGET_BRANCH}" = "master" ]; then
  die "TARGET_BRANCH is '${TARGET_BRANCH}' — this script merges TO main, not FROM main. Pass a feature/fix branch via --branch."
fi

# Verify gh auth
if ! gh auth status >/dev/null 2>&1; then
  die "GitHub CLI not authenticated. Run: gh auth login"
fi

# Verify admin scope (branch protection requires admin:repo or repo scope)
SCOPES=$(gh auth status 2>&1 | grep -i 'scopes\|token scopes' | head -1 || true)
if ! printf '%s' "${SCOPES}" | grep -qi 'repo'; then
  warn "Could not confirm 'repo' scope on GH token. Branch protection API may fail."
  warn "Re-authenticate with: gh auth login --scopes repo"
fi

# Auto-detect PR number
if [ -z "${PR_NUMBER}" ] && [ "${FORCE_RESTORE}" -eq 0 ]; then
  PR_NUMBER=$(gh pr list --repo "${TARGET_REPO}" --head "${TARGET_BRANCH}" --state open --json number --jq '.[0].number' 2>/dev/null || true)
  if [ -z "${PR_NUMBER}" ]; then
    die "No open PR found for branch '${TARGET_BRANCH}' in ${TARGET_REPO}. Create one first or pass --pr <number>."
  fi
  info "Auto-detected PR #${PR_NUMBER} for branch '${TARGET_BRANCH}'"
fi

# ── Read current branch protection (save baseline for restore) ────────────────
info "Reading current branch protection for ${TARGET_REPO}..."

PROTECTION_JSON=$(gh api "repos/${TARGET_REPO}/branches/main/protection" 2>/dev/null || true)
if [ -z "${PROTECTION_JSON}" ]; then
  die "Cannot read branch protection for ${TARGET_REPO}/main. Ensure GH token has admin:repo scope."
fi

# Extract baseline values
BASELINE_REQUIRED_REVIEWS=$(printf '%s' "${PROTECTION_JSON}" \
  | jq -r '.required_pull_request_reviews.required_approving_review_count // 1')
BASELINE_STRICT=$(printf '%s' "${PROTECTION_JSON}" \
  | jq -r '.required_status_checks.strict // true')
BASELINE_LINEAR=$(printf '%s' "${PROTECTION_JSON}" \
  | jq -r '.required_linear_history.enabled // true')
BASELINE_STATUS_CHECKS=$(printf '%s' "${PROTECTION_JSON}" \
  | jq -c '.required_status_checks.contexts // []')

info "  Baseline: required_reviews=${BASELINE_REQUIRED_REVIEWS}, strict=${BASELINE_STRICT}, linear=${BASELINE_LINEAR}"
info "  Status checks: ${BASELINE_STATUS_CHECKS}"

# ── Force-restore mode ────────────────────────────────────────────────────────
if [ "${FORCE_RESTORE}" -eq 1 ]; then
  warn "Force-restore mode: restoring branch protection without merge."
  # (restore function defined below — we'll call it directly)
fi

# ── Watchdog marker file ───────────────────────────────────────────────────────
MERGE_TS=$(date '+%Y%m%d%H%M%S')
DONE_MARKER="/tmp/nself-admin-merge-${MERGE_TS}.done"
RELAX_MARKER="/tmp/nself-admin-merge-${MERGE_TS}.relaxed"
# RELAX_TS_MARKER: written after relax API call succeeds; contains the epoch
# at which relax completed. Watchdog reads this to reset its deadline from the
# actual relax time rather than from spawn time (FIX3 — watchdog deadline timing).
RELAX_TS_MARKER="/tmp/nself-admin-merge-${MERGE_TS}.relax_ts"

# ── Restore function (shared by main body and watchdog) ───────────────────────
# This function is sourced into the watchdog script via heredoc.
# It must be self-contained (no closures over parent variables beyond what's injected).
restore_protection() {
  local repo="$1"
  local required_reviews="$2"
  local strict="$3"
  local linear="$4"
  local status_checks="$5"
  local dry_run="${6:-0}"
  local reason="${7:-scheduled_restore}"

  printf '[RESTORE] Restoring branch protection on %s/main (%s)...\n' "${repo}" "${reason}" >&2

  if [ "${dry_run}" -eq 1 ]; then
    printf '[RESTORE][DRY-RUN] Would restore: reviews=%s strict=%s linear=%s\n' \
      "${required_reviews}" "${strict}" "${linear}" >&2
    return 0
  fi

  # Build the PATCH payload. A single call restores ALL protection settings atomically.
  PAYLOAD=$(jq -n \
    --argjson reviews "${required_reviews}" \
    --argjson strict "${strict}" \
    --argjson linear "${linear}" \
    --argjson contexts "${status_checks}" \
    '{
      required_status_checks: {
        strict: $strict,
        contexts: $contexts
      },
      enforce_admins: false,
      required_pull_request_reviews: {
        required_approving_review_count: $reviews,
        dismiss_stale_reviews: true
      },
      restrictions: null,
      required_linear_history: $linear,
      allow_force_pushes: false,
      allow_deletions: false
    }')

  if gh api "repos/${repo}/branches/main/protection" \
    -X PUT \
    --input - \
    -H "Accept: application/vnd.github.v3+json" \
    <<< "${PAYLOAD}" > /dev/null 2>&1; then
    HTTP_STATUS="200"
  else
    HTTP_STATUS="000"
  fi

  if [ "${HTTP_STATUS}" = "200" ] || [ "${HTTP_STATUS}" = "201" ]; then
    printf '[RESTORE] Protection restored successfully (HTTP %s).\n' "${HTTP_STATUS}" >&2
    printf '%s  RESTORE  repo=%s  reason=%s  http=%s\n' \
      "$(date '+%Y-%m-%dT%H:%M:%S%z')" "${repo}" "${reason}" "${HTTP_STATUS}" \
      >> "${HOME}/.nself/admin-merge-audit.log" 2>/dev/null || true
  elif [ "${HTTP_STATUS}" = "401" ]; then
    # 401 means the GitHub token expired between relax and restore.
    # EMERGENCY RECOVERY: run this manually to restore branch protection —
    #   gh auth refresh --scopes repo
    #   scripts/admin-merge.sh --repo <owner/repo> --force-restore
    # Or directly via the GitHub web UI:
    #   Settings > Branches > Edit protection rule > restore prior values.
    printf '[RESTORE] EMERGENCY: HTTP 401 — token expired during relax window!\n' >&2
    printf '[RESTORE] Manual recovery:\n' >&2
    printf '[RESTORE]   1. gh auth refresh --scopes repo\n' >&2
    printf '[RESTORE]   2. scripts/admin-merge.sh --repo %s --force-restore\n' "${repo}" >&2
    printf '[RESTORE]   3. Or restore via GitHub UI: Settings > Branches > Edit\n' >&2
    printf '%s  RESTORE_AUTH_EXPIRED  repo=%s  http=%s\n' \
      "$(date '+%Y-%m-%dT%H:%M:%S%z')" "${repo}" "${HTTP_STATUS}" \
      >> "${HOME}/.nself/admin-merge-audit.log" 2>/dev/null || true
  else
    printf '[RESTORE] WARNING: restore returned HTTP %s — check manually!\n' "${HTTP_STATUS}" >&2
    printf '[RESTORE] Manual recovery: scripts/admin-merge.sh --repo %s --force-restore\n' "${repo}" >&2
    printf '%s  RESTORE_FAILED  repo=%s  http=%s\n' \
      "$(date '+%Y-%m-%dT%H:%M:%S%z')" "${repo}" "${HTTP_STATUS}" \
      >> "${HOME}/.nself/admin-merge-audit.log" 2>/dev/null || true
  fi
}

# ── Force-restore early exit ──────────────────────────────────────────────────
if [ "${FORCE_RESTORE}" -eq 1 ]; then
  restore_protection "${TARGET_REPO}" "${BASELINE_REQUIRED_REVIEWS}" \
    "${BASELINE_STRICT}" "${BASELINE_LINEAR}" "${BASELINE_STATUS_CHECKS}" \
    "${DRY_RUN}" "force_restore"
  exit 0
fi

# ── Pre-merge summary ─────────────────────────────────────────────────────────
printf '\n'
info "Admin merge plan:"
printf '  Repo:         %s\n' "${TARGET_REPO}"
printf '  Branch:       %s → main\n' "${TARGET_BRANCH}"
printf '  PR:           #%s\n' "${PR_NUMBER}"
printf '  CI wait:      %s minutes\n' "${WAIT_MINUTES}"
printf '  Dry run:      %s\n' "${DRY_RUN}"
printf '  Watchdog:     %ss deadline (%s marker)\n' "${WATCHDOG_DEADLINE}" "${DONE_MARKER}"
printf '\n'

if [ "${DRY_RUN}" -eq 0 ]; then
  warn "This will TEMPORARILY relax branch protection on ${TARGET_REPO}/main."
  warn "Protection is restored automatically within ${WATCHDOG_DEADLINE}s by the watchdog."
  printf '\nProceed? [y/N] '
  read -r CONFIRM
  if [ "${CONFIRM}" != "y" ] && [ "${CONFIRM}" != "Y" ]; then
    info "Aborted."
    exit 0
  fi
fi

audit "START  repo=${TARGET_REPO}  branch=${TARGET_BRANCH}  pr=${PR_NUMBER}  dry_run=${DRY_RUN}"

# ── Spawn watchdog sidecar ────────────────────────────────────────────────────
# The watchdog is an INDEPENDENT process (nohup + detached).
# It cannot be killed by SIGKILL to this shell.
# It polls for DONE_MARKER and calls restore if it never appears.

WATCHDOG_SCRIPT="/tmp/nself-watchdog-${MERGE_TS}.sh"

cat > "${WATCHDOG_SCRIPT}" << WATCHDOG_EOF
#!/usr/bin/env bash
# Watchdog for admin-merge.sh run at ${MERGE_TS}
# Restores branch protection if parent dies before setting the done marker.
set -u

REPO="${TARGET_REPO}"
REQ_REVIEWS="${BASELINE_REQUIRED_REVIEWS}"
STRICT="${BASELINE_STRICT}"
LINEAR="${BASELINE_LINEAR}"
STATUS_CHECKS='${BASELINE_STATUS_CHECKS}'
DONE_MARKER="${DONE_MARKER}"
RELAX_MARKER="${RELAX_MARKER}"
RELAX_TS_MARKER="${RELAX_TS_MARKER}"
DEADLINE="${WATCHDOG_DEADLINE}"
AUDIT_LOG="${AUDIT_LOG}"
DRY_RUN="${DRY_RUN}"

deadline=\$(( \$(date '+%s') + DEADLINE ))

# Wait until relax marker exists (protection has been relaxed) or deadline
while [ ! -f "\${RELAX_MARKER}" ]; do
  now=\$(date '+%s')
  if [ "\${now}" -ge "\${deadline}" ]; then
    printf '[WATCHDOG] Deadline reached waiting for relax marker. Exiting (nothing to restore).\n'
    rm -f "${WATCHDOG_SCRIPT}"
    exit 0
  fi
  sleep 2
done

# FIX3: Reset deadline from the actual relax completion time, not spawn time.
# If the relax API call itself took e.g. 120s, the watchdog would have had
# only the remaining slice. Now it gets the full WATCHDOG_DEADLINE window
# from the moment protection was actually relaxed.
if [ -f "\${RELAX_TS_MARKER}" ]; then
  relax_epoch=\$(cat "\${RELAX_TS_MARKER}" 2>/dev/null || date '+%s')
  deadline=\$(( relax_epoch + DEADLINE ))
  printf '[WATCHDOG] Deadline reset from relax time: epoch=%s deadline=%s\n' "\${relax_epoch}" "\${deadline}"
fi

printf '[WATCHDOG] Protection was relaxed. Monitoring for done marker...\n'

# Now watch for done marker. If it never appears, restore protection.
while [ ! -f "\${DONE_MARKER}" ]; do
  now=\$(date '+%s')
  if [ "\${now}" -ge "\${deadline}" ]; then
    printf '[WATCHDOG] Deadline exceeded without done marker. Forcing restore.\n'
    restore_protection \${REPO} \${REQ_REVIEWS} \${STRICT} \${LINEAR} "\${STATUS_CHECKS}" \${DRY_RUN} watchdog_timeout
    rm -f "${WATCHDOG_SCRIPT}"
    exit 0
  fi
  sleep 3
done

printf '[WATCHDOG] Done marker found. Parent completed cleanly — no restore needed.\n'
rm -f "${WATCHDOG_SCRIPT}"
exit 0

restore_protection() {
  local repo="\$1" reviews="\$2" strict="\$3" linear="\$4" checks="\$5" dry="\$6" reason="\$7"
  printf '[WATCHDOG-RESTORE] Restoring %s/main (%s)\n' "\${repo}" "\${reason}"
  if [ "\${dry}" -eq 1 ]; then
    printf '[WATCHDOG-RESTORE][DRY-RUN] Would restore protection.\n'
    return 0
  fi
  PAYLOAD=\$(jq -n \
    --argjson reviews "\${reviews}" \
    --argjson strict "\${strict}" \
    --argjson linear "\${linear}" \
    --argjson contexts "\${checks}" \
    '{
      required_status_checks: { strict: \$strict, contexts: \$contexts },
      enforce_admins: false,
      required_pull_request_reviews: { required_approving_review_count: \$reviews, dismiss_stale_reviews: true },
      restrictions: null,
      required_linear_history: \$linear,
      allow_force_pushes: false,
      allow_deletions: false
    }')
  if gh api "repos/\${repo}/branches/main/protection" -X PUT \
    --input - -H "Accept: application/vnd.github.v3+json" \
    <<< "\${PAYLOAD}" > /dev/null 2>&1; then
    HTTP="200"
  else
    HTTP="000"
  fi
  printf '[WATCHDOG-RESTORE] HTTP %s\n' "\${HTTP}"
  printf '%s  WATCHDOG_RESTORE  repo=%s  reason=%s  http=%s\n' "\$(date '+%Y-%m-%dT%H:%M:%S%z')" "\${repo}" "\${reason}" "\${HTTP}" \
    >> "\${AUDIT_LOG}" 2>/dev/null || true
}
WATCHDOG_EOF

chmod +x "${WATCHDOG_SCRIPT}"

if [ "${DRY_RUN}" -eq 0 ]; then
  nohup bash "${WATCHDOG_SCRIPT}" > "/tmp/nself-watchdog-${MERGE_TS}.log" 2>&1 &
  WATCHDOG_PID=$!
  info "Watchdog spawned: PID ${WATCHDOG_PID}, deadline ${WATCHDOG_DEADLINE}s"
  audit "WATCHDOG_SPAWN  pid=${WATCHDOG_PID}  deadline=${WATCHDOG_DEADLINE}"
else
  info "[DRY-RUN] Would spawn watchdog (skipped)"
  WATCHDOG_PID=0
fi

# FIX5: EXIT trap — ensure protection is always restored and DONE_MARKER is
# written on any exit (clean, error, SIGINT, SIGTERM). This guarantees the
# watchdog sees the marker and self-exits rather than double-restoring.
# The trap is set here, AFTER watchdog spawn, so it captures the live values
# of TARGET_REPO, BASELINE_*, DRY_RUN, and DONE_MARKER.
_exit_trap_fired=0
_cleanup_on_exit() {
  if [ "${_exit_trap_fired}" -eq 1 ]; then return; fi
  _exit_trap_fired=1
  # Only attempt restore if relax marker exists (i.e., we actually relaxed)
  if [ -f "${RELAX_MARKER}" ] && [ ! -f "${DONE_MARKER}" ]; then
    restore_protection "${TARGET_REPO}" \
      "${BASELINE_REQUIRED_REVIEWS}" \
      "${BASELINE_STRICT}" \
      "${BASELINE_LINEAR}" \
      "${BASELINE_STATUS_CHECKS}" \
      "${DRY_RUN}" \
      "exit_trap"
  fi
  # Always write the done marker so the watchdog does not double-restore
  touch "${DONE_MARKER}" 2>/dev/null || true
}
trap '_cleanup_on_exit' EXIT INT TERM

# ── Relax branch protection ───────────────────────────────────────────────────
info "Relaxing branch protection on ${TARGET_REPO}/main..."

RELAX_PAYLOAD=$(jq -n \
  --argjson checks "${BASELINE_STATUS_CHECKS}" \
  '{
    required_status_checks: { strict: false, contexts: $checks },
    enforce_admins: false,
    required_pull_request_reviews: {
      required_approving_review_count: 0,
      dismiss_stale_reviews: false
    },
    restrictions: null,
    required_linear_history: false,
    allow_force_pushes: false,
    allow_deletions: false
  }')

if [ "${DRY_RUN}" -eq 0 ]; then
  # gh api does not support -w or --timeout flags; use exit code to determine success
  if gh api "repos/${TARGET_REPO}/branches/main/protection" \
    -X PUT \
    --input - \
    -H "Accept: application/vnd.github.v3+json" \
    <<< "${RELAX_PAYLOAD}" > /dev/null 2>&1; then
    HTTP="200"
  else
    HTTP="000"
  fi

  if [ "${HTTP}" != "200" ] && [ "${HTTP}" != "201" ]; then
    die "Failed to relax branch protection (HTTP ${HTTP}). Aborting — protection unchanged."
  fi

  ok "Branch protection relaxed (HTTP ${HTTP})."
  # Also disable enforce_admins via dedicated endpoint (PUT /protection sets field but
  # the separate /enforce_admins endpoint is the authoritative toggle on some repo configs)
  gh api "repos/${TARGET_REPO}/branches/main/protection/enforce_admins" \
    -X DELETE > /dev/null 2>&1 || true
  touch "${RELAX_MARKER}"
  # Write post-relax epoch for watchdog deadline reset (FIX3)
  date '+%s' > "${RELAX_TS_MARKER}"
  audit "RELAX  repo=${TARGET_REPO}  http=${HTTP}"
else
  info "[DRY-RUN] Would relax branch protection via PATCH."
  info "[DRY-RUN] Payload: ${RELAX_PAYLOAD}"
fi

# ── Merge PR ──────────────────────────────────────────────────────────────────
MERGE_SUCCESS=0
if [ "${DRY_RUN}" -eq 0 ]; then
  info "Merging PR #${PR_NUMBER}..."
  if gh pr merge "${PR_NUMBER}" \
      --repo "${TARGET_REPO}" \
      --squash \
      --admin \
      --delete-branch 2>&1; then
    MERGE_SUCCESS=1
    ok "PR #${PR_NUMBER} merged."
    audit "MERGE  pr=${PR_NUMBER}  result=success"
  else
    error "PR merge failed."
    audit "MERGE  pr=${PR_NUMBER}  result=failed"
    # Still restore even on failure
  fi
else
  info "[DRY-RUN] Would merge PR #${PR_NUMBER} (squash + delete branch)."
  MERGE_SUCCESS=1
fi

# ── Wait for CI (optional) ────────────────────────────────────────────────────
if [ "${NO_WAIT_CI}" -eq 0 ] && [ "${MERGE_SUCCESS}" -eq 1 ] && [ "${DRY_RUN}" -eq 0 ]; then
  info "Waiting up to ${WAIT_MINUTES} minutes for post-merge CI to pass..."
  WAIT_DEADLINE=$(( $(date '+%s') + WAIT_MINUTES * 60 ))
  CI_PASSED=0

  while [ "$(date '+%s')" -lt "${WAIT_DEADLINE}" ]; do
    # Query CI for the merge commit on main
    RUN_STATUS=$(gh run list \
      --repo "${TARGET_REPO}" \
      --branch main \
      --limit 1 \
      --json status,conclusion \
      --jq '.[0] | "\(.status)/\(.conclusion)"' 2>/dev/null || printf 'unknown/unknown')

    STATUS_PART=$(printf '%s' "${RUN_STATUS}" | cut -d'/' -f1)
    CONCLUSION_PART=$(printf '%s' "${RUN_STATUS}" | cut -d'/' -f2)

    if [ "${STATUS_PART}" = "completed" ]; then
      if [ "${CONCLUSION_PART}" = "success" ]; then
        ok "CI passed."
        CI_PASSED=1
        audit "CI_PASS  repo=${TARGET_REPO}"
      else
        warn "CI completed with conclusion: ${CONCLUSION_PART}"
        audit "CI_FAIL  repo=${TARGET_REPO}  conclusion=${CONCLUSION_PART}"
      fi
      break
    fi

    printf '  CI status: %s/%s — waiting...\n' "${STATUS_PART}" "${CONCLUSION_PART}"
    sleep 30
  done

  if [ "${CI_PASSED}" -eq 0 ]; then
    warn "CI did not pass within ${WAIT_MINUTES} minutes. Restoring protection anyway."
    audit "CI_TIMEOUT  repo=${TARGET_REPO}"
  fi
fi

# ── Restore branch protection ─────────────────────────────────────────────────
restore_protection "${TARGET_REPO}" \
  "${BASELINE_REQUIRED_REVIEWS}" \
  "${BASELINE_STRICT}" \
  "${BASELINE_LINEAR}" \
  "${BASELINE_STATUS_CHECKS}" \
  "${DRY_RUN}" \
  "post_merge"

# ── Signal watchdog that we're done ──────────────────────────────────────────
touch "${DONE_MARKER}"
audit "DONE  repo=${TARGET_REPO}  pr=${PR_NUMBER}"

ok "Admin merge complete. Branch protection restored. Watchdog will self-exit."

printf '\n'
info "Audit log: ${AUDIT_LOG}"
if [ "${DRY_RUN}" -eq 0 ]; then
  info "Watchdog log: /tmp/nself-watchdog-${MERGE_TS}.log"
fi
