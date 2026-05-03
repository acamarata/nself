#!/usr/bin/env bash
# bump-version.sh — Atomic version bump across all lockstep files.
#
# Usage:
#   ./scripts/bump-version.sh <new-version> [--dry-run] [--homebrew]
#
# <new-version>   Required. Target version like "1.0.14" or "v1.0.14" (v stripped).
# --dry-run       Print what would change; write nothing.
# --homebrew      Phase 2: update homebrew formula after GitHub release exists.
#                 Without this flag, the formula is skipped and a reminder printed.
#
# Files updated (11 + 1 optional homebrew):
#   1.  cli/.github/VERSION
#   2.  cli/internal/version/version.go       Version = "x.y.z"
#   3.  cli/sdk/go/doc.go                     const Version = "x.y.z"
#   4.  cli/sdk/ts/package.json               "version": "x.y.z"
#   5.  cli/sdk/py/pyproject.toml             version = "x.y.z" under [project]
#   6.  cli/sdk/flutter/pubspec.yaml          version: x.y.z
#   7.  admin/package.json                    "version": "x.y.z"
#   8.  admin/src/lib/cli-version.ts          CLI_VERSION = 'x.y.z'
#   9.  admin/Dockerfile                      ARG NSELF_VERSION=x.y.z
#   10. admin/Dockerfile                      ENV ADMIN_VERSION=x.y.z
#   11. admin/Dockerfile                      LABEL org.opencontainers.image.version="x.y.z"
# +12. homebrew-nself/Formula/nself.rb       url/sha256/version (--homebrew only)
#
# Exit codes:
#   0 — success (all changes applied, or all already at target in idempotent run)
#   1 — usage error, bad semver, missing file, or homebrew release not found
#
# Hard rules:
#   - Atomic writes: all file writes use tmp+rename, cleaned on EXIT
#   - Line-anchored regex: patterns match exact field, not arbitrary occurrences
#   - Idempotent: skip files already at the target version
#   - shellcheck-clean (no echo -e, no ${var,,}, no declare -A)

set -euo pipefail

# ── Argument parsing ───────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
NSELF_ROOT="$(cd "${REPO_ROOT}/.." && pwd)"

DRY_RUN=false
HOMEBREW=false
RAW_VERSION=""

for arg in "$@"; do
  case "${arg}" in
    --dry-run)   DRY_RUN=true ;;
    --homebrew)  HOMEBREW=true ;;
    -*)
      printf 'Unknown flag: %s\n' "${arg}" >&2
      printf 'Usage: %s <version> [--dry-run] [--homebrew]\n' "$0" >&2
      exit 1
      ;;
    *)
      if [ -n "${RAW_VERSION}" ]; then
        printf 'Unexpected extra argument: %s\n' "${arg}" >&2
        exit 1
      fi
      RAW_VERSION="${arg}"
      ;;
  esac
done

# ── Semver validation ─────────────────────────────────────────────────────────

if [ -z "${RAW_VERSION}" ]; then
  printf 'Usage: %s <version> [--dry-run] [--homebrew]\n' "$0" >&2
  exit 1
fi

# Strip leading 'v'
NEW_VERSION="${RAW_VERSION#v}"

if ! printf '%s' "${NEW_VERSION}" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  printf 'ERROR: Invalid semver: "%s"\n' "${RAW_VERSION}" >&2
  printf '  Expected X.Y.Z (e.g. 1.0.14) or vX.Y.Z (e.g. v1.0.14)\n' >&2
  exit 1
fi

# ── File paths ────────────────────────────────────────────────────────────────

VERSION_FILE="${REPO_ROOT}/.github/VERSION"
VERSION_GO="${REPO_ROOT}/internal/version/version.go"
SDK_GO="${REPO_ROOT}/sdk/go/doc.go"
SDK_TS="${REPO_ROOT}/sdk/ts/package.json"
SDK_PY="${REPO_ROOT}/sdk/py/pyproject.toml"
SDK_FLUTTER="${REPO_ROOT}/sdk/flutter/pubspec.yaml"
ADMIN_PKG="${NSELF_ROOT}/admin/package.json"
ADMIN_CLIVER="${NSELF_ROOT}/admin/src/lib/cli-version.ts"
ADMIN_DOCKERFILE="${NSELF_ROOT}/admin/Dockerfile"
HOMEBREW_FORMULA="${NSELF_ROOT}/homebrew-nself/Formula/nself.rb"

# ── Tmp file cleanup ───────────────────────────────────────────────────────────

TMPFILES=()
cleanup_tmpfiles() {
  for f in "${TMPFILES[@]+"${TMPFILES[@]}"}"; do
    rm -f "${f}"
  done
}
trap cleanup_tmpfiles EXIT

# ── Helper functions ──────────────────────────────────────────────────────────

# atomic_write <dest> <content-via-stdin>
#   Writes stdin to dest atomically using a tmp+rename pair.
#   In dry-run mode, writes to a scratch tmp and diffs, then removes.
atomic_write() {
  local dest="$1"
  local tmp="${dest}.bumpver.tmp"
  TMPFILES+=("${tmp}")

  cat > "${tmp}"

  if [ "${DRY_RUN}" = "true" ]; then
    diff -u "${dest}" "${tmp}" || true
    rm -f "${tmp}"
    return 0
  fi

  mv "${tmp}" "${dest}"
}

# already_at <file> <pattern>
#   Returns 0 if the file already contains the exact target pattern.
already_at() {
  grep -qF "$2" "$1"
}

# report <file> <old> <new>
report() {
  printf '  %-60s  %s -> %s\n' "$1" "$2" "$3"
}

# ── Per-file update functions ─────────────────────────────────────────────────

update_version_file() {
  local f="${VERSION_FILE}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(tr -d '[:space:]' < "${f}")"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  printf '%s\n' "${NEW_VERSION}" | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_version_go() {
  local f="${VERSION_GO}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E '^\s+Version\s+string\s*=' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^\(\s*Version\s*string\s*=\s*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW_VERSION}/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_sdk_go_doc() {
  local f="${SDK_GO}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E '^const Version = ' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^const Version = \"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/const Version = \"${NEW_VERSION}\"/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_sdk_ts_pkg() {
  local f="${SDK_TS}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E '^\s*"version"\s*:' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^\(\s*\"version\"\s*:\s*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW_VERSION}/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_sdk_py_pyproject() {
  local f="${SDK_PY}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E '^version = ' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  # Only the top-level [project] version line (^version = "..."), not build-system vars
  sed "s/^version = \"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/version = \"${NEW_VERSION}\"/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_sdk_flutter_pubspec() {
  local f="${SDK_FLUTTER}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E '^version: ' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^version: [0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/version: ${NEW_VERSION}/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_admin_pkg() {
  local f="${ADMIN_PKG}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E '^\s*"version"\s*:' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^\(\s*\"version\"\s*:\s*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW_VERSION}/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_admin_cliver() {
  local f="${ADMIN_CLIVER}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(grep -E "^export const CLI_VERSION = '" "${f}" | grep -oE "[0-9]+\.[0-9]+\.[0-9]+" | head -1)"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^export const CLI_VERSION = '[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*'/export const CLI_VERSION = '${NEW_VERSION}'/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_admin_dockerfile() {
  local f="${ADMIN_DOCKERFILE}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old_nself old_admin old_label
  old_nself="$(grep -E '^ARG NSELF_VERSION=' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  old_admin="$(grep -E '^ENV ADMIN_VERSION=' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
  old_label="$(grep -E '^LABEL org.opencontainers.image.version=' "${f}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"

  if [ "${old_nself}" = "${NEW_VERSION}" ] && [ "${old_admin}" = "${NEW_VERSION}" ] && [ "${old_label}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (all three lines already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi

  # Update all three lines in one pass
  sed \
    -e "s/^ARG NSELF_VERSION=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/ARG NSELF_VERSION=${NEW_VERSION}/" \
    -e "s/^ENV ADMIN_VERSION=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/ENV ADMIN_VERSION=${NEW_VERSION}/" \
    -e "s/^LABEL org\.opencontainers\.image\.version=\"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/LABEL org.opencontainers.image.version=\"${NEW_VERSION}\"/" \
    "${f}" | atomic_write "${f}"
  report "${f} (ARG NSELF_VERSION)" "${old_nself}" "${NEW_VERSION}"
  report "${f} (ENV ADMIN_VERSION)" "${old_admin}" "${NEW_VERSION}"
  report "${f} (LABEL image.version)" "${old_label}" "${NEW_VERSION}"
}

update_homebrew_formula() {
  local f="${HOMEBREW_FORMULA}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi

  local tarball_url="https://github.com/nself-org/cli/archive/refs/tags/v${NEW_VERSION}.tar.gz"

  if [ "${DRY_RUN}" = "true" ]; then
    printf '  [dry-run] Would fetch SHA256 from %s\n' "${tarball_url}"
    printf '  [dry-run] Would update url, sha256, and version lines in %s\n' "${f}"
    return
  fi

  printf '  Checking release exists: %s\n' "${tarball_url}"
  local http_code
  http_code="$(curl -o /dev/null -sL -w '%{http_code}' "${tarball_url}")"
  if [ "${http_code}" != "200" ]; then
    printf 'ERROR: Release not found on GitHub (HTTP %s).\n' "${http_code}" >&2
    printf '  Push the tag and wait for the GitHub release before running --homebrew.\n' >&2
    printf '  URL checked: %s\n' "${tarball_url}" >&2
    exit 1
  fi

  printf '  Computing SHA256 (downloading tarball)...\n'
  local sha256
  sha256="$(curl -sL "${tarball_url}" | shasum -a 256 | cut -d' ' -f1)"
  if [ -z "${sha256}" ]; then
    printf 'ERROR: Could not compute SHA256 for %s\n' "${tarball_url}" >&2
    exit 1
  fi
  printf '  SHA256: %s\n' "${sha256}"

  local old_ver
  old_ver="$(grep -E '^\s*version\s+"' "${f}" | sed 's/.*"\(.*\)".*/\1/' | head -1)"

  # Update url, sha256 comment, sha256 value, and version
  sed \
    -e "s|url \"https://github.com/nself-org/cli/archive/refs/tags/v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.tar\.gz\"|url \"https://github.com/nself-org/cli/archive/refs/tags/v${NEW_VERSION}.tar.gz\"|" \
    -e "s|# sha256 computed from https://github.com/nself-org/cli/archive/refs/tags/v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.tar\.gz|# sha256 computed from https://github.com/nself-org/cli/archive/refs/tags/v${NEW_VERSION}.tar.gz|" \
    -e "s/^\(\s*sha256\s*\"\)[0-9a-f]*/\1${sha256}/" \
    -e "s/^\(\s*version\s*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW_VERSION}/" \
    "${f}" | atomic_write "${f}"
  report "${f}" "${old_ver}" "${NEW_VERSION}"
}

# ── Main ───────────────────────────────────────────────────────────────────────

printf '\n'
if [ "${DRY_RUN}" = "true" ]; then
  printf 'DRY RUN — no files will be modified\n'
fi
printf 'Target version: %s\n\n' "${NEW_VERSION}"

if [ "${HOMEBREW}" = "true" ]; then
  printf 'Phase 2 — updating Homebrew formula only\n\n'
  update_homebrew_formula
  printf '\nDone. Homebrew formula updated to %s.\n' "${NEW_VERSION}"
  exit 0
fi

printf 'Phase 1 — updating 11 lockstep files (Homebrew deferred to phase 2)\n\n'

update_version_file
update_version_go
update_sdk_go_doc
update_sdk_ts_pkg
update_sdk_py_pyproject
update_sdk_flutter_pubspec
update_admin_pkg
update_admin_cliver
update_admin_dockerfile

printf '\n'
if [ "${DRY_RUN}" = "true" ]; then
  printf 'Dry run complete — no files modified.\n'
else
  printf 'Version bump complete: all files now at %s\n' "${NEW_VERSION}"
  printf '\nNext steps:\n'
  printf '  1. Review the diff: git diff\n'
  printf '  2. Commit: git add -A && git commit -m "chore: bump version to v%s"\n' "${NEW_VERSION}"
  printf '  3. Push + tag: git tag v%s && git push origin main v%s\n' "${NEW_VERSION}" "${NEW_VERSION}"
  printf '  4. After GitHub release is published, run:\n'
  printf '     ./scripts/bump-version.sh %s --homebrew\n' "${NEW_VERSION}"
  printf '\nReminder: never hand-bump individual files — always use this script.\n'
fi
