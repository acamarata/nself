# NSELF Directory Structure Guide

## Overview

This document defines the standard directory structure for the NSELF tool, organizing code for maintainability, scalability, and clarity.

## Root Directory Structure

```
nself/
├── README.md                    # Project overview and quick start
├── CHANGELOG.md                 # Version history
├── LICENSE                      # License information  
├── install.sh                   # Installation script
├── .gitignore                   # Git ignore patterns
├── .claude/                     # AI assistant workspace (gitignored)
└── bin/                         # Main tool directory
```

## Bin Directory Organization

### Core Structure

```
bin/
├── nself.sh                     # Main entry point (thin wrapper)
├── VERSION                      # Current version number
├── docs/                        # Documentation
├── config/                      # Configuration management
├── shared/                      # Shared utilities and resources
├── commands/                    # Command implementations
├── services/                    # Service management modules
├── tools/                       # Specialized tools
├── templates/                   # Template files
├── certs/                       # SSL certificates
├── logs/                        # Log files (gitignored)
└── tests/                       # Test suites
```

### Detailed Structure

#### `/bin/config/` - Configuration Management
```
config/
├── defaults.sh                  # Default configuration values
├── validation.sh                # Configuration validation
└── environment.sh               # Environment loading and management
```

**Purpose**: Centralize all configuration logic
- Load and validate environment files
- Provide default values
- Validate user configuration
- Manage environment variables

#### `/bin/shared/` - Shared Resources
```
shared/
├── constants.sh                 # Global constants
├── utils/                       # Utility functions
│   ├── display.sh               # Output formatting and colors
│   ├── error.sh                 # Error handling
│   ├── progress.sh              # Progress indicators
│   ├── validation.sh            # Input validation
│   ├── file.sh                  # File operations
│   ├── network.sh               # Network utilities
│   └── docker.sh                # Docker helpers
└── hooks/                       # Hook system
    ├── pre-command.sh           # Pre-execution hooks
    ├── post-command.sh          # Post-execution hooks
    ├── error-handlers.sh        # Error recovery hooks
    └── cleanup.sh               # Cleanup hooks
```

**Purpose**: Provide reusable functionality across all modules
- Consistent output formatting
- Centralized error handling
- Common validation functions
- Shared Docker operations

#### `/bin/commands/` - Command Modules
```
commands/
├── init.sh                      # Initialize project
├── build.sh                     # Build project structure
├── up.sh                        # Start services
├── down.sh                      # Stop services
├── restart.sh                   # Restart services
├── status.sh                    # Show service status
├── logs.sh                      # View logs
├── doctor.sh                    # System diagnostics
├── update.sh                    # Self-update
├── version.sh                   # Show version
├── help.sh                      # Show help
├── diff.sh                      # Configuration diff
├── reset.sh                     # Reset project
├── trust.sh                     # SSL certificate trust
├── fix-env.sh                   # Fix environment files
└── prod.sh                      # Production setup
```

**Purpose**: Implement individual CLI commands
- One file per command
- Self-contained functionality
- Consistent interface
- Clear dependencies

#### `/bin/services/` - Service Management
```
services/
├── docker/                      # Docker services
│   ├── compose.sh               # Docker Compose operations
│   ├── containers.sh            # Container management
│   ├── networks.sh              # Network management
│   ├── volumes.sh               # Volume management
│   └── health.sh                # Health checking
├── database/                    # Database services
│   ├── postgres.sh              # PostgreSQL management
│   ├── hasura.sh                # Hasura GraphQL
│   ├── redis.sh                 # Redis cache
│   └── migrations.sh            # Database migrations
├── web/                         # Web services
│   ├── nginx.sh                 # Nginx configuration
│   ├── ssl.sh                   # SSL/TLS management
│   └── domains.sh               # Domain routing
├── storage/                     # Storage services
│   ├── minio.sh                 # MinIO S3 storage
│   └── backup.sh                # Backup operations
└── monitoring/                  # Monitoring services
    ├── metrics.sh               # Metrics collection
    └── alerts.sh                # Alert management
```

**Purpose**: Manage infrastructure services
- Service-specific logic
- Health monitoring
- Resource management
- Service orchestration

#### `/bin/tools/` - Specialized Tools
```
tools/
├── db/                          # Database tools
│   ├── schema.sh                # Schema management
│   ├── seed.sh                  # Data seeding
│   ├── backup.sh                # Database backup
│   ├── restore.sh               # Database restore
│   ├── sync.sh                  # Schema sync
│   └── migrate.sh               # Migration tools
├── email/                       # Email tools
│   ├── providers.sh             # Provider setup
│   ├── templates.sh             # Email templates
│   ├── testing.sh               # Email testing
│   └── config/                  # Provider configs
│       ├── sendgrid.sh
│       ├── aws-ses.sh
│       ├── mailgun.sh
│       └── postmark.sh
├── security/                    # Security tools
│   ├── certificates.sh          # Certificate management
│   ├── secrets.sh               # Secret generation
│   ├── encryption.sh            # Encryption utilities
│   └── mkcert                   # mkcert binary
└── development/                 # Development tools
    ├── hot-reload.sh            # Hot reload
    ├── debugging.sh             # Debug utilities
    └── testing.sh               # Test utilities
```

**Purpose**: Provide specialized functionality
- Database operations
- Email configuration
- Security management
- Development helpers

#### `/bin/templates/` - Template Files
```
templates/
├── config-server/               # Configuration server
├── hasura/                      # Hasura templates
│   ├── schema.sql.template
│   ├── actions.yaml.template
│   └── triggers.yaml.template
├── services/                    # Service templates
│   ├── nest/                    # NestJS templates
│   ├── bullmq/                  # BullMQ templates
│   ├── py/                      # Python templates
│   ├── go/                      # Go templates
│   └── shared/                  # Shared templates
├── docker/                      # Docker templates
│   ├── docker-compose.yml.template
│   ├── Dockerfile.template
│   └── .dockerignore.template
├── nginx/                       # Nginx templates
│   ├── nginx.conf.template
│   └── sites-enabled.template
├── ssl/                         # SSL templates
│   ├── openssl.conf.template
│   └── ssl-params.template
└── health/                      # Health check templates
    └── health-endpoint.js.template
```

**Purpose**: Store template files for generation
- Service templates
- Configuration templates
- Docker templates
- Infrastructure templates

#### `/bin/certs/` - Certificate Storage
```
certs/
├── local/                       # Local development
├── staging/                     # Staging environment
└── production/                  # Production environment
```

**Purpose**: Store SSL certificates
- Environment-specific certificates
- Self-signed certificates for development
- Production certificates

#### `/bin/logs/` - Log Files (gitignored)
```
logs/
├── nself.log                    # Main tool log
├── errors.log                   # Error log
├── access.log                   # Access log
└── debug.log                    # Debug log
```

**Purpose**: Store application logs
- Execution logs
- Error tracking
- Debug information
- Access logs

#### `/bin/tests/` - Test Suites (CRITICAL FOR MODULAR DEVELOPMENT)
```
tests/
├── unit/                        # Unit tests - TEST INDIVIDUAL FUNCTIONS
│   ├── commands/                # Command function tests
│   │   ├── test_init.sh        # Test init command functions
│   │   ├── test_up.sh          # Test up command functions
│   │   └── test_build.sh       # Test build command functions
│   ├── services/                # Service function tests
│   │   ├── test_postgres.sh    # Test PostgreSQL functions
│   │   ├── test_hasura.sh      # Test Hasura functions
│   │   └── test_docker.sh      # Test Docker helper functions
│   └── utils/                   # Utility function tests
│       ├── test_display.sh     # Test display/output functions
│       ├── test_validation.sh  # Test validation functions
│       └── test_error.sh       # Test error handling
├── integration/                 # Integration tests - TEST WORKFLOWS
│   ├── test_service_startup.sh # Test complete service startup
│   ├── test_error_recovery.sh  # Test error recovery flows
│   └── test_workflows.sh       # Test command sequences
├── sanity/                      # Sanity tests - QUICK VALIDATION
│   ├── test_prerequisites.sh   # Test system requirements
│   ├── test_environment.sh     # Test environment setup
│   └── test_dependencies.sh    # Test external dependencies
├── fixtures/                    # Test data and mocks
│   ├── mock_data/              # Mock data files
│   ├── test_configs/           # Test configuration files
│   └── stubs/                  # Function stubs for testing
├── run_tests.sh                 # Main test runner script
└── test_framework.sh            # Testing utilities and assertions
```

**Purpose**: CRITICAL for modular development and reliability
- **Unit Tests**: Test each function in isolation - if it passes, we can rely on it
- **Integration Tests**: Test component interactions and workflows
- **Sanity Tests**: Quick checks before full test suite
- **Test Framework**: Consistent testing utilities across all tests
- **Fixtures**: Reproducible test data and environments

**Testing Philosophy**:
- Write tests BEFORE or WITH new functions
- A passing test means we can trust that function
- When debugging, only need to fix failing tests
- As codebase grows, tests ensure nothing breaks
- Tests serve as documentation for function usage

## File Naming Conventions

### Script Files
- **Commands**: `command-name.sh` (lowercase with hyphens)
- **Modules**: `module-name.sh` (lowercase with hyphens)
- **Tests**: `test-module-name.sh` (prefix with test-)
- **Libraries**: `lib-name.sh` (prefix with lib-)

### Template Files
- **Templates**: `name.template` or `name.type.template`
- **SQL**: `schema.sql.template`
- **YAML**: `config.yaml.template`
- **JSON**: `package.json.template`

### Configuration Files
- **Environment**: `.env.example`, `.env.local`
- **Docker**: `docker-compose.yml`, `Dockerfile`
- **Config**: `config.yaml`, `settings.json`

## Module Dependencies

### Dependency Rules
1. Commands depend on services and shared utilities
2. Services depend on shared utilities only
3. Shared utilities have no dependencies
4. Tools can depend on services and utilities
5. No circular dependencies allowed
6. **TESTS ARE INDEPENDENT** - Tests should not depend on other tests
7. **TEST FIRST** - Functions must have tests before being used by other modules

### Import Pattern
```bash
# In command files
source "$(dirname "$0")/../shared/utils/display.sh"
source "$(dirname "$0")/../shared/utils/error.sh"
source "$(dirname "$0")/../services/docker/compose.sh"

# In service files
source "$(dirname "$0")/../../shared/utils/display.sh"
source "$(dirname "$0")/../../shared/utils/docker.sh"

# In tool files
source "$(dirname "$0")/../../shared/utils/display.sh"
source "$(dirname "$0")/../../services/database/postgres.sh"
```

## Directory Creation

### Required Directories
Directories that must exist for proper operation:
```bash
# Created during initialization
mkdir -p bin/{config,shared,commands,services,tools,templates,certs,logs,tests}
mkdir -p bin/shared/{utils,hooks}
mkdir -p bin/services/{docker,database,web,storage,monitoring}
mkdir -p bin/tools/{db,email,security,development}
mkdir -p bin/tests/{unit,integration,fixtures}
```

### Generated Directories
Directories created during runtime:
```bash
# Created as needed
data/                            # Application data
volumes/                         # Docker volumes
backups/                         # Backup files
tmp/                             # Temporary files
```

## Environment-Specific Structure

### Development
```
.env.local                       # Development configuration
docker-compose.override.yml      # Development overrides
seeds/development/               # Development data
```

### Staging
```
.env.staging                     # Staging configuration
docker-compose.staging.yml       # Staging compose file
seeds/staging/                   # Staging data
```

### Production
```
.env.production                  # Production configuration
docker-compose.production.yml    # Production compose file
seeds/production/                # Production data
```

## Best Practices

### Organization Rules
1. **Single Responsibility**: Each file has one clear purpose
2. **Logical Grouping**: Related files in same directory
3. **Clear Naming**: Descriptive, consistent names
4. **No Deep Nesting**: Maximum 3 levels deep
5. **Documentation**: README in complex directories

### Maintenance Guidelines
1. **Regular Cleanup**: Remove unused files
2. **Version Control**: Track important changes
3. **Backup Strategy**: Regular backups of data
4. **Log Rotation**: Prevent log files from growing too large
5. **Security**: Proper permissions on sensitive files

### Extension Points
When adding new functionality:
1. **New Command**: Add to `/bin/commands/`
2. **New Service**: Add to appropriate `/bin/services/` subdirectory
3. **New Tool**: Add to `/bin/tools/`
4. **New Utility**: Add to `/bin/shared/utils/`
5. **New Template**: Add to `/bin/templates/`

## Migration from Current Structure

### Phase 1: Create New Structure
```bash
# Create all required directories
./bin/scripts/migrate-structure.sh create

# Verify structure
./bin/scripts/migrate-structure.sh verify
```

### Phase 2: Move Existing Files
```bash
# Move files to new locations
./bin/scripts/migrate-structure.sh move

# Update import paths
./bin/scripts/migrate-structure.sh update-imports
```

### Phase 3: Test and Validate
```bash
# Run tests
./bin/scripts/migrate-structure.sh test

# Validate functionality
./bin/scripts/migrate-structure.sh validate
```

## Summary

This directory structure provides:
- **Clear Organization**: Logical file placement
- **Maintainability**: Easy to find and modify code
- **Scalability**: Easy to add new features
- **Testability**: Organized test structure
- **Documentation**: Clear purpose for each directory

Following this structure ensures consistent, professional code organization that scales with project growth.