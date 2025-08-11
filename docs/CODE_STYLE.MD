# NSELF Code Style Guide

## Shell Script Standards

### File Structure

#### File Header
```bash
#!/bin/bash
# filename.sh - Brief description of file purpose
#
# Usage: filename.sh [options] [arguments]
# 
# Dependencies:
#   - List any external dependencies
#   - Required environment variables
#
# Author: Development Team
# Version: 1.0.0
```

#### Strict Mode
Always start scripts with:
```bash
set -euo pipefail
IFS=$'\n\t'
```

### Naming Conventions

#### Files
- **Scripts**: `lowercase-with-hyphens.sh`
- **Libraries**: `module-name.sh`
- **Tests**: `test-module-name.sh`
- **Templates**: `name.template` or `name.sh.template`

#### Variables
```bash
# Constants (uppercase with underscores)
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly DEFAULT_TIMEOUT=30
readonly MAX_RETRIES=3

# Global variables (uppercase with underscores)
PROJECT_NAME="nself"
BASE_DOMAIN=""

# Local variables (lowercase with underscores)
local user_input=""
local is_valid=false
local retry_count=0

# Environment variables (uppercase with underscores)
export NSELF_DEBUG="${NSELF_DEBUG:-false}"
export NSELF_CONFIG_DIR="${NSELF_CONFIG_DIR:-$HOME/.nself}"
```

#### Functions
```bash
# Function names (lowercase with underscores)
validate_input() {
    local input="$1"
    # Function implementation
}

# Command functions (prefix with cmd_)
cmd_up() {
    # Command implementation
}

# Private functions (prefix with underscore)
_internal_helper() {
    # Internal implementation
}
```

### Code Organization

#### Function Documentation
```bash
# validate_configuration - Validates the configuration file
# 
# Arguments:
#   $1 - Configuration file path
#   $2 - Validation mode (strict|lenient)
# 
# Returns:
#   0 - Configuration is valid
#   1 - Configuration is invalid
#   2 - Configuration file not found
# 
# Example:
#   validate_configuration "/path/to/config" "strict"
validate_configuration() {
    local config_file="${1:?Configuration file required}"
    local mode="${2:-strict}"
    
    # Implementation
}
```

#### Error Handling
```bash
# Always check command success
if ! command_that_might_fail; then
    log_error "Command failed: description"
    return 1
fi

# Use explicit error checking
if [[ ! -f "$file_path" ]]; then
    handle_error 1 "File not found: $file_path" "file_error"
fi

# Trap errors for cleanup
trap 'cleanup_on_error $?' ERR
trap 'cleanup_on_exit' EXIT
```

### Best Practices

#### Variable Quoting
```bash
# Always quote variables
echo "$variable"              # Good
echo $variable                # Bad

# Exception: numeric comparisons
if [[ $count -gt 10 ]]; then  # OK for numbers

# Array expansion
"${array[@]}"                 # Good - preserves elements
${array[@]}                   # Bad - word splitting

# Default values
"${VAR:-default}"             # Good
${VAR:-default}               # Bad
```

#### Command Substitution
```bash
# Use $() instead of backticks
result="$(command)"           # Good
result=`command`              # Bad

# Quote command substitutions
file_count="$(ls -1 | wc -l)" # Good
file_count=$(ls -1 | wc -l)   # Bad when output contains spaces
```

#### Conditionals
```bash
# Use [[ ]] for conditionals
if [[ -f "$file" ]]; then     # Good
if [ -f "$file" ]; then       # Acceptable but less powerful

# String comparisons
if [[ "$string1" == "$string2" ]]; then  # Good
if [ "$string1" = "$string2" ]; then     # Acceptable

# Numeric comparisons
if [[ $num -eq 0 ]]; then     # Good
if (( num == 0 )); then       # Good for arithmetic

# Multiple conditions
if [[ -f "$file" && -r "$file" ]]; then  # Good
if [ -f "$file" ] && [ -r "$file" ]; then # More portable
```

#### Loops
```bash
# Iterate over arrays
for item in "${array[@]}"; do
    process_item "$item"
done

# Read files line by line
while IFS= read -r line; do
    process_line "$line"
done < "$file"

# Process command output
while IFS= read -r line; do
    process_line "$line"
done < <(command)

# Iterate with counter
for ((i=0; i<${#array[@]}; i++)); do
    echo "Index $i: ${array[$i]}"
done
```

### Output Formatting

#### Logging Functions
```bash
# Consistent logging format
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
    [[ "${DEBUG:-false}" == "true" ]] && \
        echo -e "${COLOR_MAGENTA}[DEBUG]${COLOR_RESET} $1"
}
```

#### User Interaction
```bash
# Prompt for input
prompt_user() {
    local prompt="$1"
    local variable_name="$2"
    local default="${3:-}"
    
    if [[ -n "$default" ]]; then
        read -p "$prompt [$default]: " input
        input="${input:-$default}"
    else
        read -p "$prompt: " input
    fi
    
    eval "$variable_name='$input'"
}

# Confirmation prompts
confirm() {
    local message="$1"
    local response
    
    read -p "$message [y/N]: " response
    [[ "$response" =~ ^[Yy]$ ]]
}

# Password input
read_secret() {
    local prompt="$1"
    local variable_name="$2"
    
    read -s -p "$prompt: " secret
    echo  # New line after hidden input
    eval "$variable_name='$secret'"
}
```

### Constants and Magic Values

#### Define Constants
```bash
# Time constants
readonly STARTUP_TIMEOUT=60
readonly HEALTH_CHECK_INTERVAL=2
readonly RETRY_DELAY=5

# Size constants
readonly MAX_LOG_SIZE=$((10 * 1024 * 1024))  # 10MB
readonly BUFFER_SIZE=4096

# Paths
readonly CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/nself"
readonly CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/nself"
readonly DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/nself"

# Exit codes
readonly EXIT_SUCCESS=0
readonly EXIT_GENERAL_ERROR=1
readonly EXIT_MISUSE=2
readonly EXIT_CANT_EXEC=126
readonly EXIT_NOT_FOUND=127
```

### Function Patterns

#### Validation Functions
```bash
# Return 0 for valid, 1 for invalid
is_valid_domain() {
    local domain="$1"
    [[ "$domain" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]
}

is_valid_email() {
    local email="$1"
    [[ "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]
}

is_valid_port() {
    local port="$1"
    [[ "$port" =~ ^[0-9]+$ ]] && [[ $port -ge 1 ]] && [[ $port -le 65535 ]]
}
```

#### Resource Management
```bash
# Always cleanup resources
cleanup() {
    local exit_code=$?
    
    # Remove temporary files
    [[ -n "${TEMP_DIR:-}" ]] && rm -rf "$TEMP_DIR"
    
    # Stop background processes
    [[ -n "${SPINNER_PID:-}" ]] && kill "$SPINNER_PID" 2>/dev/null
    
    # Restore terminal state
    tput cnorm  # Show cursor
    
    exit "$exit_code"
}

trap cleanup EXIT INT TERM
```

#### Retry Logic
```bash
# Retry with exponential backoff
retry_with_backoff() {
    local max_attempts="${1:-3}"
    local delay="${2:-1}"
    local max_delay="${3:-60}"
    shift 3
    
    local attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        if "$@"; then
            return 0
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            log_warning "Attempt $attempt failed, retrying in ${delay}s..."
            sleep "$delay"
            delay=$((delay * 2))
            [[ $delay -gt $max_delay ]] && delay=$max_delay
        fi
        
        ((attempt++))
    done
    
    log_error "All $max_attempts attempts failed"
    return 1
}
```

### Testing Standards

#### Test File Structure
```bash
#!/bin/bash
# test-module-name.sh - Tests for module-name

source "$(dirname "$0")/../test-framework.sh"
source "$(dirname "$0")/../../module-name.sh"

# Test basic functionality
test_function_basic() {
    local result
    result=$(function_name "input")
    assert_equals "$result" "expected" "Basic function test"
}

# Test error handling
test_function_error() {
    local result
    result=$(function_name "invalid" 2>&1) || true
    assert_contains "$result" "ERROR" "Should show error"
}

# Run all tests
run_tests "$@"
```

### Documentation Standards

#### Inline Comments
```bash
# Use comments to explain WHY, not WHAT
# Bad: Increment counter
((counter++))

# Good: Retry up to MAX_RETRIES times to handle transient failures
((counter++))

# Complex logic should be explained
# Calculate exponential backoff with jitter to prevent thundering herd
delay=$((base_delay * (2 ** attempt) + RANDOM % jitter))
```

#### TODO Comments
```bash
# TODO: Implement caching for performance improvement
# FIXME: Handle edge case when service is partially started
# HACK: Workaround for Docker Compose v2 compatibility
# NOTE: This assumes PostgreSQL 14+ for certain features
```

### Shell Compatibility

#### Bash Version
```bash
# Require minimum Bash version
if [[ "${BASH_VERSION%%.*}" -lt 4 ]]; then
    echo "Error: Bash 4.0+ required" >&2
    exit 1
fi
```

#### POSIX Compliance (when needed)
```bash
# Use POSIX features for portability
# POSIX compliant
if [ -f "$file" ]; then
    echo "File exists"
fi

# Bash specific (avoid in portable scripts)
if [[ -f "$file" ]]; then
    echo "File exists"
fi
```

### Security Practices

#### Input Validation
```bash
# Sanitize user input
sanitize_input() {
    local input="$1"
    # Remove dangerous characters
    input="${input//[^a-zA-Z0-9._-]/}"
    echo "$input"
}

# Validate before use
user_input="$(sanitize_input "$raw_input")"
```

#### Secret Handling
```bash
# Never echo secrets
# Bad
echo "Password: $PASSWORD"

# Good
log_debug "Authenticating user (password hidden)"

# Clear secrets after use
unset PASSWORD
unset API_KEY
```

### Performance Guidelines

#### Avoid Useless Cats
```bash
# Bad
cat file | grep pattern

# Good
grep pattern file

# Bad
cat file | while read line

# Good
while read line; do
    # ...
done < file
```

#### Minimize Subshells
```bash
# Bad - creates subshell
result=$(echo "$string" | tr '[:lower:]' '[:upper:]')

# Good - parameter expansion
result="${string^^}"

# Bad - multiple commands
files=$(find . -name "*.sh" | wc -l)

# Good - single command
files=$(find . -name "*.sh" -printf '.' | wc -c)
```

### Git Commit Messages

#### Format
```
type: Brief description (50 chars max)

Detailed explanation of what and why (wrap at 72 chars).
Include motivation for the change and contrast with previous
behavior.

- Bullet points for multiple changes
- Keep each point concise
- Use present tense

Fixes: #123
Related: #456
```

#### Types
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `style:` Code style changes
- `refactor:` Code refactoring
- `test:` Test additions/changes
- `chore:` Maintenance tasks

## Summary

Following these standards ensures:
- **Consistency**: Uniform code across the project
- **Maintainability**: Easy to understand and modify
- **Reliability**: Fewer bugs through best practices
- **Performance**: Efficient code execution
- **Security**: Safe handling of user input and secrets