# nself Refactoring Roadmap

## Executive Summary

This roadmap outlines the systematic refactoring of nself from a monolithic 2,800+ line script to a modular, maintainable architecture. The refactoring will be done in phases to minimize disruption while improving code quality.

## Current State Analysis

### Critical Issues
- **Monolithic script**: 2,807 lines in single `nself.sh` file
- **Complex auto-fix logic**: 500+ lines of nested conditionals
- **Code duplication**: Same functions defined in multiple files
- **Circular dependencies**: Scripts sourcing each other
- **Inconsistent patterns**: Mixed error handling and output styles

### Technical Debt Metrics
- **Cyclomatic Complexity**: Very High (>50 in main functions)
- **Code Duplication**: ~30% repeated code
- **Test Coverage**: 0% (no automated tests)
- **Documentation**: Minimal inline documentation
- **Maintainability Index**: Low

## Refactoring Phases

### Phase 1: Foundation (Week 1)
**Goal**: Establish new structure without breaking existing functionality

#### Tasks
1. **Create Directory Structure** (Day 1)
   ```bash
   mkdir -p bin/{config,shared,commands,services,tools,templates,tests}
   mkdir -p bin/shared/{utils,hooks}
   mkdir -p bin/services/{docker,database,web,storage,monitoring}
   ```

2. **Extract Utility Functions** (Day 2-3)
   - Create `shared/utils/display.sh`
   - Create `shared/utils/error.sh`
   - Create `shared/utils/progress.sh`
   - Create `shared/utils/validation.sh`
   - Create `shared/utils/docker.sh`

3. **Implement Hook System** (Day 4)
   - Create `shared/hooks/pre-command.sh`
   - Create `shared/hooks/post-command.sh`
   - Create `shared/hooks/error-handlers.sh`
   - Create `shared/hooks/cleanup.sh`

4. **Setup Configuration Management** (Day 5)
   - Create `config/defaults.sh`
   - Create `config/validation.sh`
   - Create `config/environment.sh`

#### Deliverables
- New directory structure created
- Shared utilities extracted and tested
- Hook system implemented
- Configuration management centralized

#### Success Criteria
- All utility functions work independently
- No breaking changes to existing functionality
- Clear separation of concerns established

### Phase 2: Command Extraction (Week 2)
**Goal**: Break down monolithic script into command modules

#### Tasks
1. **Create Command Router** (Day 1)
   ```bash
   # New thin nself.sh wrapper (50-100 lines)
   main() {
       source_utilities
       route_command "$@"
   }
   ```

2. **Extract Simple Commands** (Day 2)
   - `commands/version.sh`
   - `commands/help.sh`
   - `commands/update.sh`

3. **Extract Core Commands** (Day 3-4)
   - `commands/init.sh`
   - `commands/build.sh`
   - `commands/status.sh`
   - `commands/logs.sh`

4. **Extract Complex Commands** (Day 5)
   - `commands/up.sh` (split auto-fix logic)
   - `commands/down.sh`
   - `commands/doctor.sh`

#### Deliverables
- All commands in separate modules
- Main script reduced to <100 lines
- Consistent command interface

#### Success Criteria
- Each command works independently
- Commands follow consistent patterns
- Auto-fix logic properly separated

### Phase 3: Service Layer (Week 3)
**Goal**: Create service abstraction layer

#### Tasks
1. **Docker Services** (Day 1-2)
   - `services/docker/compose.sh`
   - `services/docker/containers.sh`
   - `services/docker/health.sh`

2. **Database Services** (Day 3)
   - `services/database/postgres.sh`
   - `services/database/hasura.sh`
   - `services/database/migrations.sh`

3. **Web Services** (Day 4)
   - `services/web/nginx.sh`
   - `services/web/ssl.sh`
   - `services/web/domains.sh`

4. **Storage Services** (Day 5)
   - `services/storage/minio.sh`
   - `services/storage/backup.sh`

#### Deliverables
- Service abstraction layer complete
- Consistent service interfaces
- Health checking standardized

#### Success Criteria
- Services encapsulate complexity
- Clear service boundaries
- Improved error handling

### Phase 4: Testing Framework (Week 4)
**Goal**: Implement comprehensive testing

#### Tasks
1. **Setup Test Framework** (Day 1)
   - Choose testing framework (BATS or custom)
   - Create `tests/test-framework.sh`
   - Setup CI/CD integration

2. **Unit Tests** (Day 2-3)
   - Test utility functions
   - Test validation logic
   - Test error handling

3. **Integration Tests** (Day 4)
   - Test command workflows
   - Test service interactions
   - Test error recovery

4. **Documentation** (Day 5)
   - Update all documentation
   - Create migration guide
   - Update README

#### Deliverables
- Complete test suite
- >80% code coverage
- Updated documentation

#### Success Criteria
- All tests passing
- Critical paths covered
- Documentation complete

## Implementation Details

### Phase 1 Implementation

#### Step 1.1: Directory Creation Script
```bash
#!/bin/bash
# create-structure.sh

create_directories() {
    local base_dir="bin"
    
    # Core directories
    mkdir -p "$base_dir"/{config,shared,commands,services,tools,templates,tests,docs}
    
    # Shared subdirectories
    mkdir -p "$base_dir"/shared/{utils,hooks}
    
    # Service subdirectories
    mkdir -p "$base_dir"/services/{docker,database,web,storage,monitoring}
    
    # Tool subdirectories
    mkdir -p "$base_dir"/tools/{db,email,security,development}
    
    # Test subdirectories
    mkdir -p "$base_dir"/tests/{unit,integration,fixtures}
    
    echo "Directory structure created successfully"
}

create_directories
```

#### Step 1.2: Utility Extraction
```bash
# Extract display utilities
extract_display_utils() {
    cat > bin/shared/utils/display.sh << 'EOF'
#!/bin/bash
# display.sh - Output formatting utilities

# Color definitions
readonly COLOR_RESET='\033[0m'
readonly COLOR_RED='\033[0;31m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_YELLOW='\033[0;33m'
readonly COLOR_BLUE='\033[0;34m'

# Logging functions
log_info() { echo -e "${COLOR_BLUE}[INFO]${COLOR_RESET} $1"; }
log_success() { echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} $1"; }
log_warning() { echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $1" >&2; }
log_error() { echo -e "${COLOR_RED}[ERROR]${COLOR_RESET} $1" >&2; }
EOF
}
```

### Phase 2 Implementation

#### Step 2.1: Command Template
```bash
#!/bin/bash
# commands/template.sh - Command template

# Source dependencies
source "$(dirname "$0")/../shared/utils/display.sh"
source "$(dirname "$0")/../shared/utils/error.sh"
source "$(dirname "$0")/../config/environment.sh"

# Command metadata
COMMAND_NAME="template"
COMMAND_DESCRIPTION="Command description"
COMMAND_USAGE="nself template [options]"

# Main command function
cmd_template() {
    # Parse options
    parse_options "$@"
    
    # Validate environment
    validate_environment || exit 1
    
    # Execute command logic
    execute_template
    
    # Show results
    show_results
}

# Command implementation
execute_template() {
    log_info "Executing template command..."
    # Implementation here
}

# Export command function
cmd_template "$@"
```

### Phase 3 Implementation

#### Step 3.1: Service Template
```bash
#!/bin/bash
# services/template-service.sh - Service template

source "$(dirname "$0")/../../shared/utils/display.sh"
source "$(dirname "$0")/../../shared/utils/error.sh"

# Service functions
start_service() {
    local service_name="$1"
    log_info "Starting $service_name..."
    # Implementation
}

stop_service() {
    local service_name="$1"
    log_info "Stopping $service_name..."
    # Implementation
}

check_service_health() {
    local service_name="$1"
    # Health check implementation
}
```

## Migration Strategy

### Backward Compatibility
1. **Maintain old interface** during transition
2. **Proxy old commands** to new modules
3. **Deprecation warnings** for old usage
4. **Grace period** before removing old code

### Risk Mitigation
1. **Feature flags** for new functionality
2. **Rollback plan** for each phase
3. **Parallel testing** of old and new code
4. **Incremental deployment** to users

### Communication Plan
1. **Announce refactoring** to users
2. **Weekly progress updates**
3. **Migration guide** for each phase
4. **Support channel** for issues

## Metrics and Monitoring

### Code Quality Metrics
- **Lines of Code**: Track reduction
- **Cyclomatic Complexity**: Measure improvement
- **Code Coverage**: Increase to >80%
- **Duplication**: Reduce to <5%

### Performance Metrics
- **Startup Time**: Measure improvement
- **Command Execution**: Track speed
- **Memory Usage**: Monitor reduction
- **Error Rate**: Track decrease

### Success Metrics
- **User Satisfaction**: Survey feedback
- **Bug Reports**: Track reduction
- **Development Velocity**: Measure increase
- **Maintenance Time**: Track reduction

## Timeline Summary

| Phase | Duration | Start Date | End Date | Status |
|-------|----------|------------|----------|--------|
| Phase 1: Foundation | 1 week | Week 1 | Week 1 | Pending |
| Phase 2: Commands | 1 week | Week 2 | Week 2 | Pending |
| Phase 3: Services | 1 week | Week 3 | Week 3 | Pending |
| Phase 4: Testing | 1 week | Week 4 | Week 4 | Pending |
| Stabilization | 1 week | Week 5 | Week 5 | Pending |

## Risk Assessment

### High Risk Items
1. **Breaking existing functionality**
   - Mitigation: Comprehensive testing
   - Mitigation: Incremental changes

2. **User disruption**
   - Mitigation: Clear communication
   - Mitigation: Backward compatibility

3. **Hidden dependencies**
   - Mitigation: Thorough analysis
   - Mitigation: Gradual extraction

### Medium Risk Items
1. **Performance regression**
   - Mitigation: Performance testing
   - Mitigation: Optimization phase

2. **Documentation gaps**
   - Mitigation: Document as we go
   - Mitigation: Review cycles

## Post-Refactoring Plans

### Future Enhancements
1. **Plugin System**: Enable third-party extensions
2. **API Layer**: RESTful API for programmatic access
3. **Web Interface**: Browser-based management
4. **Metrics Dashboard**: Usage and performance metrics
5. **Auto-scaling**: Dynamic resource management

### Continuous Improvement
1. **Regular code reviews**
2. **Quarterly refactoring sprints**
3. **Performance optimization**
4. **Security audits**
5. **User feedback integration**

## Conclusion

This refactoring roadmap transforms nself from a monolithic script to a modular, maintainable, and extensible tool. The phased approach minimizes risk while delivering incremental improvements. Success depends on:

1. **Systematic execution** of each phase
2. **Comprehensive testing** at every step
3. **Clear communication** with users
4. **Commitment to quality** over speed

The investment in refactoring will pay dividends in:
- **Reduced maintenance burden**
- **Faster feature development**
- **Improved reliability**
- **Better user experience**
- **Sustainable growth**

This roadmap is a living document and should be updated as the refactoring progresses and new insights are gained.