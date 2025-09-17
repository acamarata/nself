#!/usr/bin/env bash
# platform.sh - Platform detection and compatibility layer for build

# Detect platform and set compatibility flags
detect_build_platform() {
  PLATFORM=""
  IS_MAC=false
  IS_LINUX=false
  IS_WSL=false
  IS_WINDOWS=false

  case "$(uname -s)" in
    Darwin*)
      PLATFORM="darwin"
      IS_MAC=true
      ;;
    Linux*)
      PLATFORM="linux"
      IS_LINUX=true
      # Check for WSL
      if grep -qi microsoft /proc/version 2>/dev/null; then
        IS_WSL=true
      fi
      ;;
    CYGWIN*|MINGW*|MSYS*)
      PLATFORM="windows"
      IS_WINDOWS=true
      ;;
    *)
      PLATFORM="unknown"
      ;;
  esac

  export PLATFORM IS_MAC IS_LINUX IS_WSL IS_WINDOWS
}

# Safe arithmetic increment that works on all platforms
safe_increment() {
  local var_name="$1"
  local current_val="${!var_name:-0}"

  # Use expr for maximum compatibility
  if command -v expr >/dev/null 2>&1; then
    eval "$var_name=\$(expr \$current_val + 1)"
  else
    # Fallback to let
    let "$var_name=current_val+1" 2>/dev/null || eval "$var_name=\$((current_val + 1))"
  fi
}

# Safe arithmetic operations
safe_math() {
  local operation="$1"

  # Use expr for compatibility
  if command -v expr >/dev/null 2>&1; then
    expr $operation 2>/dev/null || echo "0"
  else
    echo "$((operation))" 2>/dev/null || echo "0"
  fi
}

# Get available CPU cores (cross-platform)
get_cpu_cores() {
  local cores=2  # Default fallback

  if [[ "$IS_MAC" == true ]]; then
    cores=$(sysctl -n hw.ncpu 2>/dev/null || echo 2)
  elif [[ "$IS_LINUX" == true ]]; then
    cores=$(nproc 2>/dev/null || grep -c processor /proc/cpuinfo 2>/dev/null || echo 2)
  elif [[ "$IS_WINDOWS" == true ]]; then
    cores=${NUMBER_OF_PROCESSORS:-2}
  fi

  echo "$cores"
}

# Get available memory in MB (cross-platform)
get_memory_mb() {
  local memory=1024  # Default 1GB

  if [[ "$IS_MAC" == true ]]; then
    memory=$(( $(sysctl -n hw.memsize 2>/dev/null || echo 1073741824) / 1024 / 1024 ))
  elif [[ "$IS_LINUX" == true ]]; then
    memory=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print int($2/1024)}' || echo 1024)
  fi

  echo "$memory"
}

# Check if running in Docker container
is_in_docker() {
  [[ -f /.dockerenv ]] || grep -q docker /proc/1/cgroup 2>/dev/null
}

# Check if command exists (portable)
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Get sed in-place flag (BSD vs GNU)
get_sed_inplace_flag() {
  if sed --version 2>&1 | grep -q GNU; then
    echo "-i"
  else
    echo "-i ''"
  fi
}

# Safe variable default with proper escaping
set_default() {
  local var_name="$1"
  local default_value="$2"

  # Check if variable is unset or empty
  if [[ -z "${!var_name:-}" ]]; then
    eval "$var_name=\$default_value"
  fi
}

# Export functions
export -f detect_build_platform
export -f safe_increment
export -f safe_math
export -f get_cpu_cores
export -f get_memory_mb
export -f is_in_docker
export -f command_exists
export -f get_sed_inplace_flag
export -f set_default