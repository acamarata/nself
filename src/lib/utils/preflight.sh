#!/usr/bin/env bash
# preflight.sh - Pre-flight checks for nself commands

# Check if running in nself source directory
check_not_in_source() {
  if [[ -f "bin/nself" ]] && [[ -d "src/cli" ]] && [[ -d "src/lib" ]] && [[ -f "install.sh" ]]; then
    log_error "Cannot run nself commands in the nself source repository!"
    echo ""
    log_info "Please run from a separate project directory:"
    echo "  mkdir ~/myproject && cd ~/myproject"
    echo "  nself init"
    return 1
  fi
  return 0
}

# Check for environment configuration file
check_env_file() {
  if [[ ! -f ".env" ]] && [[ ! -f ".env.dev" ]]; then
    log_error "No environment file found (.env or .env.dev)"
    echo ""
    log_info "Please run: nself init"
    echo ""
    echo "This will create the required configuration files."
    return 1
  fi
  return 0
}

# Check for docker-compose.yml
check_docker_compose() {
  if [[ ! -f "docker-compose.yml" ]]; then
    log_error "No docker-compose.yml found - project not built yet"
    echo ""
    log_info "Please run: nself build"
    echo ""
    echo "This will generate all required project files."
    return 1
  fi
  return 0
}

# Check Docker is installed and running
check_docker() {
  # Check if docker command exists
  if ! command -v docker &>/dev/null; then
    log_error "Docker is not installed"
    echo ""
    log_info "Please install Docker Desktop from:"
    echo "  https://www.docker.com/products/docker-desktop"
    return 1
  fi

  # Check if Docker daemon is running
  if ! docker info &>/dev/null; then
    log_error "Docker daemon is not running"
    echo ""
    log_info "Please start Docker Desktop"
    return 1
  fi

  return 0
}

# Check for required commands (cross-platform)
check_required_commands() {
  local missing_commands=()

  # List of required commands
  local required_commands=(
    "docker"
    "sed"
    "awk"
    "grep"
  )

  for cmd in "${required_commands[@]}"; do
    if ! command -v "$cmd" &>/dev/null; then
      missing_commands+=("$cmd")
    fi
  done

  if [[ ${#missing_commands[@]} -gt 0 ]]; then
    log_error "Missing required commands: ${missing_commands[*]}"
    return 1
  fi

  return 0
}

# Check platform-specific requirements
check_platform() {
  local platform="$(uname -s)"

  case "$platform" in
  Darwin*)
    # macOS specific checks
    # Bash version check (macOS ships with old bash)
    if [[ "${BASH_VERSION%%.*}" -lt 3 ]]; then
      log_warning "Bash version is old (${BASH_VERSION})"
      log_info "Consider upgrading bash: brew install bash"
    fi
    ;;
  Linux*)
    # Linux specific checks
    # Check if running with proper permissions
    if [[ "$EUID" -eq 0 ]] && [[ -z "${ALLOW_ROOT:-}" ]]; then
      log_warning "Running as root is not recommended"
      log_info "Set ALLOW_ROOT=true to bypass this warning"
    fi
    ;;
  MINGW* | MSYS* | CYGWIN*)
    # Windows specific checks
    log_warning "Windows detected - some features may not work correctly"
    log_info "Consider using WSL2 for better compatibility"
    ;;
  *)
    log_warning "Unknown platform: $platform"
    ;;
  esac

  return 0
}

# Run all pre-flight checks for init command
preflight_init() {
  check_not_in_source || return 1
  check_required_commands || return 1
  check_platform || return 1
  return 0
}

# Run all pre-flight checks for build command
preflight_build() {
  check_not_in_source || return 1
  check_env_file || return 1
  check_required_commands || return 1
  check_platform || return 1
  return 0
}

# Run all pre-flight checks for up command
preflight_up() {
  check_not_in_source || return 1
  check_env_file || return 1
  check_docker_compose || return 1
  check_docker || return 1
  check_required_commands || return 1
  check_platform || return 1
  return 0
}

# Export all functions
export -f check_not_in_source
export -f check_env_file
export -f check_docker_compose
export -f check_docker
export -f check_required_commands
export -f check_platform
export -f preflight_init
export -f preflight_build
export -f preflight_up
