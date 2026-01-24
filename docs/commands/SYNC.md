# nself sync - Environment Synchronization

**Version 0.4.8** | Synchronize data and configuration between environments

---

## Overview

The `nself sync` command synchronizes databases, files, and configuration between environments. It enables workflows like pulling production data to staging or syncing configuration across team members.

---

## Basic Usage

```bash
# Sync database from production
nself sync db prod

# Sync configuration files
nself sync config staging

# Full sync
nself sync full staging
```

---

## Sync Types

### Database Sync

```bash
# Pull database from remote
nself sync db prod
nself sync db staging

# Push database to remote
nself sync db push staging
```

### File Sync

```bash
# Sync uploads/assets
nself sync files prod

# Sync specific directory
nself sync files prod --path uploads/
```

### Configuration Sync

```bash
# Sync .env files
nself sync config prod

# Sync all config
nself sync config prod --all
```

### Full Sync

```bash
# Database + files + config
nself sync full staging
```

---

## Environment Access

Sync requires SSH access to the target environment:

```bash
# Check access
nself env access --check staging

# Configure SSH
nself servers add staging user@staging.example.com
```

---

## Options Reference

| Option | Description |
|--------|-------------|
| `db` | Sync database |
| `files` | Sync files/uploads |
| `config` | Sync configuration |
| `full` | Sync everything |
| `--path` | Specific path to sync |
| `--dry-run` | Preview changes |
| `--force` | Skip confirmations |

---

## Safety Features

### Production Protection

```
⚠ Pulling production database
  This will overwrite local data!

Proceed? [y/N]
```

### Data Anonymization

```bash
# Anonymize sensitive data
nself sync db prod --anonymize
```

---

## See Also

- [env](ENV.md) - Environment management
- [db](DB.md) - Database operations
- [deploy](DEPLOY.md) - Deployment
