#!/usr/bin/env bash
# Install coverage tools for nself test coverage
#
# Installs kcov, lcov, and other coverage dependencies

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    printf "${BLUE}ℹ${NC} %s\n" "$1"
}

log_success() {
    printf "${GREEN}✓${NC} %s\n" "$1"
}

log_warning() {
    printf "${YELLOW}⚠${NC} %s\n" "$1"
}

log_error() {
    printf "${RED}✗${NC} %s\n" "$1"
}

# Detect OS
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ -f /etc/os-release ]]; then
        . /etc/os-release
        echo "$ID"
    else
        echo "unknown"
    fi
}

# Install kcov
install_kcov() {
    local os=$(detect_os)

    log_info "Installing kcov..."

    case "$os" in
        macos)
            if command -v brew >/dev/null 2>&1; then
                brew install kcov
                log_success "kcov installed via Homebrew"
            else
                log_error "Homebrew not found. Install from: https://brew.sh"
                return 1
            fi
            ;;

        ubuntu|debian)
            sudo apt-get update
            sudo apt-get install -y kcov
            log_success "kcov installed via apt"
            ;;

        fedora|rhel|centos)
            sudo dnf install -y kcov
            log_success "kcov installed via dnf"
            ;;

        *)
            log_warning "Unknown OS: $os"
            log_info "Install kcov manually: https://github.com/SimonKagstrom/kcov"
            return 1
            ;;
    esac

    return 0
}

# Install lcov
install_lcov() {
    local os=$(detect_os)

    log_info "Installing lcov..."

    case "$os" in
        macos)
            if command -v brew >/dev/null 2>&1; then
                brew install lcov
                log_success "lcov installed via Homebrew"
            else
                log_warning "Homebrew not found, skipping lcov"
            fi
            ;;

        ubuntu|debian)
            sudo apt-get install -y lcov
            log_success "lcov installed via apt"
            ;;

        fedora|rhel|centos)
            sudo dnf install -y lcov
            log_success "lcov installed via dnf"
            ;;

        *)
            log_warning "lcov installation skipped for $os"
            ;;
    esac
}

# Install jq
install_jq() {
    local os=$(detect_os)

    log_info "Installing jq..."

    case "$os" in
        macos)
            if command -v brew >/dev/null 2>&1; then
                brew install jq
                log_success "jq installed via Homebrew"
            else
                log_warning "Homebrew not found, skipping jq"
            fi
            ;;

        ubuntu|debian)
            sudo apt-get install -y jq
            log_success "jq installed via apt"
            ;;

        fedora|rhel|centos)
            sudo dnf install -y jq
            log_success "jq installed via dnf"
            ;;

        *)
            log_warning "jq installation skipped for $os"
            ;;
    esac
}

# Install bc (for calculations)
install_bc() {
    local os=$(detect_os)

    if command -v bc >/dev/null 2>&1; then
        log_success "bc already installed"
        return 0
    fi

    log_info "Installing bc..."

    case "$os" in
        macos)
            log_success "bc included with macOS"
            ;;

        ubuntu|debian)
            sudo apt-get install -y bc
            log_success "bc installed via apt"
            ;;

        fedora|rhel|centos)
            sudo dnf install -y bc
            log_success "bc installed via dnf"
            ;;

        *)
            log_warning "bc installation skipped for $os"
            ;;
    esac
}

# Verify installations
verify_tools() {
    log_info "Verifying installations..."

    local all_ok=true

    # Check kcov
    if command -v kcov >/dev/null 2>&1; then
        log_success "kcov: $(kcov --version 2>&1 | head -1)"
    else
        log_warning "kcov: Not found (optional but recommended)"
        all_ok=false
    fi

    # Check lcov
    if command -v lcov >/dev/null 2>&1; then
        log_success "lcov: $(lcov --version 2>&1 | head -1)"
    else
        log_warning "lcov: Not found (optional)"
    fi

    # Check jq
    if command -v jq >/dev/null 2>&1; then
        log_success "jq: $(jq --version 2>&1)"
    else
        log_warning "jq: Not found (optional)"
    fi

    # Check bc
    if command -v bc >/dev/null 2>&1; then
        log_success "bc: Available"
    else
        log_warning "bc: Not found (required for calculations)"
        all_ok=false
    fi

    if [[ "$all_ok" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    printf "\n"
    log_info "=== Installing Coverage Tools ==="
    printf "\n"

    local os=$(detect_os)
    log_info "Detected OS: $os"
    printf "\n"

    # Install tools
    install_kcov || log_warning "kcov installation failed"
    install_lcov || true
    install_jq || true
    install_bc || true

    printf "\n"
    verify_tools

    printf "\n"
    log_success "=== Installation Complete ==="
    printf "\n"

    printf "Next steps:\n"
    printf "  1. Run: ./src/scripts/coverage/collect-coverage.sh\n"
    printf "  2. View: ./src/scripts/coverage/generate-coverage-report.sh\n"
    printf "  3. Check: ./src/scripts/coverage/verify-coverage.sh\n"
    printf "\n"
}

# Run main
main "$@"
