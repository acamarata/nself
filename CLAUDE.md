# NSELF Development Guide

## Critical Architecture Decisions

### Smart Defaults System (v0.3.1+)
NSELF now uses a **smart defaults** system where all configuration options have sensible defaults. Users only need to specify what they want to change.

**Priority Order:** `.env` > `.env.local` > defaults

### Command Responsibilities

#### `nself init`
- **ONLY** creates environment files
- Creates `.env.local` (minimal, user edits)
- Creates `.env.example` (reference documentation)
- Does NOT create directories or generate files

#### `nself build` 
- Generates ALL project files from environment configuration
- Creates directories, docker-compose.yml, nginx configs, services, etc.
- Uses smart defaults for any missing configuration

#### `nself reset`
- Stops all containers and removes volumes
- Backs up env files with `.old` suffix
- Removes all generated files
- User can restart with `nself init` or restore with `.old` files

### Environment Files

#### `.env.example`
- Complete reference of ALL available options
- Shows default values
- Documentation only - NOT edited by users

#### `.env.local`
- User's configuration file
- Contains ONLY overrides from defaults
- Can be empty (everything has defaults)

#### `.env`
- Production overrides
- Highest priority
- Created by `nself prod` or manually

### Default Values
All configuration has defaults in `src/lib/config/smart-defaults.sh`:
- ENV=dev
- PROJECT_NAME=myproject  
- BASE_DOMAIN=local.nself.org
- All services disabled by default (enabled via .env.local)

## Implementation Details

### Smart Defaults Loading
```bash
# In any command that needs env:
source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"
load_env_with_defaults
```

### Template Structure
- `/src/templates/.env.example` - Full reference
- `/src/templates/.env.local` - Minimal template
- All service templates in `/src/templates/services/`

## Testing Requirements

When testing commands:
1. `nself init` should ONLY create env files
2. `nself build` should work even with empty .env.local
3. `nself reset` should backup env files with .old suffix
4. System should work with NO env files (pure defaults)

## Common Pitfalls to Avoid

1. **DO NOT** generate files in `init` command
2. **DO NOT** require any specific env variables
3. **DO NOT** fail if .env.local is empty or missing
4. **ALWAYS** use defaults for missing values
5. **NEVER** modify .env.example (it's reference only)

## Go Services Issues

### Problem
Go services fail with "missing go.sum entry" because:
1. Generated projects have empty go.sum files
2. Docker builds cache the problem
3. Host machine may not have Go installed

### Solution
1. Generate proper go.sum from template
2. Update Dockerfile to run `go mod tidy` during build
3. Provide fix scripts for existing projects

## User Communication

### No Assistant References
- Never mention any AI assistant in code, commits, or documentation
- Keep commits factual and technical
- No emoji in commits unless user requests

### Documentation Priority
1. User docs in `/docs`
2. Developer guide in `CLAUDE.md`
3. Comments minimal - code should be self-documenting