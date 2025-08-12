# NSELF Output Formatting Guide

## Overview

This guide defines standards for all user-facing output from the NSELF tool, ensuring consistent, professional, and accessible communication.

## Standardized Header Format (60 Characters)

As of v0.3.0, nself uses a standardized 60-character wide format for all headers and structured output to ensure consistency across terminal displays, configuration files, and documentation.

### Header Specifications
- **Total Width**: 60 characters
- **Content Width**: 56 characters (60 - 2 borders - 2 spaces)
- **Border Style**: Unicode box-drawing characters
- **Implementation**: `src/lib/utils/header.sh`

### Example Header
```
╔══════════════════════════════════════════════════════════╗
║                    NSELF BUILD SYSTEM                    ║
║                                                          ║
║  Content is left-aligned with automatic word wrapping    ║
║  to ensure proper display within 60 characters.          ║
╚══════════════════════════════════════════════════════════╝
```

## Color Scheme

### Standard Colors
```bash
# Color definitions
readonly COLOR_RESET='\033[0m'
readonly COLOR_RED='\033[0;31m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_YELLOW='\033[0;33m'
readonly COLOR_BLUE='\033[0;34m'
readonly COLOR_MAGENTA='\033[0;35m'
readonly COLOR_CYAN='\033[0;36m'
readonly COLOR_WHITE='\033[0;37m'
readonly COLOR_BOLD='\033[1m'
readonly COLOR_DIM='\033[2m'
```

### Color Usage Guidelines

| Color | Usage | Example |
|-------|-------|---------|
| **Blue** | Information, progress | `[INFO]`, spinner animations |
| **Green** | Success, completion | `[SUCCESS]`, `✓ Done` |
| **Yellow** | Warnings, caution | `[WARNING]`, retry messages |
| **Red** | Errors, failures | `[ERROR]`, `✗ Failed` |
| **Magenta** | Debug information | `[DEBUG]`, verbose output |
| **Cyan** | Highlights, URLs | Service URLs, important values |
| **White** | Normal text | General output |
| **Bold** | Headers, emphasis | Section titles |

## Message Types

### Information Messages
```bash
log_info() {
    echo -e "${COLOR_BLUE}[INFO]${COLOR_RESET} $1"
}

# Usage
log_info "Starting services..."
log_info "Loading configuration from .env.local"
```

Output:
```
[INFO] Starting services...
[INFO] Loading configuration from .env.local
```

### Success Messages
```bash
log_success() {
    echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} $1"
}

# Usage
log_success "All services started successfully"
log_success "Configuration validated"
```

Output:
```
[SUCCESS] All services started successfully
[SUCCESS] Configuration validated
```

### Warning Messages
```bash
log_warning() {
    echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $1" >&2
}

# Usage
log_warning "Port 8080 already in use, trying alternative"
log_warning "Using default configuration"
```

Output:
```
[WARNING] Port 8080 already in use, trying alternative
[WARNING] Using default configuration
```

### Error Messages
```bash
log_error() {
    echo -e "${COLOR_RED}[ERROR]${COLOR_RESET} $1" >&2
}

# Usage
log_error "Docker is not running"
log_error "Configuration file not found"
```

Output:
```
[ERROR] Docker is not running
[ERROR] Configuration file not found
```

### Debug Messages
```bash
log_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && \
        echo -e "${COLOR_MAGENTA}[DEBUG]${COLOR_RESET} $1"
}

# Usage (only shown when DEBUG=true)
log_debug "Container ID: abc123"
log_debug "Environment: development"
```

Output:
```
[DEBUG] Container ID: abc123
[DEBUG] Environment: development
```

## Progress Indicators

### Spinner Animations
```bash
show_spinner() {
    local pid=$1
    local message=${2:-"Processing..."}
    local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local i=0
    
    tput civis  # Hide cursor
    while kill -0 "$pid" 2>/dev/null; do
        i=$(( (i+1) % 10 ))
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s" "${spin:$i:1}" "$message"
        sleep 0.1
    done
    printf "\r\033[K"  # Clear line
    tput cnorm  # Show cursor
}
```

### Progress Bar
```bash
show_progress() {
    local current=$1
    local total=$2
    local width=${3:-50}
    local percent=$((current * 100 / total))
    local filled=$((width * current / total))
    
    printf "\r["
    printf "%${filled}s" | tr ' ' '='
    printf "%$((width - filled))s" | tr ' ' '-'
    printf "] %3d%%" "$percent"
    
    [[ $current -eq $total ]] && echo
}
```

Output:
```
[==========================-------------------------] 50%
```

### Step Counter
```bash
show_step() {
    local current=$1
    local total=$2
    local description=$3
    
    echo -e "${COLOR_BOLD}Step $current/$total:${COLOR_RESET} $description"
}

# Usage
show_step 1 5 "Checking prerequisites"
show_step 2 5 "Building Docker images"
```

Output:
```
Step 1/5: Checking prerequisites
Step 2/5: Building Docker images
```

## Headers and Sections

### Main Headers (Standardized)
```bash
# Source the standardized header utilities
source "$SCRIPT_DIR/../lib/utils/header.sh"

# Simple header
show_header "NSELF INITIALIZATION"

# Header with content
show_header "BUILD COMPLETE" "All services have been configured and are ready to start."
```

Output:
```
╔══════════════════════════════════════════════════════════╗
║                  NSELF INITIALIZATION                    ║
╚══════════════════════════════════════════════════════════╝

╔══════════════════════════════════════════════════════════╗
║                     BUILD COMPLETE                       ║
║                                                          ║
║  All services have been configured and are ready to      ║
║  start.                                                  ║
╚══════════════════════════════════════════════════════════╝
```

### Section Headers
```bash
show_section() {
    local title="$1"
    echo
    echo -e "${COLOR_BOLD}▶ $title${COLOR_RESET}"
    echo "$(printf '─%.0s' $(seq 1 ${#title}))"
}

# Usage
show_section "Service Configuration"
```

Output:
```
▶ Service Configuration
───────────────────────
```

## Tables and Lists

### Simple Table
```bash
show_table() {
    local -a headers=("$@")
    
    # Header
    printf "┌"
    for header in "${headers[@]}"; do
        printf "─%.0s" $(seq 1 $((${#header} + 2)))
        printf "┬"
    done
    printf "\b┐\n"
    
    # Header content
    printf "│"
    for header in "${headers[@]}"; do
        printf " %-${#header}s │" "$header"
    done
    printf "\n"
    
    # Separator
    printf "├"
    for header in "${headers[@]}"; do
        printf "─%.0s" $(seq 1 $((${#header} + 2)))
        printf "┼"
    done
    printf "\b┤\n"
}

# Usage
show_table "Service" "Status" "Port"
```

Output:
```
┌─────────┬────────┬──────┐
│ Service │ Status │ Port │
├─────────┼────────┼──────┤
```

### Status Table
```bash
show_status_table() {
    echo "┌──────────────────┬──────────┬────────┐"
    echo "│ Service          │ Status   │ Health │"
    echo "├──────────────────┼──────────┼────────┤"
    echo "│ PostgreSQL       │ ${COLOR_GREEN}Running${COLOR_RESET}  │ ${COLOR_GREEN}✓${COLOR_RESET}      │"
    echo "│ Hasura           │ ${COLOR_GREEN}Running${COLOR_RESET}  │ ${COLOR_GREEN}✓${COLOR_RESET}      │"
    echo "│ MinIO            │ ${COLOR_YELLOW}Starting${COLOR_RESET} │ ${COLOR_YELLOW}⋯${COLOR_RESET}      │"
    echo "│ Redis            │ ${COLOR_RED}Stopped${COLOR_RESET}  │ ${COLOR_RED}✗${COLOR_RESET}      │"
    echo "└──────────────────┴──────────┴────────┘"
}
```

### Bullet Lists
```bash
show_list() {
    local bullet="${1:-•}"
    shift
    for item in "$@"; do
        echo "  $bullet $item"
    done
}

# Usage
show_list "•" "PostgreSQL Database" "Hasura GraphQL" "MinIO Storage"
```

Output:
```
  • PostgreSQL Database
  • Hasura GraphQL
  • MinIO Storage
```

## Icons and Symbols

### Status Icons
```bash
# Success/Failure
readonly ICON_SUCCESS="✓"
readonly ICON_FAILURE="✗"
readonly ICON_WARNING="⚠"
readonly ICON_INFO="ℹ"
readonly ICON_PENDING="⋯"

# Arrows
readonly ICON_ARROW_RIGHT="→"
readonly ICON_ARROW_LEFT="←"
readonly ICON_ARROW_UP="↑"
readonly ICON_ARROW_DOWN="↓"

# Bullets
readonly ICON_BULLET="•"
readonly ICON_TRIANGLE="▶"
readonly ICON_SQUARE="■"
readonly ICON_CIRCLE="●"
```

### Usage Examples
```bash
echo -e "${COLOR_GREEN}${ICON_SUCCESS}${COLOR_RESET} Configuration valid"
echo -e "${COLOR_RED}${ICON_FAILURE}${COLOR_RESET} Connection failed"
echo -e "${COLOR_YELLOW}${ICON_WARNING}${COLOR_RESET} Low memory warning"
echo -e "${COLOR_BLUE}${ICON_INFO}${COLOR_RESET} Using default settings"
```

## Interactive Elements

### Prompts
```bash
prompt_input() {
    local prompt="$1"
    local default="$2"
    local input
    
    if [[ -n "$default" ]]; then
        printf "${COLOR_CYAN}?${COLOR_RESET} %s [%s]: " "$prompt" "$default"
        read -r input
        echo "${input:-$default}"
    else
        printf "${COLOR_CYAN}?${COLOR_RESET} %s: " "$prompt"
        read -r input
        echo "$input"
    fi
}

# Usage
domain=$(prompt_input "Enter domain" "localhost")
```

Output:
```
? Enter domain [localhost]: 
```

### Confirmation
```bash
confirm() {
    local message="$1"
    local response
    
    printf "${COLOR_YELLOW}⚠${COLOR_RESET} %s [y/N]: " "$message"
    read -r response
    [[ "$response" =~ ^[Yy]$ ]]
}

# Usage
if confirm "Continue with installation?"; then
    # Proceed
fi
```

Output:
```
⚠ Continue with installation? [y/N]: 
```

### Selection Menu
```bash
show_menu() {
    local prompt="$1"
    shift
    local options=("$@")
    local selected=0
    
    echo -e "${COLOR_CYAN}$prompt${COLOR_RESET}"
    for i in "${!options[@]}"; do
        echo "  $((i+1))) ${options[$i]}"
    done
    
    read -p "Select option: " selected
    echo "$selected"
}

# Usage
choice=$(show_menu "Select environment:" "Development" "Staging" "Production")
```

Output:
```
Select environment:
  1) Development
  2) Staging
  3) Production
Select option: 
```

## Service URLs Display

### URL Formatting
```bash
show_service_urls() {
    echo
    echo -e "${COLOR_BOLD}Service URLs:${COLOR_RESET}"
    echo
    echo -e "  ${ICON_ARROW_RIGHT} GraphQL:    ${COLOR_CYAN}https://api.${BASE_DOMAIN}${COLOR_RESET}"
    echo -e "  ${ICON_ARROW_RIGHT} Auth:       ${COLOR_CYAN}https://auth.${BASE_DOMAIN}${COLOR_RESET}"
    echo -e "  ${ICON_ARROW_RIGHT} Storage:    ${COLOR_CYAN}https://storage.${BASE_DOMAIN}${COLOR_RESET}"
    echo -e "  ${ICON_ARROW_RIGHT} Dashboard:  ${COLOR_CYAN}https://dashboard.${BASE_DOMAIN}${COLOR_RESET}"
    echo
}
```

Output:
```
Service URLs:

  → GraphQL:    https://api.example.com
  → Auth:       https://auth.example.com
  → Storage:    https://storage.example.com
  → Dashboard:  https://dashboard.example.com
```

## Error Display

### Detailed Error
```bash
show_error_details() {
    local error_type="$1"
    local error_message="$2"
    local suggestion="$3"
    
    echo
    echo -e "${COLOR_RED}━━━ ERROR ━━━${COLOR_RESET}"
    echo -e "${COLOR_BOLD}Type:${COLOR_RESET} $error_type"
    echo -e "${COLOR_BOLD}Message:${COLOR_RESET} $error_message"
    [[ -n "$suggestion" ]] && echo -e "${COLOR_BOLD}Suggestion:${COLOR_RESET} $suggestion"
    echo -e "${COLOR_RED}━━━━━━━━━━━━━${COLOR_RESET}"
    echo
}

# Usage
show_error_details "Docker" "Docker daemon is not running" "Start Docker Desktop or run: sudo systemctl start docker"
```

Output:
```
━━━ ERROR ━━━
Type: Docker
Message: Docker daemon is not running
Suggestion: Start Docker Desktop or run: sudo systemctl start docker
━━━━━━━━━━━━━
```

## Completion Messages

### Success Summary
```bash
show_success_summary() {
    local duration="$1"
    
    echo
    echo -e "${COLOR_GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}"
    echo -e "${COLOR_GREEN}${ICON_SUCCESS} SUCCESS: All operations completed${COLOR_RESET}"
    echo -e "${COLOR_GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}"
    echo
    echo "  Duration: $duration seconds"
    echo "  Status:   All services healthy"
    echo
    echo "Next steps:"
    echo "  1. Access the dashboard at https://dashboard.${BASE_DOMAIN}"
    echo "  2. Check service status with: nself status"
    echo "  3. View logs with: nself logs"
    echo
}
```

## Verbose Output

### Debug Mode Formatting
```bash
# When DEBUG=true or --verbose flag
if [[ "${VERBOSE:-false}" == "true" ]]; then
    echo -e "${COLOR_DIM}[$(date '+%H:%M:%S')] Executing: docker-compose up -d${COLOR_RESET}"
    echo -e "${COLOR_DIM}[$(date '+%H:%M:%S')] Working directory: $(pwd)${COLOR_RESET}"
    echo -e "${COLOR_DIM}[$(date '+%H:%M:%S')] Environment: development${COLOR_RESET}"
fi
```

## Accessibility Considerations

### NO_COLOR Support
```bash
# Respect NO_COLOR environment variable
if [[ -n "${NO_COLOR:-}" ]]; then
    unset COLOR_RESET COLOR_RED COLOR_GREEN COLOR_YELLOW
    unset COLOR_BLUE COLOR_MAGENTA COLOR_CYAN COLOR_WHITE
    unset COLOR_BOLD COLOR_DIM
fi
```

### Terminal Detection
```bash
# Check if output is to terminal
if [[ ! -t 1 ]]; then
    # Disable colors for non-terminal output
    unset COLOR_RESET COLOR_RED COLOR_GREEN COLOR_YELLOW
    unset COLOR_BLUE COLOR_MAGENTA COLOR_CYAN COLOR_WHITE
    unset COLOR_BOLD COLOR_DIM
fi
```

### Screen Reader Friendly
```bash
# Use clear text indicators instead of only colors
echo "[SUCCESS] Operation completed"  # Good
echo "✓"  # Bad for screen readers

# Provide text alternatives
echo "Status: Running (Green)"  # Good for accessibility
```

## Output Redirection

### Log File Format
```bash
# When writing to log files, exclude color codes
log_to_file() {
    local message="$1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') $message" >> "$LOG_FILE"
}

# Strip colors for file output
strip_colors() {
    sed 's/\x1b\[[0-9;]*m//g'
}

# Usage
echo -e "${COLOR_GREEN}Success${COLOR_RESET}" | strip_colors >> output.log
```

## Best Practices

### Consistency Rules
1. **Always use defined functions** for output (log_info, log_error, etc.)
2. **Consistent icon usage** across all commands
3. **Proper error stream** usage (stderr for errors/warnings)
4. **Clear visual hierarchy** with headers and sections
5. **Accessible output** with text alternatives

### Performance Guidelines
1. **Minimize output in loops** to prevent terminal flooding
2. **Use carriage returns** for progress updates
3. **Buffer output** when possible
4. **Clear lines properly** after temporary output
5. **Respect quiet mode** flags

### User Experience
1. **Clear, concise messages** without technical jargon
2. **Actionable error messages** with suggestions
3. **Progress indication** for long operations
4. **Summary information** after completion
5. **Respect user preferences** (colors, verbosity)

## Summary

This formatting guide ensures:
- **Consistency**: Uniform output across all commands
- **Clarity**: Clear communication with users
- **Accessibility**: Support for various terminal environments
- **Professionalism**: Polished, modern appearance
- **Usability**: Easy to read and understand output