# nself clean - Clean Docker Resources

**Version 0.4.8** | Remove containers, volumes, and images

---

## Overview

The `nself clean` command removes Docker resources created by nself. It provides options for selective cleanup, from stopping containers to complete removal of all project data.

---

## Basic Usage

```bash
# Stop and remove containers
nself clean

# Remove containers and volumes
nself clean --volumes

# Remove everything including images
nself clean --all

# Dry run (show what would be removed)
nself clean --dry-run
```

---

## Cleanup Levels

### Level 1: Containers Only (Default)

```bash
nself clean
```

Removes:
- All project containers
- Project networks

Keeps:
- Volumes (database data)
- Images

### Level 2: With Volumes

```bash
nself clean --volumes
```

Removes:
- All project containers
- Project networks
- Named volumes (database data!)

Keeps:
- Images

### Level 3: Everything

```bash
nself clean --all
```

Removes:
- All project containers
- Project networks
- All volumes
- Project images

---

## Options Reference

| Option | Description |
|--------|-------------|
| `--volumes` | Also remove volumes |
| `--images` | Also remove images |
| `--all` | Remove everything |
| `--dry-run` | Show what would be removed |
| `--force` | Skip confirmation prompts |

---

## Safety Features

### Confirmation Required

```
⚠ This will remove all containers and volumes for 'myapp'
  All database data will be permanently deleted!

Are you sure? [y/N]
```

### Dry Run

```bash
nself clean --all --dry-run
```

Shows what would be removed without actually deleting.

---

## Common Use Cases

### Fresh Start

```bash
# Complete reset
nself clean --all
nself init
nself build
nself start
```

### Free Disk Space

```bash
# Remove unused images
nself clean --images

# Docker system prune
docker system prune -a
```

### Development Reset

```bash
# Keep images, reset data
nself clean --volumes
nself start
```

---

## See Also

- [reset](RESET.md) - Reset project configuration
- [stop](STOP.md) - Stop services
- [start](START.md) - Start services
