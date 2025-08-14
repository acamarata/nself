# Migration Guide: v0.2.x to v0.3.0

## Overview

Version 0.3.0 introduces a major architectural refactor, moving from a monolithic structure to a clean, modular src-centric design. While this is a breaking change, the upgrade process is automated.

## Key Changes

### Directory Structure
- **Before**: Implementation code mixed in `/bin` directory
- **After**: Clean separation with `/src` for implementation, `/bin` for thin shims

### New Organization
```
/bin/           # Only thin delegation scripts
/src/           # All implementation
  ├── cli/      # Command implementations
  ├── lib/      # Shared libraries and utilities
  ├── services/ # Service management
  ├── templates/# All templates
  ├── tools/    # Development tools
  └── tests/    # Test suite
```

## Upgrade Process

### Automatic Upgrade

For existing installations, simply run:
```bash
nself update
```

This will:
1. Download v0.3.0
2. Migrate your installation to the new structure
3. Preserve your existing configurations

### Manual Upgrade

If you prefer manual control:
```bash
# Backup current installation
cp -r ~/.nself ~/.nself.backup

# Download and install v0.3.0
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/v0.3.0/install.sh | bash
```

## Project Updates

After upgrading nself itself, update your projects:

### 1. Rebuild Configuration
```bash
cd your-project
nself build
```

### 2. Verify Environment
Check your `.env.local` for any custom port configurations:
```bash
grep -E "NGINX_HTTP_PORT|NGINX_HTTPS_PORT|POSTGRES_PORT" .env.local
```

### 3. Start Services
```bash
nself up
```

## Go Services

If you have custom Go services:

### Ensure go.mod Exists
For each Go service without a `go.mod`:
```bash
cd services/go/your-service
go mod init your-service
go mod tidy
```

### Dockerfile Updates
The new Dockerfiles automatically run `go mod tidy`. If you have custom Dockerfiles, ensure they include:
```dockerfile
RUN go mod tidy && go mod download
ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o main .
```

## macOS Users

No special action required. v0.3.0 includes full compatibility with macOS bash 3.2:
- Associative array fallbacks implemented
- All scripts use portable bash features
- Tested on macOS 10.14+

## New Features

Take advantage of new capabilities:

### Enhanced Error Recovery
```bash
# Comprehensive system check
nself doctor

# Real-time status monitoring
nself status

# Advanced log filtering
nself logs --filter error --tail 50
```

### Auto-Fix System
Port conflicts and build errors now offer automatic fixes:
- Port conflicts: Automatic alternative port configuration
- Go module errors: Automatic dependency resolution
- Docker build failures: Intelligent recovery options

## Breaking Changes

### Environment Variables
- `HASURA_ADMIN_SECRET` → `HASURA_GRAPHQL_ADMIN_SECRET` (auto-migrated)

### Command Locations
- Commands remain the same, but internal paths have changed
- Existing scripts calling nself commands will continue to work

## Troubleshooting

### Installation Issues
```bash
# Check version
nself version  # Should show 0.3.0

# Run doctor for diagnostics
nself doctor

# Check installation
ls -la ~/.nself/src/  # Should show new structure
```

### Port Conflicts
The new version handles port conflicts automatically:
1. Detects conflicts during `nself up`
2. Offers interactive options
3. Automatically rebuilds after port changes

### Build Errors
Enhanced error detection and recovery:
- Go modules: Automatic `go mod tidy`
- Node.js: Automatic dependency cleanup
- Docker: Intelligent cache management

## Rollback

If needed, rollback to previous version:
```bash
# Restore backup
rm -rf ~/.nself
mv ~/.nself.backup ~/.nself

# Add to PATH if needed
export PATH="$HOME/.nself/bin:$PATH"
```

## Support

For issues or questions:
- Check [TROUBLESHOOTING.MD](TROUBLESHOOTING.MD)
- Run `nself doctor` for diagnostics
- Report issues at https://github.com/acamarata/nself/issues

## Summary

v0.3.0 is a major improvement in code organization and reliability. While the internal structure has changed significantly, the user experience remains familiar with enhanced error handling and recovery capabilities.