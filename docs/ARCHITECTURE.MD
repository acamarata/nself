# nself Architecture Documentation

## Overview

nself v0.3.9 is a comprehensive CLI tool for managing self-hosted infrastructure stacks with admin UI, enterprise search, and VPS deployment capabilities. This document outlines the current architecture, design patterns, and system organization.

## Current Architecture

### Directory Structure

```
src/
├── cli/                         # Command implementations
│   ├── nself.sh                 # Main entry point and command router
│   ├── init.sh                  # Project initialization
│   ├── build.sh                 # Infrastructure building
│   ├── up.sh                    # Service startup with auto-fix
│   ├── down.sh                  # Service shutdown
│   ├── status.sh                # Service health checks
│   ├── logs.sh                  # Log viewing
│   ├── doctor.sh                # System diagnostics
│   ├── db.sh                    # Database management
│   ├── email.sh                 # Email configuration
│   ├── scaffold.sh              # Service scaffolding
│   └── ...                      # Other commands
├── lib/                         # Core libraries
│   ├── auto-fix/                # Auto-fix and validation
│   │   ├── auto-fixer-v2.sh    # Enhanced auto-fix logic
│   │   ├── config-validator-v2.sh # Configuration validation
│   │   ├── postgres-extensions.sh # PostgreSQL extension management
│   │   └── ...                  # Other auto-fix modules
│   ├── config/                  # Configuration management
│   │   ├── constants.sh         # Global constants
│   │   ├── defaults.sh          # Default values
│   │   └── smart-defaults.sh    # Intelligent default detection
│   ├── errors/                  # Error handling
│   │   ├── base.sh              # Base error definitions
│   │   ├── handlers/            # Specific error handlers
│   │   └── scanner.sh           # Error detection
│   ├── hooks/                   # Command hooks
│   │   ├── pre-command.sh       # Pre-execution hooks
│   │   └── post-command.sh      # Post-execution hooks
│   └── utils/                   # Utility functions
│       ├── display.sh           # UI and formatting
│       ├── docker.sh            # Docker operations
│       ├── env.sh               # Environment management
│       ├── header.sh            # Banner display
│       ├── output-formatter-v2.sh # Enhanced output formatting
│       └── validation.sh        # Input validation
├── services/                    # Service management
│   └── docker/                  # Docker compose generation
│       ├── compose-generate.sh  # Main compose file generator
│       └── services-generate.sh # Service-specific generation
├── templates/                   # Service templates
│   ├── .env.example             # Environment template
│   ├── hasura/                  # Hasura templates
│   └── services/                # Microservice templates
│       ├── nest/                # NestJS templates
│       ├── bullmq/              # BullMQ worker templates
│       ├── go/                  # Go service templates
│       └── py/                  # Python service templates
└── tests/                       # Test suite
    ├── nself_tests.bats         # Main test suite
    └── test_framework.sh        # Test utilities
```

### Component Responsibilities

#### Main Wrapper (`nself.sh`)
- Command routing only
- Load core configuration
- Delegate to command modules
- No business logic

#### Command Modules (`core/commands/`)
- Single responsibility per command
- Parse command-specific options
- Coordinate service calls
- Return consistent exit codes

#### Service Modules (`core/services/`)
- Docker operations
- Health checking
- Network management
- Volume management

#### Utility Modules (`shared/utils/`)
- Display formatting
- Error handling
- Progress indicators
- Validation functions

## Design Patterns

### 1. Command Pattern
Each command is a self-contained module with:
- Metadata (name, description, usage)
- Option parsing
- Main execution function
- Help generation

```bash
# Example: commands/up.sh
COMMAND_NAME="up"
COMMAND_DESCRIPTION="Start all services"
COMMAND_USAGE="nself up [options]"

cmd_up() {
    parse_options "$@"
    validate_environment
    execute_command
    handle_results
}
```

### 2. Service Layer Pattern
Services encapsulate complex operations:
- Abstract Docker complexity
- Provide consistent interfaces
- Handle retries and recovery
- Manage state transitions

```bash
# Example: services/docker/compose.sh
start_services() {
    check_prerequisites
    start_core_services
    start_dependent_services
    wait_for_healthy_state
}
```

### 3. Hook System
Pre/post execution hooks for:
- Environment validation
- Logging setup
- Cleanup operations
- Error recovery

```bash
# Pre-command validation
pre_command_validation() {
    check_initialization
    check_docker_availability
    load_environment
}

# Post-command cleanup
post_command_cleanup() {
    log_completion
    cleanup_temp_files
    show_results
}
```

## Error Handling Strategy

### Centralized Error Management
```bash
# Consistent error handling across all modules
handle_error() {
    local exit_code=$1
    local error_message=$2
    local error_type=$3
    
    log_error "$error_message"
    attempt_recovery "$error_type"
    exit "$exit_code"
}
```

### Error Recovery
- Categorized error types
- Specific recovery strategies
- User notification
- Logging for debugging

### Error Types
1. **Validation Errors**: Configuration issues
2. **Docker Errors**: Container/service failures
3. **Network Errors**: Port conflicts, connectivity
4. **Permission Errors**: File/directory access
5. **Dependency Errors**: Missing requirements

## Configuration Management

### Environment Configuration
- Centralized loading
- Validation before use
- Safe variable expansion
- Default value handling

### Configuration Layers
1. **System Defaults**: Built-in defaults
2. **User Configuration**: `.env.local`
3. **Environment Overrides**: Runtime variables
4. **Command Options**: CLI arguments

## Testing Strategy

### Unit Testing
- Test individual functions
- Mock external dependencies
- Validate error handling
- Check edge cases

### Integration Testing
- Test command workflows
- Verify service interactions
- Validate error recovery
- Check state transitions

### Test Framework
```bash
# Using BATS or custom framework
test_command_execution() {
    result=$(cmd_up --dry-run 2>&1)
    assert_equals "$?" "0"
    assert_contains "$result" "SUCCESS"
}
```

## Performance Considerations

### Optimization Points
1. **Lazy Loading**: Load modules only when needed
2. **Parallel Execution**: Start services concurrently
3. **Caching**: Cache validation results
4. **Minimal Dependencies**: Reduce external calls

### Resource Management
- Proper cleanup of temporary files
- Docker resource limits
- Memory-efficient operations
- Network connection pooling

## Security Considerations

### Security Practices
1. **Input Validation**: Sanitize all user input
2. **Secret Management**: Never log sensitive data
3. **Permission Checks**: Validate file access
4. **Secure Defaults**: Safe default configurations

### Secret Handling
```bash
# Never echo secrets
load_secret() {
    local secret_name=$1
    read -s -p "Enter $secret_name: " secret_value
    export "$secret_name=$secret_value"
}
```

## Migration Path

### Phase 1: Foundation (Week 1)
1. Create directory structure
2. Extract utility functions
3. Implement hook system
4. Create test framework

### Phase 2: Core Refactoring (Week 2)
1. Split main script
2. Create command modules
3. Implement service layer
4. Add error handling

### Phase 3: Enhancement (Week 3)
1. Add comprehensive tests
2. Implement caching
3. Optimize performance
4. Update documentation

## Best Practices

### Code Quality
- ShellCheck compliance
- Consistent naming conventions
- Proper quoting and escaping
- Error checking on all operations

### Documentation
- Inline function documentation
- Usage examples
- Error code documentation
- Migration guides

### Version Control
- Semantic versioning
- Detailed changelogs
- Feature branches
- Code review process

## Monitoring and Logging

### Logging Levels
1. **ERROR**: Critical failures
2. **WARNING**: Recoverable issues
3. **INFO**: Normal operations
4. **DEBUG**: Detailed diagnostics

### Log Management
```bash
# Structured logging
log_event() {
    local level=$1
    local message=$2
    local context=$3
    
    echo "$(date -Iseconds) [$level] $message ${context:+(context: $context)}" >> "$LOG_FILE"
}
```

## Future Enhancements

### Planned Features
1. **Plugin System**: Third-party extensions
2. **API Layer**: Programmatic access
3. **Web Interface**: Browser-based management
4. **Metrics Collection**: Usage analytics
5. **Auto-scaling**: Dynamic resource management

### Extensibility Points
- Command plugins
- Service providers
- Output formatters
- Custom validators

## Conclusion

This architecture provides:
- **Maintainability**: Clear separation of concerns
- **Testability**: Isolated, testable components
- **Scalability**: Easy to add new features
- **Reliability**: Robust error handling
- **Performance**: Optimized operations

The modular design ensures long-term sustainability while preserving all existing functionality.