#!/bin/bash
# nSelf CLI Installer
# Usage: curl -fsSL https://install.nself.org | bash
#
# Install modes:
#   curl ... | bash                      # Default: install to /usr/local/bin
#   curl ... | bash -s -- --user         # Install to ~/.nself/bin (no sudo)
#   curl ... | bash -s -- --dir /path    # Custom directory
#
# Environment variables:
#   NSELF_VERSION      — Specific version (default: latest release)
#   NSELF_INSTALL_DIR  — Override install directory
#   FORCE_REINSTALL    — Set to "true" to overwrite existing install

set -euo pipefail

REPO="nself-org/cli"
BINARY_NAME="nself"
HOMEBREW_TAP="nself-org/nself"

# ── Colors ─────────────────────────────────────────────────────────
if [ -t 1 ]; then
  GREEN='\033[0;32m' RED='\033[0;31m' BLUE='\033[0;34m'
  YELLOW='\033[0;33m' DIM='\033[2m' BOLD='\033[1m' RESET='\033[0m'
else
  GREEN='' RED='' BLUE='' YELLOW='' DIM='' BOLD='' RESET=''
fi

info()    { printf "${BLUE}ℹ${RESET} %s\n" "$1"; }
success() { printf "${GREEN}✓${RESET} %s\n" "$1"; }
warn()    { printf "${YELLOW}⚠${RESET} %s\n" "$1"; }
error()   { printf "${RED}✗${RESET} %s\n" "$1" >&2; exit 1; }

# ── Platform Detection ─────────────────────────────────────────────
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       error "Unsupported OS: $(uname -s). nSelf supports Linux and macOS." ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)             error "Unsupported architecture: $(uname -m). nSelf supports amd64 and arm64." ;;
  esac
}

# ── Version Resolution ─────────────────────────────────────────────
get_latest_version() {
  if [ -n "${NSELF_VERSION:-}" ]; then
    echo "$NSELF_VERSION"
    return
  fi

  local latest
  latest=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' \
    | sed 's/^v//')

  if [ -z "$latest" ]; then
    error "Could not determine latest version. Set NSELF_VERSION=x.y.z or check https://github.com/${REPO}/releases"
  fi

  echo "$latest"
}

# ── Existing Installation Detection ────────────────────────────────
check_existing() {
  local existing
  existing=$(command -v nself 2>/dev/null || true)

  if [ -n "$existing" ]; then
    local current_version
    current_version=$("$existing" version --short 2>/dev/null || echo "unknown")
    warn "nSelf v${current_version} already installed at ${existing}"

    if [ "${FORCE_REINSTALL:-false}" = "true" ]; then
      info "FORCE_REINSTALL=true — overwriting"
    else
      # Check if it's a Homebrew install
      if echo "$existing" | grep -q "Cellar\|homebrew"; then
        info "Installed via Homebrew. Upgrade with: brew upgrade nself"
        info "Or set FORCE_REINSTALL=true to install alongside."
        exit 0
      fi

      info "Upgrading ${current_version} → ${1}"
    fi
  fi
}

# ── Docker Requirement Check ───────────────────────────────────────
check_docker() {
  if ! command -v docker > /dev/null 2>&1; then
    warn "Docker not found. nSelf requires Docker to run."
    info "Install Docker: https://docs.docker.com/get-docker/"
  elif ! docker info > /dev/null 2>&1; then
    warn "Docker is installed but not running."
    info "Start Docker Desktop or run: sudo systemctl start docker"
  else
    success "Docker is running"
  fi
}

# ── PATH Management ────────────────────────────────────────────────
ensure_in_path() {
  local dir="$1"

  # Already in PATH
  if echo "$PATH" | tr ':' '\n' | grep -qx "$dir"; then
    return
  fi

  info "Adding ${dir} to PATH"

  # Detect shell config file
  local shell_rc=""
  case "${SHELL:-/bin/bash}" in
    */zsh)  shell_rc="$HOME/.zshrc" ;;
    */bash)
      if [ -f "$HOME/.bash_profile" ]; then
        shell_rc="$HOME/.bash_profile"
      else
        shell_rc="$HOME/.bashrc"
      fi
      ;;
    */fish) shell_rc="$HOME/.config/fish/config.fish" ;;
  esac

  if [ -n "$shell_rc" ]; then
    if ! grep -q "$dir" "$shell_rc" 2>/dev/null; then
      echo "export PATH=\"${dir}:\$PATH\"" >> "$shell_rc"
      info "Added to ${shell_rc} — restart your shell or run: source ${shell_rc}"
    fi
  fi
}

# ── Main ───────────────────────────────────────────────────────────
main() {
  # Parse arguments
  local install_mode="system"
  local custom_dir=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --user)    install_mode="user"; shift ;;
      --system)  install_mode="system"; shift ;;
      --dir)     install_mode="custom"; custom_dir="$2"; shift 2 ;;
      --force)   FORCE_REINSTALL="true"; shift ;;
      --help|-h)
        printf "Usage: curl -fsSL https://install.nself.org | bash [-- OPTIONS]\n\n"
        printf "Options:\n"
        printf "  --user       Install to ~/.nself/bin (no sudo required)\n"
        printf "  --system     Install to /usr/local/bin (default, may need sudo)\n"
        printf "  --dir PATH   Install to custom directory\n"
        printf "  --force      Overwrite existing installation\n"
        printf "\nEnvironment:\n"
        printf "  NSELF_VERSION=x.y.z    Install specific version\n"
        printf "  FORCE_REINSTALL=true   Overwrite existing\n"
        exit 0
        ;;
      *) shift ;;
    esac
  done

  # Determine install directory
  local install_dir
  case "$install_mode" in
    user)   install_dir="$HOME/.nself/bin" ;;
    system) install_dir="${NSELF_INSTALL_DIR:-/usr/local/bin}" ;;
    custom) install_dir="$custom_dir" ;;
  esac

  # ASCII banner
  printf "\n${BOLD}"
  printf "  ___  _____ ___ _  __ \n"
  printf " | _ \/ ____|  _| |/ _|\n"
  printf " |  _/\__ \| |_| |  _| \n"
  printf " |_|  |____/|___|_|_|  \n"
  printf "${RESET}\n"
  printf "  ${BOLD}nSelf CLI Installer${RESET}\n\n"

  # Detect platform
  local os arch version
  os=$(detect_os)
  arch=$(detect_arch)
  info "Platform: ${os}/${arch}"

  # Get version
  version=$(get_latest_version)
  info "Version: ${version}"

  # Check existing
  check_existing "$version"

  # Check Docker
  check_docker

  # Ensure install directory exists
  mkdir -p "$install_dir" 2>/dev/null || true

  # Build download URL
  local tarball="nself-${version}-${os}-${arch}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/v${version}/${tarball}"
  info "Downloading..."

  # Download tarball to temp file
  local tmptarball tmpfile
  tmptarball=$(mktemp /tmp/nself-install-XXXXXX.tar.gz)
  tmpfile=$(mktemp /tmp/nself-install-XXXXXX)
  trap "rm -f '$tmptarball' '$tmpfile'" EXIT

  if ! curl -fsSL --progress-bar -o "$tmptarball" "$url"; then
    error "Download failed. Check https://github.com/${REPO}/releases for available binaries."
  fi

  # Extract binary from tarball — structure is nself-{version}-{os}-{arch}/nself
  local dir_prefix="nself-${version}-${os}-${arch}"
  if ! tar -xzf "$tmptarball" -O "${dir_prefix}/nself" > "$tmpfile" 2>/dev/null; then
    error "Failed to extract binary from tarball."
  fi

  chmod +x "$tmpfile"

  # Verify binary
  if ! "$tmpfile" version --short > /dev/null 2>&1; then
    error "Downloaded binary is corrupt. Try again or download manually."
  fi

  # Install
  local install_path="${install_dir}/${BINARY_NAME}"

  if [ -w "$install_dir" ]; then
    mv "$tmpfile" "$install_path"
  else
    info "Installing to ${install_path} (requires sudo)"
    sudo mv "$tmpfile" "$install_path"
    sudo chmod +x "$install_path"
  fi

  success "Installed nself v${version} to ${install_path}"

  # Ensure in PATH (for --user mode)
  if [ "$install_mode" = "user" ]; then
    ensure_in_path "$install_dir"
  fi

  # Post-install smoke test
  printf "\n${BOLD}Running post-install checks...${RESET}\n"
  local smoke_ok=true

  # Smoke 1: version
  if "$install_path" version --short > /dev/null 2>&1; then
    local installed_ver
    installed_ver=$("$install_path" version --short 2>/dev/null || echo "unknown")
    success "nself --version: ${installed_ver}"
  else
    warn "nself --version check failed — binary may not be executable"
    smoke_ok=false
  fi

  # Smoke 2: quick doctor (non-fatal; informational)
  if "$install_path" doctor --quick > /dev/null 2>&1; then
    success "nself doctor --quick: OK"
  else
    warn "nself doctor --quick: one or more checks failed (run manually for details)"
    smoke_ok=false
  fi

  if [ "$smoke_ok" = "false" ]; then
    printf "\n${YELLOW}⚠${RESET}  Some checks failed. Run ${BOLD}nself doctor${RESET} for details.\n"
    printf "   If issues persist, enable diagnostics:\n"
    printf "   ${DIM}NSELF_DIAG=1 nself doctor --deep 2>&1 | tee /tmp/nself-diag.txt${RESET}\n"
    printf "   Then share /tmp/nself-diag.txt when reporting an issue:\n"
    printf "   ${DIM}https://github.com/${REPO}/issues/new${RESET}\n\n"
  fi

  # Verify installed version
  printf "\n"
  "$install_path" version
  printf "\n"

  # Quick start
  printf "${BOLD}Quick start:${RESET}\n"
  printf "  ${DIM}mkdir my-project && cd my-project${RESET}\n"
  printf "  nself init\n"
  printf "  nself start\n"
  printf "\n"

  # Homebrew hint for macOS — with Postgres conflict warning
  if [ "$os" = "darwin" ]; then
    printf "${DIM}Tip: You can also install via Homebrew:${RESET}\n"
    printf "${DIM}  brew install ${HOMEBREW_TAP}/nself${RESET}\n\n"

    # Homebrew Postgres conflict hint
    if command -v brew > /dev/null 2>&1; then
      if brew list postgresql@14 > /dev/null 2>&1 || brew list postgresql@15 > /dev/null 2>&1 || brew list postgresql > /dev/null 2>&1; then
        printf "${YELLOW}⚠${RESET}  Homebrew Postgres detected.\n"
        printf "   nSelf runs its own Postgres in Docker — port 5432 may conflict.\n"
        printf "   Stop Homebrew Postgres before running nself start:\n"
        printf "     ${DIM}brew services stop postgresql@15${RESET}\n"
        printf "   Or configure a different port in your .env (POSTGRES_PORT=5433).\n"
        printf "   Run ${BOLD}nself doctor${RESET} after init to check for port conflicts.\n\n"
      fi
    fi
  fi
}

main "$@"
