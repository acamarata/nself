# NSELF Architecture Documentation

## Overview

NSELF is a comprehensive CLI tool for managing self-hosted infrastructure stacks. This document outlines the target architecture for maintainability, scalability, and professional code organization.

## Current Issues

### 1. Monolithic Script Problem
- **Main script size**: 2,800+ lines in single file
- **Mixed concerns**: Business logic, UI, error handling all intertwined
- **Difficult debugging**: Hard to isolate issues
- **Testing challenges**: Cannot test components independently

### 2. Code Duplication
- Multiple definitions of same functions across files
- Inconsistent utility usage
- Repeated error handling patterns
- No centralized configuration

### 3. Complex Auto-Fix Logic
- 500+ lines of nested if-else chains in `cmd_up()`
- Silent failures without user notification
- Difficult to understand failure causes
- Unmaintainable error recovery

## Target Architecture

### Modular Design Principles

```
bin/
├── nself.sh                     # Thin wrapper (50-100 lines)
├── core/                        # Core functionality
│   ├── commands/                # Individual command implementations
│   ├── services/                # Service management
│   └── utils/                   # Shared utilities
├── shared/                      # Shared resources
│   ├── constants.sh             # Global constants
│   ├── hooks/                   # Pre/post command hooks
│   └── utils/                   # Utility functions
├── config/                      # Configuration management
│   ├── defaults.sh              # Default values
│   ├── validation.sh            # Validation logic
│   └── environment.sh           # Environment loading
└── tools/                       # Specialized tools
    ├── db/                      # Database operations
    ├── email/                   # Email configuration
    └── security/                # Security utilities
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