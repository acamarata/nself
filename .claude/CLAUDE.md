# Claude Assistant Instructions for nself Project

## Project Overview

**nself** is a comprehensive self-hosted backend stack CLI that provides all the features of Nhost.io, Supabase, and other BaaS platforms, but fully self-hosted using Docker Compose.

**Current Version**: 0.3.9
**Release Date**: September 5, 2024 (Patch Updates)
**Repository**: https://github.com/acamarata/nself
**License**: Source Available (free for personal use, paid for commercial)
**Documentation**: /docs/
**Release Notes**: /docs/RELEASES.md

## Core Architecture

### Technology Stack
- **Container Orchestration**: Docker Compose v2
- **Database**: PostgreSQL 16 Alpine with 60+ extensions
- **GraphQL API**: Hasura v2.44.0
- **Authentication**: Nhost Auth service v0.36.0 (JWT-based)
- **Storage**: MinIO (S3-compatible) + Hasura Storage v0.6.1
- **Email**: MailPit (dev) / 16+ production providers
- **SSL**: mkcert for local, Let's Encrypt for production
- **Reverse Proxy**: Nginx Alpine with automatic SSL
- **Admin UI**: nself-admin v0.0.3 (React-based monitoring)
- **Cache**: Redis (optional)
- **Monitoring**: Prometheus, Grafana, Loki (optional)

### Project Structure
```
nself/
├── bin/               # Entry point scripts
│   └── nself         # Main CLI executable
├── src/
│   ├── VERSION       # Version file (0.3.9)
│   ├── cli/          # Command implementations (35 commands)
│   ├── lib/          # Shared libraries
│   │   ├── auto-fix/ # Auto-fix and validation
│   │   ├── config/   # Configuration management
│   │   ├── hooks/    # Pre/post command hooks
│   │   └── utils/    # Utilities (display, docker, env)
│   ├── services/     # Service configurations
│   │   └── docker/   # Docker compose generation
│   ├── templates/    # Docker and config templates
│   └── tools/        # Additional tools
├── docs/             # User documentation
├── tests/            # Test suites
├── .claude/          # AI assistant instructions
└── ROADMAP.md        # Project roadmap
```

## Recent Changes (v0.3.9 - Patch Updates)

### Critical Bug Fixes Applied (September 5, 2024)
1. ✅ **Build Command Hanging** - Fixed infinite loop in service generation
   - Added timeout protections for service generator scripts
   - Fixed .env.local sourcing error in compose-generate.sh
   - Resolved recursive environment loading issues
2. ✅ **Missing generate_password Function** - Added to smart-defaults.sh
3. ✅ **Docker Compose Generation** - Fixed hanging and timeout issues
4. ✅ **Service Generation** - Restored functionality with proper error handling

### Previous Bug Fixes (September 3, 2024)
1. ✅ **Status Command** - Fixed hanging due to log_debug loops in env.sh
2. ✅ **Stop Command** - Fixed compose wrapper usage (line 136)
3. ✅ **Exec Command** - Fixed container detection with proper imports
4. ✅ **Email Command** - Implemented SMTP testing with swaks
5. ✅ **Doctor Command** - Fixed function name references (error_found → issue_found)
6. ✅ **Display Library** - Added missing show_header function aliases

### Known Issues
- **Auth Health Check**: Reports unhealthy (port 4001 vs 4000) but service works correctly

## Available Commands (v0.3.9)

### Core Commands (35 total)
- `nself init` - Initialize a new project with .env
- `nself build` - Build Docker images and generate configs
- `nself start` - Start all enabled services
- `nself stop` - Stop all services (fixed in v0.3.9)
- `nself restart [service]` - Restart all or specific service
- `nself status` - Show service status (fixed in v0.3.9)
- `nself logs [service]` - View service logs
- `nself clean` - Clean up containers and volumes

### Database Commands
- `nself db` - Database management menu
- `nself backup` - Comprehensive backup system (create, restore, list, verify, schedule)
- `nself reset` - Reset database to clean state

### Configuration Commands
- `nself config` - View current configuration
- `nself prod` - Generate production configuration
- `nself email` - Configure email service
- `nself ssl` - Manage SSL certificates
- `nself trust` - Trust SSL certificates locally
- `nself validate` - Validate configuration

### Admin & Management
- `nself admin` - Admin UI management (NEW in v0.3.9)
  - `enable` - Enable admin UI with temp password
  - `disable` - Disable admin UI
  - `status` - Show admin status
  - `password` - Set admin password
  - `reset` - Reset to defaults
  - `logs` - View admin logs
  - `open` - Open in browser
- `nself exec` - Execute commands in containers (fixed in v0.3.9)
- `nself scale` - Scale services
- `nself deploy` - Deploy to remote servers
- `nself rollback` - Rollback changes
- `nself update` - Update nself CLI

### Monitoring & Diagnostics
- `nself doctor` - System diagnostics and fixes
- `nself monitor` - Monitor services
- `nself metrics` - View metrics
- `nself urls` - Show service URLs

### Development Commands
- `nself diff` - Show configuration differences
- `nself scaffold` - Generate code scaffolding
- `nself search` - Search codebase

## Key Features

### 1. Smart Defaults & Auto-Fix
- Automatically detects and fixes configuration issues
- Port conflict resolution
- Missing file generation
- Service dependency management
- Intelligent error recovery

### 2. Multi-Environment Support

#### Environment Loading Cascade
Files are loaded in this specific order (later overrides earlier):
1. `.env.dev` - Team development defaults (always loaded first)
2. `.env.staging` - Staging overrides (only if ENV=staging)
3. `.env.prod` - Production config (only if ENV=prod)
4. `.env.secrets` - Sensitive production data (only if ENV=prod)
5. `.env` - Local overrides (HIGHEST PRIORITY, always loaded last)

#### Key Philosophy
- `.env` is ALWAYS the final override file
- Users can work with just a single `.env` file if desired
- Smart defaults ensure everything works without configuration
- ENV variable in `.env.dev` determines which cascade path to follow

### 3. Admin UI (NEW in v0.3.9)
- Web-based administration at localhost:3100
- Real-time service monitoring
- Docker container management
- Database query interface
- Log viewer
- Health checks
- Backup management

### 4. SSL Management
- Automatic certificate generation with mkcert
- Wildcard certificates for *.local.nself.org
- Browser trust integration
- Production SSL with Let's Encrypt

### 5. Backup System
- Local and S3 backup support
- Scheduled backups with cron
- Point-in-time recovery
- Backup verification and pruning
- GFS retention policies

### 6. Email Configuration
- 16+ email providers supported
- Zero-config development with MailPit
- Interactive setup wizard
- Template management

## Recent Fixes (v0.3.9)

### Fixed Issues
1. **Status Command** - Fixed env.sh syntax errors with log_debug functions
2. **Stop Command** - Fixed compose wrapper function calls
3. **Exec Command** - Fixed by adding docker.sh import and environment loading
4. **Admin Integration** - Full integration with nself-admin Docker image
5. **SSL Generation** - Fixed nginx startup issues with missing certificates
6. **Container Name Resolution** - Fixed dynamic container naming issues

### Known Issues
- Auth service health check reports unhealthy (but service works on port 4001)
- Some commands may need PROJECT_NAME environment variable set

## Configuration Files

### Primary Configuration: .env
```bash
# Core settings
BASE_DOMAIN=local.nself.org
PROJECT_NAME=my-project
ENV=dev

# Service toggles
POSTGRES_ENABLED=true
HASURA_ENABLED=true
AUTH_ENABLED=true
STORAGE_ENABLED=true
REDIS_ENABLED=false
FUNCTIONS_ENABLED=false
NSELF_ADMIN_ENABLED=true

# Admin UI (NEW)
NSELF_ADMIN_PORT=3100
NSELF_ADMIN_AUTH_PROVIDER=basic
ADMIN_SECRET_KEY=<generated>
ADMIN_PASSWORD_HASH=<generated>

# Microservices
SERVICES_ENABLED=false
NESTJS_SERVICES=api,workers
GOLANG_SERVICES=analytics
PYTHON_SERVICES=ml,data

# Monitoring
PROMETHEUS_ENABLED=false
GRAFANA_ENABLED=false
LOKI_ENABLED=false
```

### Generated Files
- `docker-compose.yml` - Main services
- `docker-compose.override.yml` - Development overrides
- `nginx/default.conf` - Reverse proxy configuration
- `ssl/certificates/` - SSL certificates
- Various service-specific configs

## Development Guidelines

### When Adding New Features
1. Follow the modular structure in src/cli/
2. Update help text in src/cli/help.sh
3. Add auto-fix rules if applicable
4. Include smart defaults
5. Update documentation in docs/COMMANDS.md
6. Add tests if significant

### Command Implementation Pattern
```bash
#!/usr/bin/env bash
# src/cli/command.sh

cmd_command() {
    local arg="${1:-}"
    
    # Show header
    show_command_header "Command" "Description"
    
    # Load environment
    if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
        load_env_with_priority
    fi
    
    # Validate environment
    ensure_project_context
    check_docker_running
    
    # Execute command logic
    # ...
    
    # Show success
    show_success "Command completed"
}
```

### Important Functions to Use
```bash
# From utils/docker.sh
compose()           # Docker compose wrapper with project support
is_service_running() # Check if service is running
wait_service_healthy() # Wait for service health

# From utils/env.sh
load_env_with_priority() # Load environment files in correct order
get_env_var()           # Get environment variable with default

# From utils/display.sh
show_command_header()   # Show consistent command header
log_info()             # Info message
log_success()          # Success message
log_error()            # Error message
log_warning()          # Warning message
```

### Auto-Fix Pattern
```bash
# src/lib/auto-fix/rules.sh
fix_issue() {
    if [[ condition ]]; then
        log_warning "Issue detected"
        if [[ "$AUTO_FIX" == "true" ]]; then
            # Apply fix
            log_info "Auto-fixing..."
            # fix logic
            log_success "Fixed"
        else
            log_error "Manual fix required"
            return 1
        fi
    fi
}
```

## Testing

### Test Structure
```
tests/
├── unit/           # Unit tests
├── integration/    # Integration tests
├── bats/          # Bats test files
└── fixtures/      # Test data
```

### Running Tests
```bash
# Run all tests
./tests/run-all.sh

# Run specific test
bats tests/bats/init.bats

# Run with CI
.github/workflows/ci.yml
```

## Important Notes

### Security Considerations
- Never commit .env or .env.secrets
- Always generate strong passwords for production
- Use separate configs for dev/staging/prod
- Enable 2FA for admin access
- Rotate secrets regularly
- Admin UI runs on separate port (3100) by default

### Performance Tips
- Start only needed services
- Use resource limits in production
- Enable caching where appropriate
- Monitor resource usage with `nself status`
- Use volume mounts efficiently

### Common Issues & Solutions
1. **Port conflicts**: Auto-fix will reassign ports
2. **Docker not running**: Doctor command provides install instructions
3. **SSL errors**: Run `nself ssl` then `nself trust` to fix certificates
4. **Service failures**: Check logs with `nself logs [service]`
5. **Config issues**: Validate command shows all problems
6. **Admin UI not accessible**: Check NSELF_ADMIN_ENABLED=true and port 3100
7. **Auth unhealthy**: Known issue - service runs on 4001, health check expects 4000

## Upcoming Features (See ROADMAP.md)

### v0.4.0: Production Ready
- Kubernetes deployment options
- Multi-node clustering
- Advanced monitoring with alerts
- Automated backup verification

### v0.5.0: Enterprise Features
- Multi-tenant support
- LDAP/SAML integration
- Audit logging
- Compliance tools

## AI Assistant Best Practices

### When Helping Users:
1. **Always check current directory** - Ensure user is in project directory, not nself source
2. **Load environment first** - Use `load_env_with_priority` before accessing env vars
3. **Use compose wrapper** - Always use `compose` function, not `docker compose` directly
4. **Validate before suggesting** - Run validate/doctor before making changes
5. **Use auto-fix** - Leverage auto-fix capabilities rather than manual fixes
6. **Follow conventions** - Use existing patterns and structures
7. **Security first** - Never expose secrets or suggest insecure practices

### Common User Tasks:
- **New project**: `nself init` → `nself admin enable` → `nself build` → `nself start`
- **Add service**: Edit .env → `nself build` → `nself restart`
- **Production**: `nself prod` → Review .env → Deploy
- **Debugging**: `nself doctor` → `nself logs [service]` → `nself status`
- **Backup**: `nself backup create` → `nself backup list`
- **Admin UI**: `nself admin enable` → `nself admin password` → `nself admin open`

### Code Patterns to Follow:
- Use `show_command_header` for consistent output
- Check `ensure_project_context` before file operations
- Load environment with `load_env_with_priority`
- Use `log_*` functions for consistent messaging
- Follow existing error handling patterns
- Maintain backward compatibility
- Always source docker.sh when using compose commands

### Critical Files to Know:
- `/src/lib/utils/docker.sh` - Docker compose wrapper
- `/src/lib/utils/env.sh` - Environment loading (fixed in v0.3.9)
- `/src/lib/utils/display.sh` - Logging and display functions
- `/src/cli/admin.sh` - Admin UI management
- `/src/services/docker/compose-generate.sh` - Docker compose generation
- `/src/services/docker/compose-inline-append.sh` - Service append logic

## Related Documentation
- User Docs: /docs/COMMANDS.md (comprehensive reference)
- Changelog: /docs/CHANGELOG.md
- Roadmap: /ROADMAP.md
- Admin UI: Integrated via nself-admin Docker image
- Contributing: /CONTRIBUTING.md

## Support Channels
- GitHub Issues: https://github.com/acamarata/nself/issues
- Discord: [Coming Soon]
- Documentation: /docs/

---
*Last Updated: v0.3.9 - Admin UI integration, bug fixes, comprehensive command documentation, 40+ service templates*