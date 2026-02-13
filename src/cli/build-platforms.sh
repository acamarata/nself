#!/usr/bin/env bash
# build-platforms.sh - Multi-platform build automation

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
Usage: nself build platforms [options]

Build application for multiple platforms.

Options:
  --all                    Build all available platforms
  --platforms=<list>       Build specific platforms (comma-separated)
  --version=<version>      Set version number for builds
  --output=<dir>           Output directory (default: .releases/)
  --checksums              Generate SHA-256 checksums
  -h, --help               Show this help message

Examples:
  nself build platforms --all
  nself build platforms --platforms=web,android,ios
  nself build platforms --all --version=1.0.0 --checksums

Supported Platforms:
  web, android, ios, macos, windows, linux, tv

EOF
}

# Build web platform
build_web() {
  local version="$1"
  local output_dir="$2"

  printf "${COLOR_BLUE}⠿${COLOR_RESET} Building web platform...\n"

  if [[ ! -f "package.json" ]]; then
    printf "  ${COLOR_RED}✗${COLOR_RESET} No package.json found\n" >&2
    return 1
  fi

  # Run build
  if npm run build >/dev/null 2>&1; then
    # Package build output
    local output_file="$output_dir/app-${version}-web.zip"
    if [[ -d "build" ]]; then
      (cd build && zip -r "../$output_file" . >/dev/null 2>&1)
    elif [[ -d "dist" ]]; then
      (cd dist && zip -r "../$output_file" . >/dev/null 2>&1)
    fi

    local size=$(du -h "$output_file" 2>/dev/null | cut -f1)
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Web (${size}) → $output_file\n"
    return 0
  else
    printf "  ${COLOR_RED}✗${COLOR_RESET} Build failed\n" >&2
    return 1
  fi
}

# Build Android platform
build_android() {
  local version="$1"
  local output_dir="$2"

  printf "${COLOR_BLUE}⠿${COLOR_RESET} Building Android APK...\n"

  if [[ ! -d "android" ]]; then
    printf "  ${COLOR_YELLOW}⚠${COLOR_RESET} Android not configured\n"
    return 0
  fi

  # Run Android build
  if (cd android && ./gradlew assembleRelease >/dev/null 2>&1); then
    local apk_file=$(find android/app/build/outputs/apk/release -name "*.apk" | head -n 1)
    if [[ -n "$apk_file" ]]; then
      local output_file="$output_dir/app-${version}.apk"
      cp "$apk_file" "$output_file"

      local size=$(du -h "$output_file" 2>/dev/null | cut -f1)
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Android APK (${size}) → $output_file\n"
      return 0
    fi
  fi

  printf "  ${COLOR_RED}✗${COLOR_RESET} Build failed\n" >&2
  return 1
}

# Generate checksums
generate_checksums() {
  local output_dir="$1"

  printf "${COLOR_BLUE}⠿${COLOR_RESET} Generating checksums...\n"

  (cd "$output_dir" && shasum -a 256 *.{zip,apk,ipa,dmg,exe,AppImage} 2>/dev/null > SHA256SUMS) || true

  if [[ -f "$output_dir/SHA256SUMS" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Checksums → $output_dir/SHA256SUMS\n"
  fi
}

# Main build function
main() {
  local build_all=false
  local platforms=""
  local version="1.0.0"
  local output_dir=".releases"
  local gen_checksums=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --all)
        build_all=true
        shift
        ;;
      --platforms=*)
        platforms="${1#*=}"
        shift
        ;;
      --version=*)
        version="${1#*=}"
        shift
        ;;
      --output=*)
        output_dir="${1#*=}"
        shift
        ;;
      --checksums)
        gen_checksums=true
        shift
        ;;
      -h|--help)
        show_help
        return 0
        ;;
      *)
        shift
        ;;
    esac
  done

  # Create output directory
  mkdir -p "$output_dir"

  printf "${COLOR_BLUE}🔨 Building platforms (version: $version)${COLOR_RESET}\n\n"

  local success=0
  local failed=0

  # Determine which platforms to build
  if [[ "$build_all" == "true" ]]; then
    platforms="web,android"
  fi

  IFS=',' read -ra PLATFORM_LIST <<< "$platforms"

  for platform in "${PLATFORM_LIST[@]}"; do
    case "$platform" in
      web)
        if build_web "$version" "$output_dir"; then
          ((success++))
        else
          ((failed++))
        fi
        ;;
      android)
        if build_android "$version" "$output_dir"; then
          ((success++))
        else
          ((failed++))
        fi
        ;;
      *)
        printf "${COLOR_YELLOW}⚠${COLOR_RESET} Platform not yet implemented: $platform\n"
        ;;
    esac
  done

  printf "\n"

  # Generate checksums if requested
  if [[ "$gen_checksums" == "true" ]]; then
    generate_checksums "$output_dir"
    printf "\n"
  fi

  # Summary
  if [[ $failed -eq 0 ]]; then
    printf "${COLOR_GREEN}✓${COLOR_RESET} Build complete: $success platform(s) built\n"
    printf "  Output: ${COLOR_BLUE}$output_dir${COLOR_RESET}\n"
    return 0
  else
    printf "${COLOR_RED}✗${COLOR_RESET} Build failed: $success succeeded, $failed failed\n" >&2
    return 1
  fi
}

main "$@"
