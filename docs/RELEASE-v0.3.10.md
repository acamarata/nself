# nself v0.3.10 Release Notes

**Release Date**: September 5, 2024  
**Type**: Minor Release - Search Services & nself-admin Integration

## Overview

Version 0.3.10 introduces comprehensive search service support with 6 different engines, full compatibility with nself-admin wizard, and extensive documentation updates. This release focuses on making nself feature-complete for the admin initialization wizard while adding powerful search capabilities.

## New Features

### 🔍 Search Services (6 Engines)

Added full support for 6 different search engines, all self-hosted:

1. **Meilisearch** (default)
   - Lightning-fast search with typo tolerance
   - Built-in dashboard
   - Port: 7700

2. **Typesense**
   - Real-time search (<50ms)
   - Vector search support
   - Port: 8108

3. **Zinc**
   - Lightweight Elasticsearch alternative
   - Minimal resource usage
   - Port: 4080

4. **Elasticsearch**
   - Industry standard
   - Full aggregation support
   - Port: 9200

5. **OpenSearch**
   - AWS's Elasticsearch fork
   - Advanced security features
   - Ports: 9200 (API), 5601 (Dashboard)

6. **Sonic**
   - Ultra-lightweight (30MB RAM)
   - Perfect for embedded systems
   - Port: 1491

### 🎛️ nself-admin Integration

Full compatibility with nself-admin initialization wizard:

- **Service Enable Flags**: All services now have explicit enable/disable flags
- **Frontend App Support**: Configure multiple frontend applications with routing
- **Variable Compatibility**: Full backward compatibility for renamed variables
- **Environment Cascade**: Proper support for `.env.dev` → `.env` loading order

### 📚 Documentation

- **New**: `SEARCH.md` - Comprehensive search services guide
- **New**: `ENVIRONMENT-VARIABLES.md` - Complete variable reference
- **Updated**: `COMMANDS.md` - Added search command documentation
- **Created**: Integration specifications for nself-admin team

## Breaking Changes

None. Full backward compatibility maintained.

## Improvements

### Configuration Management

- Added smart defaults for all new variables
- Improved environment loading with proper cascade
- Enhanced validation for service configurations
- Better error messages for configuration issues

### Service Enable Flags

All services now use consistent enable flags:
```bash
POSTGRES_ENABLED=true      # Core services default to true
HASURA_ENABLED=true
AUTH_ENABLED=true
STORAGE_ENABLED=true
NSELF_ADMIN_ENABLED=false  # Optional services default to false
REDIS_ENABLED=false
SEARCH_ENABLED=false
```

### Frontend Application Support

Support for multiple frontend apps with two formats:

**Individual Variables** (Wizard-friendly):
```bash
FRONTEND_APP_COUNT=2
FRONTEND_APP_1_NAME=web
FRONTEND_APP_1_PORT=3001
FRONTEND_APP_1_PREFIX=app
```

**Compact Format** (CLI-friendly):
```bash
FRONTEND_APPS="web:3001:app,mobile:3002:m"
```

### Variable Mappings

Automatic mapping for backward compatibility:
- `NADMIN_ENABLED` → `NSELF_ADMIN_ENABLED`
- `MINIO_ENABLED` ↔ `STORAGE_ENABLED`
- `DB_BACKUP_*` → `BACKUP_*`

## Bug Fixes

- Fixed search command implementation
- Resolved variable loading in test environments
- Corrected Docker compose generation for search services
- Fixed environment cascade for proper override behavior

## Configuration Changes

### New Variables

**Search Configuration**:
```bash
SEARCH_ENABLED=false
SEARCH_ENGINE=meilisearch
MEILISEARCH_MASTER_KEY=<generated>
TYPESENSE_API_KEY=<generated>
ZINC_ADMIN_PASSWORD=<generated>
SONIC_PASSWORD=<generated>
```

**Admin Configuration**:
```bash
PROJECT_DESCRIPTION=""
ADMIN_EMAIL=admin@example.com
NGINX_CLIENT_MAX_BODY_SIZE=100M
```

### Updated Defaults

- All core services now explicitly default to `true`
- Search services properly configured with secure defaults
- Admin UI integrated with proper authentication

## Migration Guide

### From v0.3.9

1. **Update nself**:
   ```bash
   nself update
   ```

2. **Enable search** (optional):
   ```bash
   # Edit .env
   SEARCH_ENABLED=true
   SEARCH_ENGINE=meilisearch  # or your choice
   
   # Rebuild and restart
   nself build
   nself restart
   ```

3. **Configure frontend apps** (if needed):
   ```bash
   # Edit .env
   FRONTEND_APP_COUNT=1
   FRONTEND_APP_1_NAME=web
   FRONTEND_APP_1_PORT=3001
   FRONTEND_APP_1_PREFIX=app
   ```

### Variable Updates

If using old variable names, they'll continue to work but consider updating:
- `NADMIN_ENABLED` → `NSELF_ADMIN_ENABLED`
- `DB_BACKUP_ENABLED` → `BACKUP_ENABLED`
- `DB_BACKUP_SCHEDULE` → `BACKUP_SCHEDULE`

## Performance Impact

Search services resource usage:
- **Minimal**: Sonic (30-100MB RAM)
- **Low**: Zinc (100-500MB RAM)
- **Medium**: Meilisearch (500MB-2GB RAM), Typesense (200MB-1GB RAM)
- **High**: Elasticsearch/OpenSearch (2GB-8GB RAM minimum)

Choose based on your needs and available resources.

## Security Notes

- All search services configured with authentication
- API keys auto-generated during build
- Services not exposed externally by default
- Use nginx proxy for external access

## Testing

### Search Services
```bash
# Enable and test Meilisearch
SEARCH_ENABLED=true SEARCH_ENGINE=meilisearch nself build
nself start
nself search status

# Test other engines
for engine in typesense zinc elasticsearch opensearch sonic; do
  SEARCH_ENGINE=$engine nself build
  nself restart search
  nself search status
done
```

### Frontend Apps
```bash
# Configure frontend app
echo "FRONTEND_APP_COUNT=1" >> .env
echo "FRONTEND_APP_1_NAME=web" >> .env
echo "FRONTEND_APP_1_PORT=3001" >> .env
nself build
nself nginx reload
```

## Known Issues

- Auth service health check reports unhealthy (port 4001 vs 4000) but service works correctly
- MLflow is not yet implemented (removed from documentation)
- Functions service partially implemented (configuration only, no runtime yet)

## What's Next (v0.4.0)

- Kubernetes deployment support
- Functions runtime implementation
- Multi-node clustering
- Advanced monitoring with alerts
- MLflow integration

## Credits

Special thanks to:
- nself-admin team for integration requirements
- Community for search engine suggestions
- Contributors for testing and feedback

## Support

- GitHub Issues: https://github.com/acamarata/nself/issues
- Documentation: /docs/
- Discord: [Coming Soon]

## Checksums

```
nself-v0.3.10.tar.gz: [SHA256 will be added after release]
```

---

*For detailed upgrade instructions, see [MIGRATION.md](./MIGRATION.md)*  
*For complete documentation, see [README.md](./README.md)*