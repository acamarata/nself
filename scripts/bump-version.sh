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
# Files updated (10 + 1 optional homebrew) — Flutter SDK removed 2026-06-30 (#159):
#   1.  cli/.github/VERSION
#   2.  cli/internal/version/version.go       Version string = "x.y.z" (type-annotated)
#   3.  cli/sdk/go/doc.go                     const Version = "x.y.z"
#   4.  cli/sdk/ts/package.json               "version": "x.y.z"   (via jq)
#   5.  cli/sdk/py/pyproject.toml             version = "x.y.z" under [project]
#   6.  admin/package.json                    "version": "x.y.z"   (via jq)
#   7.  admin/src/lib/cli-version.ts          CLI_VERSION = 'x.y.z'
#   8.  admin/Dockerfile                      ARG NSELF_VERSION=x.y.z
#   9.  admin/Dockerfile                      ENV ADMIN_VERSION=x.y.z
#   10. admin/Dockerfile                      LABEL org.opencontainers.image.version="x.y.z"
# +11. homebrew-nself/Formula/nself.rb       url/sha256/version (--homebrew only)
#
# Exit codes:
#   0 — success (all changes applied, or all already at target in idempotent run)
#   1 — usage error, bad semver, missing file, or homebrew release not found
#
# Hard rules:
#   - Atomic writes: all file writes use tmp+rename, cleaned on EXIT
#   - Portable patterns: POSIX BRE only (no GNU \s); jq for JSON files
#   - Idempotent: skip files already at the target version
#   - shellcheck-clean (no echo -e, no ${var,,}, no declare -A)
#
# L-P98-06 fix (2026-05-13):
#   - Replaced GNU \s with POSIX [[:space:]] in all regex
#   - Routed JSON edits through jq to handle nested/contextual "version" fields
#   - Added Go type-annotated form: Version<spaces>string<spaces>=<spaces>"..."
#   - Added scripts/bump-version_test.sh with fixtures for all 11 line shapes

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

NEW_VERSION="${RAW_VERSION#v}"

if ! printf '%s' "${NEW_VERSION}" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  printf 'ERROR: Invalid semver: "%s"\n' "${RAW_VERSION}" >&2
  printf '  Expected X.Y.Z (e.g. 1.0.14) or vX.Y.Z (e.g. v1.0.14)\n' >&2
  exit 1
fi

# ── Tool availability checks ──────────────────────────────────────────────────

if ! command -v jq >/dev/null 2>&1; then
  printf 'ERROR: jq is required for JSON file bumps (sdk/ts/package.json, admin/package.json)\n' >&2
  printf '  Install: brew install jq  (macOS)  |  apt-get install jq  (Linux)\n' >&2
  exit 1
fi

# ── File paths ────────────────────────────────────────────────────────────────

VERSION_FILE="${REPO_ROOT}/.github/VERSION"
VERSION_GO="${REPO_ROOT}/internal/version/version.go"
SDK_GO="${REPO_ROOT}/sdk/go/doc.go"
SDK_TS="${REPO_ROOT}/sdk/ts/package.json"
SDK_PY="${REPO_ROOT}/sdk/py/pyproject.toml"
# SDK_FLUTTER removed 2026-06-30 (#159, ASI Policy 2)
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

# report <file> <old> <new>
report() {
  printf '  %-60s  %s -> %s\n' "$1" "$2" "$3"
}

# extract_semver <file>
#   Extract the first X.Y.Z occurrence on any line matching a given grep pattern.
#   Args: file, grep-pattern
extract_semver() {
  grep -E "$2" "$1" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
}

# update_json_version <file>
#   Top-level .version field replacement via jq. Atomic via tmp+rename.
update_json_version() {
  local f="$1"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(jq -r '.version // ""' "${f}")"
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  if [ -z "${old}" ]; then
    printf 'ERROR: %s has no top-level "version" field\n' "${f}" >&2
    exit 1
  fi
  jq --arg v "${NEW_VERSION}" '.version = $v' "${f}" | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
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

# version.go shape: `\tVersion   string = "1.0.13"` (tab/spaces, type annotation)
update_version_go() {
  local f="${VERSION_GO}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(extract_semver "${f}" '^[[:space:]]+Version[[:space:]]+string[[:space:]]*=')"
  if [ -z "${old}" ]; then
    printf 'ERROR: %s has no matching "Version<sp>string<sp>=" line\n' "${f}" >&2
    exit 1
  fi
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  # POSIX BRE — explicit [[:space:]]+, no \s
  sed "s/^\([[:space:]]*Version[[:space:]][[:space:]]*string[[:space:]]*=[[:space:]]*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW_VERSION}/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

# sdk/go/doc.go shape: `const Version = "2.0.0"`
update_sdk_go_doc() {
  local f="${SDK_GO}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(extract_semver "${f}" '^const Version = ')"
  if [ -z "${old}" ]; then
    # doc.go may not declare a const Version; treat as soft-skip
    printf '  SKIP  %s  (no const Version declaration)\n' "${f}"
    return
  fi
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  sed "s/^const Version = \"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/const Version = \"${NEW_VERSION}\"/" "${f}" \
    | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_sdk_ts_pkg() {
  update_json_version "${SDK_TS}"
}

# pyproject.toml shape: `version = "2.0.0"` under [project]
# Also handles Cargo.toml shape: `version = "X.Y.Z"` under [package]
# Uses awk to scope replacement to the FIRST matching version line after the
# expected [project] or [package] section header — safer than naive sed on
# files with [dependencies.foo] sub-tables containing their own version fields.
update_toml_version() {
  local f="$1"
  local section="${2:-project}"  # default [project] for pyproject; pass "package" for Cargo
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  # Extract: first version = "X.Y.Z" line inside [section]…(next section or EOF)
  old="$(awk -v sect="[${section}]" '
    $0 == sect { in_section = 1; next }
    in_section && /^\[/ { in_section = 0 }
    in_section && /^[[:space:]]*version[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"/ {
      match($0, /[0-9]+\.[0-9]+\.[0-9]+/)
      print substr($0, RSTART, RLENGTH); exit
    }
  ' "${f}")"
  if [ -z "${old}" ]; then
    printf 'ERROR: %s has no "version = X.Y.Z" line under [%s]\n' "${f}" "${section}" >&2
    exit 1
  fi
  if [ "${old}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi
  awk -v sect="[${section}]" -v newv="${NEW_VERSION}" '
    $0 == sect { in_section = 1; print; next }
    in_section && /^\[/ { in_section = 0 }
    in_section && !replaced && /^[[:space:]]*version[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"/ {
      sub(/[0-9]+\.[0-9]+\.[0-9]+/, newv)
      replaced = 1
    }
    { print }
  ' "${f}" | atomic_write "${f}"
  report "${f}" "${old}" "${NEW_VERSION}"
}

update_sdk_py_pyproject() {
  update_toml_version "${SDK_PY}" project
}

# Flutter SDK removed 2026-06-30 (#159, ASI Policy 2) — pubspec updater retired.

update_admin_pkg() {
  update_json_version "${ADMIN_PKG}"
}

update_admin_cliver() {
  local f="${ADMIN_CLIVER}"
  if [ ! -f "${f}" ]; then
    printf 'ERROR: Missing file: %s\n' "${f}" >&2; exit 1
  fi
  local old
  old="$(extract_semver "${f}" "^export const CLI_VERSION = '")"
  if [ -z "${old}" ]; then
    printf 'ERROR: %s has no matching "export const CLI_VERSION = " line\n' "${f}" >&2
    exit 1
  fi
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
  old_nself="$(extract_semver "${f}" '^ARG NSELF_VERSION=')"
  old_admin="$(extract_semver "${f}" '^ENV ADMIN_VERSION=')"
  old_label="$(extract_semver "${f}" '^LABEL org\.opencontainers\.image\.version=')"

  if [ "${old_nself}" = "${NEW_VERSION}" ] && [ "${old_admin}" = "${NEW_VERSION}" ] && [ "${old_label}" = "${NEW_VERSION}" ]; then
    printf '  SKIP  %s  (all three lines already %s)\n' "${f}" "${NEW_VERSION}"
    return
  fi

  sed \
    -e "s/^ARG NSELF_VERSION=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/ARG NSELF_VERSION=${NEW_VERSION}/" \
    -e "s/^ENV ADMIN_VERSION=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/ENV ADMIN_VERSION=${NEW_VERSION}/" \
    -e "s/^LABEL org\.opencontainers\.image\.version=\"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/LABEL org.opencontainers.image.version=\"${NEW_VERSION}\"/" \
    "${f}" | atomic_write "${f}"
  report "${f} (ARG NSELF_VERSION)" "${old_nself:-?}" "${NEW_VERSION}"
  report "${f} (ENV ADMIN_VERSION)" "${old_admin:-?}" "${NEW_VERSION}"
  report "${f} (LABEL image.version)" "${old_label:-?}" "${NEW_VERSION}"
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
  old_ver="$(grep -E '^[[:space:]]*version[[:space:]]+"' "${f}" | sed 's/.*"\(.*\)".*/\1/' | head -1)"

  sed \
    -e "s|url \"https://github.com/nself-org/cli/archive/refs/tags/v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.tar\.gz\"|url \"https://github.com/nself-org/cli/archive/refs/tags/v${NEW_VERSION}.tar.gz\"|" \
    -e "s|# sha256 computed from https://github.com/nself-org/cli/archive/refs/tags/v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.tar\.gz|# sha256 computed from https://github.com/nself-org/cli/archive/refs/tags/v${NEW_VERSION}.tar.gz|" \
    -e "s/^\([[:space:]]*sha256[[:space:]]*\"\)[0-9a-f]*/\1${sha256}/" \
    -e "s/^\([[:space:]]*version[[:space:]]*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW_VERSION}/" \
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

printf 'Phase 1 — updating 10 lockstep files (Homebrew deferred to phase 2)\n\n'

update_version_file
update_version_go
update_sdk_go_doc
update_sdk_ts_pkg
update_sdk_py_pyproject
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
