#!/usr/bin/env bash
# release.sh - Release packaging & distribution

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source utilities
source "$LIB_DIR/utils/display.sh"

# Colors
COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_BLUE='\033[0;34m'
COLOR_DIM='\033[2m'

# Show help
show_help() {
  cat << 'EOF'
Usage: nself release <command> [options]

Release packaging and distribution management.

Commands:
  create <version>        Create a new release
  notes                   Generate release notes from git commits
  checksums               Generate checksums for release artifacts
  publish                 Publish release to GitHub

Options:
  --all                   Build all platforms
  --platforms=<list>      Build specific platforms
  --output=<dir>          Output directory (default: .releases/)
  --since=<tag>           Generate notes since tag/commit
  --github                Publish to GitHub releases
  -h, --help              Show this help message

Examples:
  nself release create v1.0.0 --all
  nself release notes --since=v0.9.0
  nself release checksums
  nself release publish --github

EOF
}

# Create release
cmd_create() {
  local version="$1"
  shift || true

  if [[ -z "$version" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} Version required\n" >&2
    printf "  Usage: nself release create <version>\n" >&2
    return 1
  fi

  # Remove v prefix if present
  version="${version#v}"

  printf "${COLOR_BLUE}🚀 Creating release v$version${COLOR_RESET}\n\n"

  # Build platforms
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Building platforms...\n"
  if bash "$SCRIPT_DIR/build-platforms.sh" --version="$version" --checksums "$@"; then
    printf "\n${COLOR_GREEN}✓${COLOR_RESET} Build complete\n\n"
  else
    printf "\n${COLOR_RED}✗${COLOR_RESET} Build failed\n" >&2
    return 1
  fi

  # Generate changelog
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Generating CHANGELOG.md...\n"
  cmd_notes --since="v$(git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")" > .releases/CHANGELOG.md 2>/dev/null || true
  printf "${COLOR_GREEN}✓${COLOR_RESET} Created .releases/CHANGELOG.md\n\n"

  # Summary
  local artifact_count=$(find .releases -maxdepth 1 -type f \( -name "*.zip" -o -name "*.apk" -o -name "*.ipa" \) 2>/dev/null | wc -l | tr -d ' ')
  local total_size=$(du -sh .releases 2>/dev/null | cut -f1)

  printf "${COLOR_GREEN}✓${COLOR_RESET} Release v$version created\n"
  printf "  Artifacts: $artifact_count file(s)\n"
  printf "  Total size: $total_size\n"
  printf "  Output: ${COLOR_BLUE}.releases/${COLOR_RESET}\n"
}

# Generate release notes
cmd_notes() {
  local since="${1:-}"

  if [[ "$since" == "--since="* ]]; then
    since="${since#--since=}"
  fi

  if [[ -z "$since" ]]; then
    since="$(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~10")"
  fi

  printf "# Release Notes\n\n"
  printf "## Changes since $since\n\n"

  # Get commits
  local commits=$(git log "$since"..HEAD --pretty=format:"%s" 2>/dev/null || echo "")

  if [[ -z "$commits" ]]; then
    printf "No changes\n"
    return 0
  fi

  # Categorize commits
  printf "### Features\n"
  echo "$commits" | grep "^feat:" | sed 's/^feat: /- /' || echo "- None"
  printf "\n"

  printf "### Bug Fixes\n"
  echo "$commits" | grep "^fix:" | sed 's/^fix: /- /' || echo "- None"
  printf "\n"

  printf "### Other Changes\n"
  echo "$commits" | grep -v "^feat:" | grep -v "^fix:" | sed 's/^/- /' || echo "- None"
  printf "\n"
}

# Generate checksums
cmd_checksums() {
  if [[ ! -d ".releases" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} No .releases directory found\n" >&2
    return 1
  fi

  printf "${COLOR_BLUE}⠿${COLOR_RESET} Generating checksums...\n"

  (cd .releases && shasum -a 256 *.{zip,apk,ipa,dmg,exe,AppImage} 2>/dev/null > SHA256SUMS) || {
    printf "${COLOR_YELLOW}⚠${COLOR_RESET} No artifacts found\n"
    return 0
  }

  printf "${COLOR_GREEN}✓${COLOR_RESET} Checksums generated: .releases/SHA256SUMS\n"
}

# Publish to GitHub
cmd_publish() {
  printf "${COLOR_YELLOW}⚠${COLOR_RESET} GitHub publishing not yet implemented\n"
  printf "  Use GitHub CLI manually:\n"
  printf "    ${COLOR_BLUE}gh release create v1.0.0 .releases/*${COLOR_RESET}\n"
}

# Main command dispatcher
main() {
  local command="${1:-}"
  shift || true

  case "$command" in
    create)
      cmd_create "$@"
      ;;
    notes)
      cmd_notes "$@"
      ;;
    checksums)
      cmd_checksums
      ;;
    publish)
      cmd_publish
      ;;
    -h|--help|help|"")
      show_help
      ;;
    *)
      printf "${COLOR_RED}✗${COLOR_RESET} Unknown command: $command\n" >&2
      return 1
      ;;
  esac
}

main "$@"
