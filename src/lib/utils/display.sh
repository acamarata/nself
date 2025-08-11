#!/usr/bin/env bash
# display.sh - Centralized display utilities for consistent output

# Mark as sourced to prevent double-sourcing
export DISPLAY_SOURCED=1

# Color definitions
COLOR_RESET='\033[0m'
COLOR_RED='\033[0;31m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_BLUE='\033[0;34m'
COLOR_MAGENTA='\033[0;35m'
COLOR_CYAN='\033[0;36m'
COLOR_WHITE='\033[0;37m'
COLOR_BOLD='\033[1m'
COLOR_DIM='\033[2m'

# Icons
ICON_SUCCESS="✓"
ICON_FAILURE="✗"
ICON_WARNING="⚠"
ICON_INFO="ℹ"
ICON_ARROW="→"
ICON_BULLET="•"

# Check for NO_COLOR environment variable
if [[ -n "${NO_COLOR:-}" ]] || [[ ! -t 1 ]]; then
    # Can't unset readonly variables, so redefine them as empty
    COLOR_RESET=""
    COLOR_RED=""
    COLOR_GREEN=""
    COLOR_YELLOW=""
    COLOR_BLUE=""
    COLOR_MAGENTA=""
    COLOR_CYAN=""
    COLOR_WHITE=""
    COLOR_BOLD=""
    COLOR_DIM=""
fi

# Logging functions - All commands must use these
log_info() {
    echo -e "${COLOR_BLUE}[INFO]${COLOR_RESET} $1"
}

log_success() {
    echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} $1"
}

log_warning() {
    echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $1" >&2
}

log_error() {
    echo -e "${COLOR_RED}[ERROR]${COLOR_RESET} $1" >&2
}

log_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo -e "${COLOR_MAGENTA}[DEBUG]${COLOR_RESET} $1"
}

# Header and section formatting
show_header() {
    local title="$1"
    local width=60
    local padding=$(( (width - ${#title}) / 2 ))
    
    echo
    echo "╔$(printf '═%.0s' $(seq 1 $width))╗"
    printf "║%*s%s%*s║\n" $padding "" "$title" $((width - padding - ${#title})) ""
    echo "╚$(printf '═%.0s' $(seq 1 $width))╝"
    echo
}

# Alias for compatibility
log_header() {
    show_header "$@"
}

show_section() {
    local title="$1"
    echo
    echo -e "${COLOR_BOLD}▶ $title${COLOR_RESET}"
    echo "$(printf '─%.0s' $(seq 1 ${#title}))"
}

# Table formatting
show_table_header() {
    local -a headers=("$@")
    
    printf "┌"
    for header in "${headers[@]}"; do
        printf "─%.0s" $(seq 1 $((${#header} + 2)))
        printf "┬"
    done
    printf "\b┐\n"
    
    printf "│"
    for header in "${headers[@]}"; do
        printf " %-${#header}s │" "$header"
    done
    printf "\n"
    
    printf "├"
    for header in "${headers[@]}"; do
        printf "─%.0s" $(seq 1 $((${#header} + 2)))
        printf "┼"
    done
    printf "\b┤\n"
}

show_table_row() {
    printf "│"
    for value in "$@"; do
        printf " %s │" "$value"
    done
    printf "\n"
}

show_table_footer() {
    local -a headers=("$@")
    printf "└"
    for header in "${headers[@]}"; do
        printf "─%.0s" $(seq 1 $((${#header} + 2)))
        printf "┴"
    done
    printf "\b┘\n"
}

# Box drawing
draw_box() {
    local message="$1"
    local type="${2:-info}"
    local width=$((${#message} + 4))
    
    case "$type" in
        success) local color="$COLOR_GREEN" ;;
        error) local color="$COLOR_RED" ;;
        warning) local color="$COLOR_YELLOW" ;;
        *) local color="$COLOR_BLUE" ;;
    esac
    
    echo -e "${color}┌$(printf '─%.0s' $(seq 1 $width))┐${COLOR_RESET}"
    echo -e "${color}│  $message  │${COLOR_RESET}"
    echo -e "${color}└$(printf '─%.0s' $(seq 1 $width))┘${COLOR_RESET}"
}

# Strip colors for log file output
strip_colors() {
    sed 's/\x1b\[[0-9;]*m//g'
}

# Export all functions
export -f log_info log_success log_warning log_error log_debug log_header
export -f show_header show_section
export -f show_table_header show_table_row show_table_footer
export -f draw_box strip_colors