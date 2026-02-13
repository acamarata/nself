#!/usr/bin/env bash
# platforms.sh - Platform detection & setup status

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
Usage: nself platforms <command> [options]

Platform detection and setup status for multi-platform builds.

Commands:
  status              Show platform build status
  setup <platform>    Setup a platform (interactive)
  validate <platform> Validate platform configuration

Platforms:
  web, android, ios, macos, windows, linux, tv

Examples:
  nself platforms status
  nself platforms setup ios
  nself platforms validate android

EOF
}

# Check web platform
check_web() {
  local status="ready"
  local message="Build command: npm run build"

  if [[ ! -f "package.json" ]]; then
    status="not-configured"
    message="No package.json found"
  elif ! grep -q '"build"' package.json 2>/dev/null; then
    status="missing"
    message="No build script in package.json"
  fi

  echo "$status:$message"
}

# Check Android platform
check_android() {
  local status="ready"
  local message="Keystore configured"

  if [[ ! -d "android" ]]; then
    status="not-configured"
    message="No android/ directory found"
    echo "$status:$message"
    return
  fi

  if [[ ! -f "android/app/build.gradle" ]]; then
    status="not-configured"
    message="No android/app/build.gradle found"
  elif [[ ! -f "android/app/release.keystore" ]] && [[ ! -f "android/app/my-upload-key.keystore" ]]; then
    status="missing"
    message="No keystore found for signing"
  fi

  echo "$status:$message"
}

# Check iOS platform
check_ios() {
  local status="ready"
  local message="Xcode configured"

  if [[ ! -d "ios" ]]; then
    status="not-configured"
    message="No ios/ directory found"
    echo "$status:$message"
    return
  fi

  # Check for Xcode
  if ! command -v xcodebuild >/dev/null 2>&1; then
    status="missing"
    message="Xcode not installed"
  # Check for provisioning profiles
  elif [[ ! -d "$HOME/Library/MobileDevice/Provisioning Profiles" ]] || \
       [[ -z "$(ls -A "$HOME/Library/MobileDevice/Provisioning Profiles" 2>/dev/null)" ]]; then
    status="missing"
    message="No provisioning profiles found"
  fi

  echo "$status:$message"
}

# Check desktop platforms
check_desktop() {
  local platform="$1"
  local status="ready"
  local message="Tauri configured"

  if [[ ! -d "platforms/desktop" ]] && [[ ! -f "src-tauri/tauri.conf.json" ]]; then
    status="not-configured"
    message="No Tauri configuration found"
    echo "$status:$message"
    return
  fi

  # Check for Rust
  if ! command -v cargo >/dev/null 2>&1; then
    status="missing"
    message="Rust/Cargo not installed"
  fi

  echo "$status:$message"
}

# Show status for all platforms
cmd_status() {
  printf "${COLOR_BLUE}Platform Build Status:${COLOR_RESET}\n\n"

  # Web
  local web_status=$(check_web)
  local web_state=$(echo "$web_status" | cut -d: -f1)
  local web_msg=$(echo "$web_status" | cut -d: -f2-)

  if [[ "$web_state" == "ready" ]]; then
    printf "${COLOR_GREEN}✅ Web${COLOR_RESET}              Ready to build\n"
    printf "   ${COLOR_DIM}$web_msg${COLOR_RESET}\n\n"
  elif [[ "$web_state" == "missing" ]]; then
    printf "${COLOR_YELLOW}⚠️  Web${COLOR_RESET}              $web_msg\n"
    printf "   ${COLOR_DIM}Run: npm run build${COLOR_RESET}\n\n"
  else
    printf "${COLOR_RED}❌ Web${COLOR_RESET}              Not configured\n"
    printf "   ${COLOR_DIM}$web_msg${COLOR_RESET}\n\n"
  fi

  # Android
  local android_status=$(check_android)
  local android_state=$(echo "$android_status" | cut -d: -f1)
  local android_msg=$(echo "$android_status" | cut -d: -f2-)

  if [[ "$android_state" == "ready" ]]; then
    printf "${COLOR_GREEN}✅ Android (APK)${COLOR_RESET}    Ready\n"
    printf "   ${COLOR_DIM}$android_msg${COLOR_RESET}\n\n"
  elif [[ "$android_state" == "missing" ]]; then
    printf "${COLOR_YELLOW}⚠️  Android (APK)${COLOR_RESET}    $android_msg\n"
    printf "   ${COLOR_DIM}Run: nself platforms setup android${COLOR_RESET}\n\n"
  else
    printf "${COLOR_RED}❌ Android (APK)${COLOR_RESET}    Not configured\n"
    printf "   ${COLOR_DIM}$android_msg${COLOR_RESET}\n\n"
  fi

  # iOS
  local ios_status=$(check_ios)
  local ios_state=$(echo "$ios_status" | cut -d: -f1)
  local ios_msg=$(echo "$ios_status" | cut -d: -f2-)

  if [[ "$ios_state" == "ready" ]]; then
    printf "${COLOR_GREEN}✅ iOS${COLOR_RESET}              Ready\n"
    printf "   ${COLOR_DIM}$ios_msg${COLOR_RESET}\n\n"
  elif [[ "$ios_state" == "missing" ]]; then
    printf "${COLOR_YELLOW}⚠️  iOS${COLOR_RESET}              $ios_msg\n"
    printf "   ${COLOR_DIM}Run: nself platforms setup ios${COLOR_RESET}\n\n"
  else
    printf "${COLOR_RED}❌ iOS${COLOR_RESET}              Not configured\n"
    printf "   ${COLOR_DIM}$ios_msg${COLOR_RESET}\n\n"
  fi

  # Desktop platforms
  for platform in macos windows linux; do
    local desktop_status=$(check_desktop "$platform")
    local desktop_state=$(echo "$desktop_status" | cut -d: -f1)
    local desktop_msg=$(echo "$desktop_status" | cut -d: -f2-)

    local display_name="${platform^}"

    if [[ "$desktop_state" == "ready" ]]; then
      printf "${COLOR_GREEN}✅ $display_name${COLOR_RESET}          Ready\n"
      printf "   ${COLOR_DIM}$desktop_msg${COLOR_RESET}\n\n"
    elif [[ "$desktop_state" == "missing" ]]; then
      printf "${COLOR_YELLOW}⚠️  $display_name${COLOR_RESET}          $desktop_msg\n\n"
    else
      printf "${COLOR_RED}❌ $display_name${COLOR_RESET}          Not configured\n"
      printf "   ${COLOR_DIM}$desktop_msg${COLOR_RESET}\n\n"
    fi
  done
}

# Setup a platform (placeholder for interactive setup)
cmd_setup() {
  local platform="$1"

  if [[ -z "$platform" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} Platform name required\n" >&2
    printf "  Usage: nself platforms setup <platform>\n" >&2
    return 1
  fi

  printf "${COLOR_YELLOW}⚠${COLOR_RESET} Interactive setup not yet implemented\n"
  printf "  Platform: $platform\n"
  printf "\n${COLOR_DIM}Manual setup steps:${COLOR_RESET}\n"

  case "$platform" in
    android)
      printf "  1. Install Android Studio\n"
      printf "  2. Configure SDK (API 34 or higher)\n"
      printf "  3. Generate keystore: keytool -genkey -v -keystore release.keystore ...\n"
      printf "  4. Add keystore path to android/gradle.properties\n"
      ;;
    ios)
      printf "  1. Install Xcode from App Store\n"
      printf "  2. Install Xcode Command Line Tools\n"
      printf "  3. Configure Apple Developer account\n"
      printf "  4. Download provisioning profiles\n"
      ;;
    macos|windows|linux)
      printf "  1. Install Rust: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh\n"
      printf "  2. Install Tauri CLI: cargo install tauri-cli\n"
      printf "  3. Initialize Tauri: cargo tauri init\n"
      ;;
    *)
      printf "  Unknown platform: $platform\n"
      ;;
  esac
}

# Validate platform configuration
cmd_validate() {
  local platform="$1"

  if [[ -z "$platform" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} Platform name required\n" >&2
    return 1
  fi

  printf "${COLOR_BLUE}Validating $platform platform...${COLOR_RESET}\n\n"

  case "$platform" in
    web)
      local status=$(check_web)
      local state=$(echo "$status" | cut -d: -f1)
      if [[ "$state" == "ready" ]]; then
        printf "${COLOR_GREEN}✓${COLOR_RESET} Web platform validated successfully\n"
      else
        printf "${COLOR_RED}✗${COLOR_RESET} Validation failed: $(echo "$status" | cut -d: -f2-)\n"
        return 1
      fi
      ;;
    android)
      local status=$(check_android)
      local state=$(echo "$status" | cut -d: -f1)
      if [[ "$state" == "ready" ]]; then
        printf "${COLOR_GREEN}✓${COLOR_RESET} Android platform validated successfully\n"
      else
        printf "${COLOR_RED}✗${COLOR_RESET} Validation failed: $(echo "$status" | cut -d: -f2-)\n"
        return 1
      fi
      ;;
    *)
      printf "${COLOR_YELLOW}⚠${COLOR_RESET} Validation for $platform not yet implemented\n"
      ;;
  esac
}

# Main command dispatcher
main() {
  local command="${1:-status}"
  shift || true

  case "$command" in
    status)
      cmd_status
      ;;
    setup)
      cmd_setup "$@"
      ;;
    validate)
      cmd_validate "$@"
      ;;
    -h|--help|help)
      show_help
      ;;
    *)
      printf "${COLOR_RED}✗${COLOR_RESET} Unknown command: $command\n" >&2
      return 1
      ;;
  esac
}

main "$@"
