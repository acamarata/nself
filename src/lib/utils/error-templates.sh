#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/display.sh"
source "$(dirname "${BASH_SOURCE[0]}")/output-formatter.sh"

# Professional error message templates for common issues

show_docker_not_running_error() {
  local platform="${1:-$PLATFORM}"

  format_error "Docker is not running" ""

  echo -e "\n${BOLD}Quick Fix:${RESET}"

  case "$platform" in
  macos)
    echo -e "  ${BLUE}→${RESET} Open Docker Desktop: ${BOLD}open -a Docker${RESET}"
    echo -e "  ${BLUE}→${RESET} Or from Spotlight: Press ${BOLD}⌘ + Space${RESET} and type 'Docker'"
    echo -e "\n${DIM}If Docker Desktop is not installed:${RESET}"
    echo -e "  ${BLUE}→${RESET} Download from: ${UNDERLINE}https://docker.com/products/docker-desktop${RESET}"
    echo -e "  ${BLUE}→${RESET} Or install via Homebrew: ${BOLD}brew install --cask docker${RESET}"
    ;;
  linux)
    echo -e "  ${BLUE}→${RESET} Start Docker service: ${BOLD}sudo systemctl start docker${RESET}"
    echo -e "  ${BLUE}→${RESET} Enable on boot: ${BOLD}sudo systemctl enable docker${RESET}"
    echo -e "\n${DIM}If Docker is not installed:${RESET}"
    echo -e "  ${BLUE}→${RESET} Install: ${BOLD}curl -fsSL https://get.docker.com | sh${RESET}"
    echo -e "  ${BLUE}→${RESET} Add user to group: ${BOLD}sudo usermod -aG docker \$USER${RESET}"
    ;;
  windows)
    echo -e "  ${BLUE}→${RESET} Open Docker Desktop from Start Menu"
    echo -e "  ${BLUE}→${RESET} Or run: ${BOLD}\"C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe\"${RESET}"
    echo -e "\n${DIM}If Docker Desktop is not installed:${RESET}"
    echo -e "  ${BLUE}→${RESET} Download from: ${UNDERLINE}https://docker.com/products/docker-desktop${RESET}"
    echo -e "  ${BLUE}→${RESET} Or via winget: ${BOLD}winget install Docker.DockerDesktop${RESET}"
    ;;
  esac

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} After starting Docker, wait 10-15 seconds for it to fully initialize."
}

show_port_conflict_error() {
  local port="$1"
  local service="$2"

  format_error "Port $port is already in use" ""

  echo -e "\n${BOLD}Conflicting Service:${RESET}"

  # Try to identify what's using the port
  local process_info=""
  if [[ "$PLATFORM" == "macos" ]]; then
    process_info=$(lsof -i :$port -P 2>/dev/null | grep LISTEN | head -1)
  elif [[ "$PLATFORM" == "linux" ]]; then
    process_info=$(ss -tlnp 2>/dev/null | grep ":$port" | head -1)
  fi

  if [[ -n "$process_info" ]]; then
    echo -e "  ${RED}✗${RESET} $process_info"
  fi

  echo -e "\n${BOLD}Solutions:${RESET}"
  echo -e "  ${BLUE}1.${RESET} Stop the conflicting service:"

  case "$port" in
  5432)
    echo -e "     ${DIM}# PostgreSQL${RESET}"
    [[ "$PLATFORM" == "macos" ]] && echo -e "     ${BOLD}brew services stop postgresql${RESET}"
    [[ "$PLATFORM" == "linux" ]] && echo -e "     ${BOLD}sudo systemctl stop postgresql${RESET}"
    ;;
  6379)
    echo -e "     ${DIM}# Redis${RESET}"
    [[ "$PLATFORM" == "macos" ]] && echo -e "     ${BOLD}brew services stop redis${RESET}"
    [[ "$PLATFORM" == "linux" ]] && echo -e "     ${BOLD}sudo systemctl stop redis${RESET}"
    ;;
  8080)
    echo -e "     ${DIM}# Common web services${RESET}"
    echo -e "     ${BOLD}lsof -ti:$port | xargs kill -9${RESET}"
    ;;
  3000 | 3001)
    echo -e "     ${DIM}# Node.js applications${RESET}"
    echo -e "     ${BOLD}npx kill-port $port${RESET}"
    ;;
  esac

  echo -e "\n  ${BLUE}2.${RESET} Or change the port in your ${BOLD}.env.local${RESET}:"
  echo -e "     ${DIM}# Add to .env.local:${RESET}"

  case "$service" in
  postgres)
    echo -e "     ${BOLD}POSTGRES_PORT=$((port + 1000))${RESET}"
    ;;
  redis)
    echo -e "     ${BOLD}REDIS_PORT=$((port + 1000))${RESET}"
    ;;
  hasura)
    echo -e "     ${BOLD}HASURA_PORT=$((port + 1000))${RESET}"
    ;;
  dashboard)
    echo -e "     ${BOLD}DASHBOARD_PORT=$((port + 1000))${RESET}"
    ;;
  esac

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} Use 'lsof -i :$port' to see what's using the port."
}

show_memory_warning() {
  local available_gb="$1"
  local required_gb="${2:-4}"

  format_warning "Low system memory: ${available_gb}GB available (${required_gb}GB recommended)" ""

  echo -e "\n${BOLD}Impact:${RESET}"
  echo -e "  ${YELLOW}⚠${RESET} Services may run slowly"
  echo -e "  ${YELLOW}⚠${RESET} Builds might fail with out-of-memory errors"
  echo -e "  ${YELLOW}⚠${RESET} Database operations could timeout"

  echo -e "\n${BOLD}Solutions:${RESET}"

  case "$PLATFORM" in
  macos)
    echo -e "  ${BLUE}→${RESET} Close unnecessary applications"
    echo -e "  ${BLUE}→${RESET} Increase Docker Desktop memory allocation:"
    echo -e "     ${DIM}Docker Desktop → Settings → Resources → Memory${RESET}"
    ;;
  linux)
    echo -e "  ${BLUE}→${RESET} Check memory usage: ${BOLD}free -h${RESET}"
    echo -e "  ${BLUE}→${RESET} Clear cache: ${BOLD}sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'${RESET}"
    echo -e "  ${BLUE}→${RESET} Add swap space if needed"
    ;;
  windows)
    echo -e "  ${BLUE}→${RESET} Close unnecessary applications"
    echo -e "  ${BLUE}→${RESET} Increase Docker Desktop memory in Settings"
    echo -e "  ${BLUE}→${RESET} Consider WSL2 memory configuration"
    ;;
  esac

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} You can run with reduced services for development:"
  echo -e "  ${BOLD}REDIS_ENABLED=false BULLMQ_ENABLED=false nself start${RESET}"
}

show_disk_space_error() {
  local available_gb="$1"
  local required_gb="${2:-10}"

  format_error "Insufficient disk space: ${available_gb}GB available (${required_gb}GB required)" ""

  echo -e "\n${BOLD}Free up space:${RESET}"
  echo -e "  ${BLUE}1.${RESET} Clean Docker resources:"
  echo -e "     ${BOLD}docker system prune -a --volumes${RESET}"
  echo -e "     ${DIM}This will remove all unused containers, images, and volumes${RESET}"

  echo -e "\n  ${BLUE}2.${RESET} Remove old nself builds:"
  echo -e "     ${BOLD}rm -rf ./generated ./node_modules ./services/*/node_modules${RESET}"

  case "$PLATFORM" in
  macos)
    echo -e "\n  ${BLUE}3.${RESET} macOS specific cleanup:"
    echo -e "     ${BOLD}rm -rf ~/Library/Caches/*${RESET}"
    echo -e "     ${BOLD}brew cleanup${RESET}"
    ;;
  linux)
    echo -e "\n  ${BLUE}3.${RESET} Linux specific cleanup:"
    echo -e "     ${BOLD}sudo apt-get clean${RESET} ${DIM}(or yum/dnf clean all)${RESET}"
    echo -e "     ${BOLD}journalctl --vacuum-size=100M${RESET}"
    ;;
  windows)
    echo -e "\n  ${BLUE}3.${RESET} Windows specific cleanup:"
    echo -e "     Run Disk Cleanup utility"
    echo -e "     ${BOLD}cleanmgr /sageset:1${RESET}"
    ;;
  esac

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} Check disk usage with: ${BOLD}df -h${RESET} (Unix) or ${BOLD}dir${RESET} (Windows)"
}

show_permission_error() {
  local path="$1"

  format_error "Permission denied: $path" ""

  echo -e "\n${BOLD}Solutions:${RESET}"

  case "$PLATFORM" in
  macos | linux)
    echo -e "  ${BLUE}1.${RESET} Fix ownership:"
    echo -e "     ${BOLD}sudo chown -R \$(whoami) $path${RESET}"
    echo -e "\n  ${BLUE}2.${RESET} Fix permissions:"
    echo -e "     ${BOLD}chmod -R 755 $path${RESET}"

    if [[ "$path" =~ docker ]]; then
      echo -e "\n  ${BLUE}3.${RESET} Add user to docker group:"
      echo -e "     ${BOLD}sudo usermod -aG docker \$USER${RESET}"
      echo -e "     ${DIM}Then log out and back in${RESET}"
    fi
    ;;
  windows)
    echo -e "  ${BLUE}1.${RESET} Run as Administrator"
    echo -e "  ${BLUE}2.${RESET} Check file properties → Security tab"
    echo -e "  ${BLUE}3.${RESET} For WSL issues, check Windows Defender settings"
    ;;
  esac

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} Always run 'nself init' in a directory you own."
}

show_network_error() {
  local service="$1"
  local url="${2:-}"

  format_error "Network connection failed: $service" ""

  echo -e "\n${BOLD}Check:${RESET}"
  echo -e "  ${BLUE}✓${RESET} Internet connection: ${BOLD}ping -c 1 google.com${RESET}"
  echo -e "  ${BLUE}✓${RESET} DNS resolution: ${BOLD}nslookup $service${RESET}"
  echo -e "  ${BLUE}✓${RESET} Proxy settings: ${BOLD}echo \$HTTP_PROXY${RESET}"

  if [[ -n "$url" ]]; then
    echo -e "  ${BLUE}✓${RESET} Service status: ${BOLD}curl -I $url${RESET}"
  fi

  echo -e "\n${BOLD}Common fixes:${RESET}"
  echo -e "  ${BLUE}→${RESET} Behind corporate proxy? Set proxy environment variables"
  echo -e "  ${BLUE}→${RESET} Using VPN? Try disconnecting temporarily"
  echo -e "  ${BLUE}→${RESET} Firewall blocking? Check firewall rules"

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} For Docker Hub issues, try: ${BOLD}docker pull hello-world${RESET}"
}

show_dependency_missing_error() {
  local dep="$1"

  format_error "Required dependency not found: $dep" ""

  echo -e "\n${BOLD}Install $dep:${RESET}"

  case "$dep" in
  node | nodejs)
    echo -e "  ${BLUE}→${RESET} Via nvm: ${BOLD}curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash${RESET}"
    echo -e "  ${BLUE}→${RESET} Then: ${BOLD}nvm install --lts${RESET}"
    [[ "$PKG_MANAGER" == "brew" ]] && echo -e "  ${BLUE}→${RESET} Or via Homebrew: ${BOLD}brew install node${RESET}"
    [[ "$PKG_MANAGER" == "apt" ]] && echo -e "  ${BLUE}→${RESET} Or via apt: ${BOLD}sudo apt-get install nodejs npm${RESET}"
    ;;
  go | golang)
    echo -e "  ${BLUE}→${RESET} Download: ${UNDERLINE}https://go.dev/dl/${RESET}"
    [[ "$PKG_MANAGER" == "brew" ]] && echo -e "  ${BLUE}→${RESET} Or via Homebrew: ${BOLD}brew install go${RESET}"
    [[ "$PKG_MANAGER" == "apt" ]] && echo -e "  ${BLUE}→${RESET} Or via apt: ${BOLD}sudo apt-get install golang${RESET}"
    ;;
  python | python3)
    [[ "$PKG_MANAGER" == "brew" ]] && echo -e "  ${BLUE}→${RESET} Via Homebrew: ${BOLD}brew install python3${RESET}"
    [[ "$PKG_MANAGER" == "apt" ]] && echo -e "  ${BLUE}→${RESET} Via apt: ${BOLD}sudo apt-get install python3 python3-pip${RESET}"
    echo -e "  ${BLUE}→${RESET} Or via pyenv for version management"
    ;;
  git)
    [[ "$PKG_MANAGER" == "brew" ]] && echo -e "  ${BLUE}→${RESET} Via Homebrew: ${BOLD}brew install git${RESET}"
    [[ "$PKG_MANAGER" == "apt" ]] && echo -e "  ${BLUE}→${RESET} Via apt: ${BOLD}sudo apt-get install git${RESET}"
    [[ "$PLATFORM" == "windows" ]] && echo -e "  ${BLUE}→${RESET} Download: ${UNDERLINE}https://git-scm.com/download/win${RESET}"
    ;;
  esac

  echo -e "\n${YELLOW}⚡ Pro Tip:${RESET} Dependencies will run in Docker even if not installed locally."
}

show_success_banner() {
  local message="$1"

  echo
  echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${RESET}"
  echo -e "${GREEN}║${RESET}                                                          ${GREEN}║${RESET}"
  echo -e "${GREEN}║${RESET}   ${GREEN}✨ SUCCESS ✨${RESET}                                        ${GREEN}║${RESET}"
  printf "${GREEN}║${RESET}   %-54s${GREEN}║${RESET}\n" "$message"
  echo -e "${GREEN}║${RESET}                                                          ${GREEN}║${RESET}"
  echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${RESET}"
  echo
}

show_welcome_message() {
  echo
  echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${RESET}"
  echo -e "${BLUE}║${RESET}                                                          ${BLUE}║${RESET}"
  echo -e "${BLUE}║${RESET}   ${BOLD}Welcome to nself${RESET} - Modern Full-Stack Platform       ${BLUE}║${RESET}"
  echo -e "${BLUE}║${RESET}                                                          ${BLUE}║${RESET}"
  echo -e "${BLUE}║${RESET}   ${DIM}Build production-ready applications with ease${RESET}       ${BLUE}║${RESET}"
  echo -e "${BLUE}║${RESET}                                                          ${BLUE}║${RESET}"
  echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${RESET}"
  echo
}

export -f show_docker_not_running_error
export -f show_port_conflict_error
export -f show_memory_warning
export -f show_disk_space_error
export -f show_permission_error
export -f show_network_error
export -f show_dependency_missing_error
export -f show_success_banner
export -f show_welcome_message
