# nself Output Standards

## Color Scheme

### Status Indicators
- **Primary Success**: Green checkmark (✓) for main successful operations
- **Secondary Success**: Blue checkmark (✓) for secondary/backup/optional actions
- **Error**: Red X (✗) for errors or failures  
- **Info**: Blue info symbol (ℹ) followed by normal text
- **Warning**: Yellow asterisk (✱) for warnings
- **Tips**: Yellow lightning bolt (⚡) for tips
- **Section Header**: Light blue arrow (➞) for section headers (entire header text is cyan)

### Text Colors
- **Headers/Banners**: Regular blue for boxes and main headers
- **Brand Name**: Light blue for "nself" brand name (always capitalize as "nself" unless in all-caps context)
- **Section Headers**: Entire header in light blue
- **Numbers**: Blue for numbered lists (1., 2., 3.)
- **Commands/Files**: Blue text for nself commands and file names in context
- **Descriptions**: Dimmed for secondary text

## Standard Output Format

### Command Headers (v0.3.7+)
All commands should use the standardized command header pattern:

```
╔══════════════════════════════════════════════════════════╗
║ nself command                                            ║
║ Brief description of what the command does               ║
╚══════════════════════════════════════════════════════════╝
```

Implementation:
```bash
show_command_header "nself command" "Brief description"
```

### Welcome Banner
For initialization and major setup commands, use the welcome banner:

```
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║   Welcome to nself - Modern Full-Stack Platform          ║
║   Build production-ready applications with ease          ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
```

### Status Messages
- Primary Success: `✓ Main operation completed`
- Secondary Success: `✓ Backup created` (blue checkmark)
- Error: `✗ Operation failed: reason`
- Info: `ℹ Information text`
- Warning: `✱ Warning text`
- Tip: `⚡ Run nself build immediately`

### Section Headers
Section headers use a light blue arrow with all text in cyan:
```
➞ Section Name
```

### Numbered Lists
Numbered lists use blue numbers with blue command text:
```
1. Edit .env.local to customize (optional)
   Additional details in dimmed text

2. nself build - Generate all project files
   Additional details in dimmed text

3. nself up - Start your backend
   Additional details in dimmed text
```

## Example Complete Output

Here's an example of the complete output format for `nself init`:

```
% nself init

╔══════════════════════════════════════════════════════════╗
║                                                          ║
║   Welcome to nself - Modern Full-Stack Platform          ║
║   Build production-ready applications with ease          ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝

✓ Existing env files backed up with .backup suffix
✓ Created .env.example (reference documentation)
✓ Created .env.local (your configuration file)

➞ Quick Tips

⚡ Run nself build immediately - it works with defaults!
⚡ Use nself prod to generate secure production passwords
⚡ Check .env.example for all available options
⚡ Run nself help for complete command documentation

➞ Next Steps

1. Edit .env.local to customize (optional)
   Only add what you want to change - defaults handle the rest

2. nself build - Generate all project files
   Creates Docker configs, service templates, and more

3. nself up - Start your backend
   Launches PostgreSQL, Hasura, and all configured services
```

## Implementation Guidelines

### Bash Color Variables
Define these color variables at the start of each script:

```bash
# Define colors - use if available, otherwise empty
GREEN="${COLOR_GREEN:-}"
RED="${COLOR_RED:-}"
YELLOW="${COLOR_YELLOW:-}"
BLUE="${COLOR_BLUE:-}"
CYAN="${COLOR_CYAN:-}"
RESET="${COLOR_RESET:-}"
BOLD="${COLOR_BOLD:-}"
DIM="${COLOR_DIM:-}"
```

### Standard Helper Functions (v0.3.7+)

**All commands MUST use these standard logging functions:**

```bash
# Source the display utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"

# Command header (required for all commands)
show_command_header "nself command" "Description of what this command does"

# Standard logging functions (use instead of echo)
log_success "Operation completed successfully"     # Green ✓
log_error "Something went wrong"                   # Red ✗  
log_warning "This needs attention"                 # Yellow ✱
log_info "Additional information"                  # Blue ℹ

# Section headers and formatting
show_section "Section Title"                       # Cyan ➞
```

**Command Structure Pattern:**
```bash
#!/usr/bin/env bash
# command.sh - Brief description

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"

# Help function
show_command_help() {
    echo "nself command - Description"
    echo ""
    echo "Usage: nself command [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  nself command                  # Description"
}

# Main command function
cmd_command() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_command_help
                return 0
                ;;
            *)
                log_error "Unknown option: $1"
                log_info "Use 'nself command --help' for usage information"
                return 1
                ;;
        esac
    done
    
    # Show command header
    show_command_header "nself command" "Brief description"
    
    # Command implementation here
    log_info "Starting operation..."
    # ... do work ...
    log_success "Operation completed successfully"
}

# Execute the command
cmd_command "$@"
```

## Design Principles

1. **Clarity**: Keep output clean and focused on essential information
2. **Consistency**: Use the same patterns across all commands
3. **Meaningful Color**: Colors convey information (green=good, red=bad, yellow=attention, blue=info/commands)
4. **Progressive Disclosure**: Show summary first, details on request
5. **Visual Hierarchy**: Use indentation and formatting to show relationships
6. **Actionable Messages**: Always tell users what to do next
7. **Proper Naming**: Always use "nself" with capital N, lowercase s (unless in all-caps context)
8. **Command Highlighting**: Use blue text for nself commands and file names, not bold

## Important Rules

- **nself Naming**: Always capitalize as "nself" in normal text, "nself" in all-caps contexts
- **Command Highlighting**: Use blue text instead of bold for commands
- **File Highlighting**: Use blue text for file names in context
- **Numbered Lists**: Use regular numbers (1., 2., 3.) in blue
- **Section Headers**: Entire header text should be cyan, not just the arrow

## Commands Following These Standards (v0.3.7)

### ✅ Fully Compliant Commands
- `nself init` - Header + Help + Standard logging
- `nself build` - Header + Help + Standard logging  
- `nself start` - Header + Help + Standard logging
- `nself stop` - Header + Help + Standard logging
- `nself status` - Header + Help + Standard logging
- `nself doctor` - Header + Help + Standard logging
- `nself logs` - Header + Help + Standard logging
- `nself prod` - Header + Help + Standard logging
- `nself urls` - Header + Help + Standard logging
- `nself update` - Header + Help + Standard logging
- `nself trust` - Header + Help + Standard logging
- `nself reset` - Header + Help + Standard logging
- `nself restart` - Header + Help + Standard logging
- `nself db` - Header + Help + Standard logging
- `nself scaffold` - Header + Help + Standard logging
- `nself email` - Header + Help + Standard logging
- `nself ssl` - Header + Help + Standard logging

### ✅ Simple Commands (Appropriate Level)
- `nself version` - Minimal output appropriate
- `nself help` - Special case (displays help only)
- `nself up` - Alias to start (inherits compliance)
- `nself down` - Alias to stop (inherits compliance)

### 📊 Compliance Status: 79% (19/24 commands)
Most commands follow the v0.3.7 output standards. Remaining items:
- 5 commands need show_command_header implementation
- Some commands need echo-to-log conversion for full compliance

## Technical Notes

- Colors degrade gracefully in non-TTY environments
- Line width should stay under 80 characters when possible
- Unicode characters (✓, ✗, ℹ, ✱, ⚡, ➞) have ASCII fallbacks
- Output is tested across different terminal emulators

## ASCII Fallbacks

For environments that don't support Unicode:
- ✓ → [OK]
- ✗ → [ERROR]
- ℹ → [INFO]
- ✱ → [WARNING]
- ⚡ → [TIP]
- ➞ → >>