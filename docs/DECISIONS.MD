# Architectural Decision Records

This document captures key decisions made in the nself project and the reasoning behind them.

## Table of Contents
- [Shell Script Architecture](#shell-script-architecture)
- [Docker Compose Wrapper](#docker-compose-wrapper)
- [Auto-Fix Philosophy](#auto-fix-philosophy)
- [Hooks System](#hooks-system)
- [Directory Organization](#directory-organization)
- [Output Standards](#output-standards)
- [Version Management](#version-management)
- [Safety Mechanisms](#safety-mechanisms)

## Shell Script Architecture

### Decision
Use modular shell scripts with one command per file instead of a monolithic script.

### Reasoning
- **Maintainability**: Easier to find and modify specific commands
- **Testing**: Individual commands can be tested in isolation
- **Collaboration**: Multiple developers can work on different commands
- **Loading**: Only load what's needed, improving performance

### Implementation
```bash
/bin/
├── nself.sh          # Thin dispatcher
├── init.sh           # One command
├── build.sh          # One command
└── shared/           # Shared utilities
```

## Docker Compose Wrapper

### Decision
Always use a `compose()` wrapper function instead of direct `docker compose` calls.

### Reasoning
- **Consistency**: Ensures project name and env file are always set
- **Migration**: Easy transition from v1 to v2
- **Configuration**: Centralized compose configuration
- **Debugging**: Single point to add logging/debugging

### Implementation
```bash
compose() {
    local env_file="${COMPOSE_ENV_FILE:-.env.local}"
    local project="${PROJECT_NAME:-nself}"
    
    if [[ -f "$env_file" ]]; then
        docker compose --project-name "$project" --env-file "$env_file" "$@"
    else
        docker compose --project-name "$project" "$@"
    fi
}
```

## Auto-Fix Philosophy

### Decision
Limit auto-fix to exactly 4 safe, predictable scenarios.

### Reasoning
- **Predictability**: Users know what might be auto-fixed
- **Safety**: No risky operations that could damage systems
- **Trust**: Users can rely on auto-fix not doing unexpected things
- **Opt-out**: Easy to disable if not wanted

### Approved Scenarios
1. Docker build cache issues → Clear and rebuild
2. Port conflicts → Only if owned by our project
3. Missing dependencies → Only from approved list
4. Configuration issues → Fix permissions/paths only

### Rejected Scenarios
- Network configuration changes
- System-wide package installation
- Firewall modifications
- User permission changes
- Database repairs

## Hooks System

### Decision
Every command must call pre_command and post_command hooks.

### Reasoning
- **Consistency**: All commands behave predictably
- **Validation**: Central place for prerequisite checks
- **Logging**: Automatic command logging
- **Cleanup**: Guaranteed cleanup operations
- **Metrics**: Easy to add telemetry later

### Implementation
```bash
# In every command
pre_command "commandname" || exit $?
# ... command logic ...
exit_code=$?
post_command "commandname" $exit_code
exit $exit_code
```

## Directory Organization

### Decision
Keep only command files in `/bin/`, everything else in subdirectories.

### Reasoning
- **Clarity**: Clear what files are commands vs utilities
- **Discovery**: Easy to see all available commands
- **Organization**: Related files grouped together
- **PATH**: Only commands need to be in PATH

### Structure
```
/bin/
├── *.sh              # Commands only
├── shared/           # All shared code
├── services/         # Service-related
├── templates/        # Templates
└── tools/           # Additional tools
```

## Output Standards

### Decision
Use structured output functions, never raw echo for user output.

### Reasoning
- **Consistency**: All output looks the same
- **Accessibility**: Respects NO_COLOR environment variable
- **Routing**: Errors automatically go to stderr
- **Formatting**: Easy to change output format globally
- **Testing**: Can capture/suppress output in tests

### Implementation
```bash
log_info "Information"      # Blue [INFO]
log_success "Success"       # Green [SUCCESS]
log_warning "Warning"       # Yellow [WARNING]
log_error "Error"          # Red [ERROR] to stderr
log_debug "Debug"          # Only if DEBUG=true
```

## Version Management

### Decision
Use `/bin/VERSION` file as source of truth, with git tags for releases.

### Reasoning
- **Simplicity**: Plain text file, easy to read
- **Availability**: Works without git (in archives)
- **Compatibility**: Standard practice for shell projects
- **Flexibility**: Can be updated by scripts

### Implementation
- Primary: `/bin/VERSION`
- Backup: Hardcoded fallback
- Releases: Git tags (`v0.3.0`)
- Format: Semantic versioning

## Safety Mechanisms

### Decision
Prevent nself commands from running in the nself repository.

### Reasoning
- **Protection**: Prevents accidental repository pollution
- **Clarity**: Repository is for source, not projects
- **Testing**: Forces proper testing practices
- **Cleanliness**: Keeps repository root clean

### Implementation
```bash
# Check for nself repository markers
if [[ -f "bin/nself.sh" ]] && [[ -d "bin/shared" ]]; then
    log_error "Cannot run in nself repository!"
    return 1
fi
```

## Environment Safety

### Decision
Load environment files safely without executing embedded commands.

### Reasoning
- **Security**: Prevents command injection
- **Reliability**: Predictable environment loading
- **Validation**: Can validate before loading
- **Debugging**: Clear error messages

### Implementation
```bash
load_env_safe() {
    # Validate file exists
    # Check for suspicious content
    # Source with restrictions
    # Validate required variables
}
```

## Logging Strategy

### Decision
All commands log to `logs/nself.log` with timestamps.

### Reasoning
- **Debugging**: Historical record of commands
- **Audit**: Track what happened when
- **Support**: Users can share logs for help
- **Metrics**: Can analyze usage patterns

### Format
```
[2025-08-10 12:34:56] nself up --detach
[2025-08-10 12:34:57] SUCCESS: Services started
```

## Error Handling

### Decision
Use consistent exit codes across all commands.

### Reasoning
- **Automation**: Scripts can react to specific errors
- **Debugging**: Clear what type of error occurred
- **Standards**: Follows Unix conventions
- **Documentation**: Easy to document

### Exit Codes
- 0: Success
- 1: General error
- 2: Missing prerequisites
- 3: Configuration error
- 4: Docker error
- 5: Network error
- 127: Command not found

## Repository Cleanliness

### Decision
Move documentation to `/docs/`, keep root minimal.

### Reasoning
- **Focus**: Root shows only essential files
- **Organization**: Documentation grouped together
- **GitHub**: Respects GitHub conventions
- **Professionalism**: Clean root looks professional

### Root Contents
- README.md (required by GitHub)
- LICENSE (required by GitHub)
- install.sh (entry point)
- /bin/ (all code)
- /docs/ (all documentation)
- /.github/ (GitHub configuration)

## Template Strategy

### Decision
Use simple bash string templates with placeholder replacement.

### Reasoning
- **Simplicity**: No external dependencies
- **Control**: Full control over generation
- **Speed**: Fast template processing
- **Debugging**: Easy to see what's generated

### Implementation
```bash
# Template with {{PLACEHOLDER}}
sed "s/{{PROJECT_NAME}}/$PROJECT_NAME/g" template > output
```

## Testing Philosophy

### Decision
Focus on manual testing with clear test procedures.

### Reasoning
- **Practicality**: Shell scripts are hard to unit test
- **Reality**: Most issues are integration issues
- **Speed**: Manual testing is often faster for shell scripts
- **Documentation**: Test procedures serve as documentation

### Approach
1. Clear test directories
2. Documented test procedures
3. End-to-end testing focus
4. Real Docker/system testing

## Future Considerations

### Potential Changes
- **Compiled binary**: Bundle bash for better distribution
- **Homebrew formula**: Easier installation on macOS
- **GitHub Actions**: Automated testing
- **Telemetry**: Optional anonymous usage stats

### Rejected Ideas
- **Python rewrite**: Would lose simplicity and portability
- **Node.js rewrite**: Would require Node installation
- **Go rewrite**: Would need compilation per platform
- **Complex templating**: Adds dependencies and complexity

## Conclusion

These decisions prioritize:
1. **Simplicity** over complexity
2. **Safety** over features
3. **Consistency** over flexibility
4. **Maintainability** over cleverness

The goal is a reliable, predictable tool that does one thing well: manage self-hosted infrastructure.