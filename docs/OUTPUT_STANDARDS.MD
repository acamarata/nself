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

### Welcome Banner
The welcome banner should be displayed in a blue box with the nself brand name in light blue:

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

### Helper Functions

Use these helper functions for consistent output:

```bash
# Primary success message
show_success() {
    echo "${GREEN}✓${RESET} $1"
}

# Secondary success message (backup, optional actions)
show_secondary() {
    echo "${BLUE}✓${RESET} $1"
}

# Error message
show_error() {
    echo "${RED}✗${RESET} $1"
}

# Info message
show_info() {
    echo "${BLUE}ℹ${RESET} $1"
}

# Warning message
show_warning() {
    echo "${YELLOW}✱${RESET} $1"
}

# Tip message
show_tip() {
    echo "${YELLOW}⚡${RESET} $1"
}

# Section header
show_section() {
    echo ""
    echo "${CYAN}➞ $1${RESET}"
    echo ""
}

# Numbered item
show_numbered_item() {
    local number="$1"
    local command="$2"
    local description="$3"
    local detail="$4"
    
    echo "${BLUE}${number}.${RESET} ${BLUE}${command}${RESET} - ${description}"
    if [[ -n "$detail" ]]; then
        echo "   ${DIM}${detail}${RESET}"
    fi
}
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

## Commands Following These Standards

- `nself init` - ✅ Implemented
- `nself build` - To be updated
- `nself up` - To be updated
- `nself down` - To be updated
- `nself status` - To be updated
- `nself doctor` - To be updated
- `nself logs` - To be updated
- `nself prod` - To be updated
- `nself help` - To be updated

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