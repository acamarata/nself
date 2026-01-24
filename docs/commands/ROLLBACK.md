# nself rollback - Deployment Rollback

**Version 0.4.8** | Roll back to previous deployment

---

## Overview

The `nself rollback` command reverts to a previous deployment state. It's useful for recovering from failed deployments or reverting problematic changes.

> **Note**: This is a legacy command. Use `nself deploy rollback` for new workflows.

---

## Basic Usage

```bash
# Rollback to previous deployment
nself rollback

# Rollback to specific version
nself rollback --version v1.2.3

# Rollback specific environment
nself rollback staging
```

---

## Rollback Types

### Quick Rollback

```bash
# Roll back one version
nself rollback
```

Reverts to the previous deployment using stored state.

### Version Rollback

```bash
# Roll back to specific tag
nself rollback --version v1.2.3
```

Deploys a specific tagged version.

### Database Rollback

```bash
# Roll back with database
nself rollback --include-db
```

Also restores the database from the matching backup.

---

## Deployment History

View available rollback points:

```bash
nself history
```

```
Deployment History
─────────────────────────────────────────────────────────────────
  #1  v1.2.4  2024-01-20 10:15  current
  #2  v1.2.3  2024-01-19 14:30  stable
  #3  v1.2.2  2024-01-18 09:00
```

---

## Options Reference

| Option | Description |
|--------|-------------|
| `--version` | Target version/tag |
| `--include-db` | Also rollback database |
| `--dry-run` | Preview changes |
| `--force` | Skip confirmations |

---

## See Also

- [deploy](DEPLOY.md) - Deployment
- [history](HISTORY.md) - Deployment history
