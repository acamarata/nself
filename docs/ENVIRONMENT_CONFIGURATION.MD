# nself Environment Configuration Guide

## Overview

nself uses a strict priority system for environment configuration files. This ensures predictable behavior across development, staging, and production environments.

## File Priority Order

nself loads environment variables from **ONLY ONE** file, in this strict priority order:

1. **`.env`** - Production configuration (highest priority)
2. **`.env.local`** - Development configuration (medium priority)
3. **`.env.dev`** - Team defaults (lowest priority)

### Important Rules

- **If `.env` exists**, both `.env.local` and `.env.dev` are **COMPLETELY IGNORED**
- **If `.env.local` exists** (and no `.env`), `.env.dev` is **COMPLETELY IGNORED**
- **Only ONE file is loaded** - there is NO merging of values between files
- After loading the chosen file, smart defaults are applied for any missing values

## Environment Files Explained

### `.env.example` (Reference Only)
- **Purpose**: Complete documentation of ALL available environment variables
- **Usage**: Reference only - NEVER loaded by the application
- **Location**: Created by `nself init` for documentation
- **Contents**: Every possible configuration option with detailed comments

### `.env.local` (Development)
- **Purpose**: Your primary development configuration
- **Usage**: Created by `nself init` with minimal settings
- **Best Practice**: Only include values you need to change from defaults
- **Git**: Can be committed if it contains no secrets

### `.env.dev` (Team Defaults)
- **Purpose**: Shared team development defaults
- **Usage**: Optional - only loaded if no `.env` or `.env.local` exists
- **Best Practice**: Include common team settings
- **Git**: Should be committed to share team defaults

### `.env` (Production)
- **Purpose**: Production configuration overrides
- **Usage**: Created for production deployments
- **Best Practice**: Never commit to Git, contains production secrets
- **Git**: Must be in `.gitignore`

## Quick Start

### Development Setup

```bash
# 1. Initialize project (creates .env.local and .env.example)
nself init

# 2. Edit .env.local with your settings
nano .env.local

# 3. Build and run
nself build
nself up
```

### Production Setup

```bash
# 1. Generate secure production config
nself prod

# 2. Review and edit .env
nano .env

# 3. Deploy
nself up
```

## Minimal Configuration

Thanks to smart defaults, you only need to specify what you want to change. A minimal `.env.local` might be:

```bash
# Just set your project name
PROJECT_NAME=myapp

# Everything else uses smart defaults!
```

## Common Configurations

### Enable Optional Services

```bash
# .env.local

PROJECT_NAME=myapp

# Enable extra services
REDIS_ENABLED=true
FUNCTIONS_ENABLED=true
DASHBOARD_ENABLED=true

# Add microservices
NESTJS_ENABLED=true
NESTJS_SERVICES=api,webhooks
```

### Production Configuration

```bash
# .env (production)

ENV=prod
PROJECT_NAME=myapp
BASE_DOMAIN=myapp.com

# Secure passwords (generate with: nself prod)
POSTGRES_PASSWORD=<generated-secure-password>
HASURA_GRAPHQL_ADMIN_SECRET=<generated-secure-secret>
HASURA_JWT_KEY=<generated-32-char-minimum>

# Production email
EMAIL_PROVIDER=sendgrid
SENDGRID_API_KEY=<your-api-key>
```

## Smart Defaults

nself provides sensible defaults for all configuration options. These defaults are:

- **Security-conscious**: Different defaults for dev vs prod
- **Platform-aware**: Adjusts for your OS and available resources
- **Dependency-smart**: Automatically configures related services
- **Override-friendly**: Any default can be overridden

### Examples of Smart Defaults

- `BASE_DOMAIN` defaults to `local.nself.org` (resolves to 127.0.0.1)
- `POSTGRES_PASSWORD` gets a secure default in dev, requires setting in prod
- `HASURA_GRAPHQL_ENABLE_CONSOLE` is `true` in dev, `false` in prod
- Service URLs are auto-computed from `BASE_DOMAIN`
- Docker network names are derived from `PROJECT_NAME`

## Testing Priority

To verify the environment loading priority in your project:

```bash
# Create test files
echo "TEST_VAR=from-dev" > .env.dev
echo "TEST_VAR=from-local" > .env.local
echo "TEST_VAR=from-prod" > .env

# Check which loads
nself doctor

# Clean up
rm .env.dev .env.local .env
```

## Troubleshooting

### Issue: Wrong environment file is loading

**Solution**: Check file existence in priority order:
```bash
ls -la .env .env.local .env.dev 2>/dev/null
```

### Issue: Values aren't what I expect

**Solution**: Verify which file is being loaded:
```bash
# Add temporary debug output
DEBUG=true nself build
```

### Issue: Secrets in Git

**Solution**: Ensure `.env` is in `.gitignore`:
```bash
echo ".env" >> .gitignore
git rm --cached .env  # If already committed
```

## Best Practices

1. **Development**: Use `.env.local` with minimal configuration
2. **Team Sharing**: Put shared defaults in `.env.dev`
3. **Production**: Use `.env` with full configuration and secrets
4. **Documentation**: Keep `.env.example` updated with new options
5. **Security**: Never commit `.env` with production secrets
6. **Simplicity**: Only configure what you need to change

## Migration from Old Versions

If upgrading from an older version of nself:

1. Your existing `.env.local` will continue to work
2. The new priority system is backward compatible
3. Consider moving production settings to `.env`
4. Review `.env.example` for new configuration options

## Summary

- **One file rules**: Only one environment file is loaded
- **Priority matters**: `.env` > `.env.local` > `.env.dev`
- **No merging**: Files don't merge - it's all or nothing
- **Smart defaults**: Unset values get sensible defaults
- **Keep it simple**: Only configure what you need to change