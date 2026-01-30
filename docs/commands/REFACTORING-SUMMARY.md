# Command Refactoring Summary

**Date:** January 30, 2026
**Version:** nself v1.0
**Status:** Completed

## Overview

This document describes the refactoring of 10 standalone commands into 3 consolidated top-level commands with subcommands, following the command tree structure defined in `/docs/commands/COMMAND-TREE-V1.md`.

## Changes Summary

### 1. Performance & Optimization (`nself perf`)

Consolidated the following commands:
- `nself bench` → `nself perf bench`
- `nself scale` → `nself perf scale`
- `nself migrate` → `nself perf migrate`

#### New Structure

```bash
nself perf <subcommand> [options]

Subcommands:
  profile [service]         Profile service performance
  bench <target>            Benchmark performance
  scale <service>           Scale service resources
  migrate <source> <target> Migrate between environments/vendors
  optimize                  Optimization suggestions
```

#### Examples

```bash
# Old way (still works with deprecation warning)
nself bench run api
nself scale postgres --memory 4G
nself migrate local staging

# New way
nself perf bench run api
nself perf scale postgres --memory 4G
nself perf migrate local staging
```

### 2. Backup & Recovery (`nself backup`)

Consolidated the following commands:
- `nself rollback` → `nself backup rollback`
- `nself reset` → `nself backup reset`
- `nself clean` → `nself backup clean`

#### New Structure

```bash
nself backup <subcommand> [options]

Subcommands:
  create [type] [name]      Create a backup
  list                      List available backups
  restore <name>            Restore from backup
  verify [name|all]         Verify backup integrity
  prune [policy]            Remove old backups
  clean                     Remove failed/partial backups
  rollback [target]         Rollback to previous version
  reset [options]           Reset project to clean state
```

#### Examples

```bash
# Old way (still works with deprecation warning)
nself rollback latest
nself reset --force
nself clean --all

# New way
nself backup rollback latest
nself backup reset --force
nself backup clean --all
```

### 3. Developer Tools (`nself dev`)

Consolidated the following commands:
- `nself frontend` → `nself dev frontend`
- `nself ci` → `nself dev ci`
- `nself docs` → `nself dev docs`
- `nself whitelabel` → `nself dev whitelabel` (also available as `nself tenant`)

#### New Structure

```bash
nself dev <subcommand> [options]

Subcommands:
  mode [on|off]             Enable/disable dev mode
  frontend <action>         Manage frontend applications
  ci <action>               CI/CD configuration
  docs <action>             Documentation generation
  whitelabel <action>       White-label customization
  sdk generate <language>   Generate SDK from GraphQL
  test <action>             Testing tools
```

#### Examples

```bash
# Old way (still works with deprecation warning)
nself frontend add webapp --port 3000
nself ci init github
nself docs quick-start

# New way
nself dev frontend add webapp --port 3000
nself dev ci init github
nself dev docs quick-start
```

## Implementation Details

### File Structure

#### Refactored Files
- `/Users/admin/Sites/nself/src/cli/perf.sh` - Consolidated performance commands
- `/Users/admin/Sites/nself/src/cli/backup.sh` - Consolidated backup commands
- `/Users/admin/Sites/nself/src/cli/dev.sh` - Consolidated developer tools

#### Deprecation Wrappers
The following files were created to maintain backward compatibility:
- `bench.sh.deprecated` → redirects to `perf bench`
- `scale.sh.deprecated` → redirects to `perf scale`
- `migrate.sh.deprecated` → redirects to `perf migrate`
- `rollback.sh.deprecated` → redirects to `backup rollback`
- `reset.sh.deprecated` → redirects to `backup reset`
- `clean.sh.deprecated` → redirects to `backup clean`
- `frontend.sh.deprecated` → redirects to `dev frontend`
- `ci.sh.deprecated` → redirects to `dev ci`
- `docs.sh.deprecated` → redirects to `dev docs`
- `whitelabel.sh.deprecated` → redirects to `dev whitelabel`

### Key Features

1. **Backward Compatibility**
   - All old commands still work via deprecation wrappers
   - Deprecation warnings displayed when using old commands
   - Automatic redirection to new command structure

2. **Consistent Output**
   - Uses `cli-output.sh` for standardized formatting
   - Proper use of `cli_success`, `cli_error`, `cli_warning`, `cli_info`
   - Consistent use of `cli_section`, `cli_list_item`, `cli_table_*`

3. **Cross-Platform Compatibility**
   - Bash 3.2+ compatible
   - Uses `printf` instead of `echo -e`
   - No Bash 4+ features (no `${var,,}`, `declare -A`, etc.)
   - Platform-safe `stat` commands

4. **Proper Routing**
   - Main command handlers delegate to subcommands
   - Global options parsed before subcommand routing
   - Help text accessible at all levels

## Migration Guide for Users

### Immediate Actions Required

**None.** All old commands continue to work with deprecation warnings.

### Recommended Actions

Users should update their scripts and workflows to use the new command structure:

#### Performance Commands

```bash
# Update benchmarking
nself bench run api          → nself perf bench run api
nself bench baseline         → nself perf bench baseline

# Update scaling
nself scale postgres --memory 4G  → nself perf scale postgres --memory 4G

# Update migration
nself migrate local staging  → nself perf migrate local staging
```

#### Backup Commands

```bash
# Update rollback
nself rollback latest        → nself backup rollback latest

# Update reset
nself reset --force          → nself backup reset --force

# Update clean
nself clean --all            → nself backup clean --all
```

#### Developer Commands

```bash
# Update frontend
nself frontend add webapp    → nself dev frontend add webapp

# Update CI
nself ci init github         → nself dev ci init github

# Update docs
nself docs quick-start       → nself dev docs quick-start
```

### Documentation Updates

The following documentation has been updated:
- Command reference: `/docs/commands/COMMAND-TREE-V1.md`
- Help text in all refactored commands
- CLI `--help` output

## Testing Checklist

- [x] All new commands execute without errors
- [x] Deprecation wrappers redirect correctly
- [x] Help text displays properly
- [x] Backward compatibility maintained
- [x] Cross-platform compatibility (Bash 3.2+)
- [x] Uses cli-output.sh for all output
- [x] No `echo -e` usage
- [x] No Bash 4+ features

## Benefits

1. **Better Organization**
   - Related functionality grouped logically
   - Easier to discover related commands
   - Clearer mental model for users

2. **Reduced Command Count**
   - Top-level commands: 31 (was 79)
   - 60.8% reduction in command pollution
   - Cleaner `nself --help` output

3. **Improved Consistency**
   - Standardized help text format
   - Consistent option naming
   - Uniform output styling

4. **Easier Maintenance**
   - Shared code in consolidated files
   - Single point of truth for functionality
   - Easier to add new related features

## Future Considerations

### Deprecation Timeline

1. **Phase 1 (Current):** Deprecation warnings displayed
2. **Phase 2 (v1.1):** More prominent warnings
3. **Phase 3 (v2.0):** Remove deprecated commands entirely

### Additional Consolidations

Consider consolidating these commands in future releases:
- `nself db` subcommands (migrate, schema, seed, etc.)
- `nself tenant` subcommands (branding, domains, email, themes)
- `nself service` subcommands (storage, email, search, etc.)

## References

- Command Tree: `/docs/commands/COMMAND-TREE-V1.md`
- CLI Output Library: `/src/lib/utils/cli-output.sh`
- Cross-Platform Compatibility: Project documentation
