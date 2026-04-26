# Uninstalling ɳSelf

This page covers how to fully remove ɳSelf from your machine, including the CLI binary, local state, Docker containers, volumes, networks, and images.

## Contents

- [macOS](#macos)
- [Linux](#linux)
- [Windows (WSL2)](#windows-wsl2)
- [Verify zero residue](#verify-zero-residue)

## macOS

### 1. Remove the CLI binary

```bash
brew uninstall nself
```

If you installed manually instead of via Homebrew:

```bash
sudo rm /usr/local/bin/nself
```

### 2. Remove local state

```bash
rm -rf ~/.nself
```

### 3. Stop and remove Docker resources

Stop any running ɳSelf stack first:

```bash
# Inside your project directory
nself stop

# Or with Docker Compose directly if nself is already removed
cd /path/to/your/project
docker compose down --volumes --remove-orphans
```

Remove ɳSelf Docker images:

```bash
docker images | grep nself | awk '{print $3}' | xargs docker rmi -f
```

Remove any ɳSelf Docker networks:

```bash
docker network ls | grep nself | awk '{print $1}' | xargs docker network rm
```

### 4. Remove project files (optional)

Your project directory (containing `.env`, `.env.dev`, `docker-compose.yml`, etc.) is not touched by the steps above. Remove it manually if you no longer need it:

```bash
rm -rf /path/to/your/nself/project
```

## Linux

### 1. Remove the CLI binary

```bash
sudo rm /usr/local/bin/nself
```

Or if you installed to a different location:

```bash
which nself
sudo rm $(which nself)
```

### 2. Remove local state

```bash
rm -rf ~/.nself
```

### 3. Stop and remove Docker resources

```bash
# Inside your project directory
nself stop

# Remove containers and volumes
docker compose down --volumes --remove-orphans

# Remove nSelf Docker images
docker images | grep nself | awk '{print $3}' | xargs docker rmi -f

# Remove nSelf Docker networks
docker network ls | grep nself | awk '{print $1}' | xargs docker network rm
```

### 4. Remove project files (optional)

```bash
rm -rf /path/to/your/nself/project
```

## Windows (WSL2)

ɳSelf runs inside WSL2. The uninstall steps are the same as Linux, run inside your WSL2 terminal.

### 1. Remove the CLI binary (inside WSL2)

```bash
sudo rm /usr/local/bin/nself
rm -rf ~/.nself
```

### 2. Remove Docker resources (inside WSL2)

```bash
docker compose down --volumes --remove-orphans
docker images | grep nself | awk '{print $3}' | xargs docker rmi -f
docker network ls | grep nself | awk '{print $1}' | xargs docker network rm
```

### 3. Remove WSL2 distribution (optional, nuclear option)

If you want to remove the entire WSL2 environment:

```powershell
# In PowerShell (Windows)
wsl --unregister Ubuntu
```

This deletes the entire WSL2 virtual disk. Only do this if you have no other projects inside it.

## Verify zero residue

After completing the steps above, verify nothing remains:

```bash
# Binary is gone
which nself 2>&1 || echo "nself not found — OK"

# State directory is gone
ls ~/.nself 2>&1 || echo "~/.nself not found — OK"

# No nself containers running
docker ps -a | grep nself || echo "no nself containers — OK"

# No nself images
docker images | grep nself || echo "no nself images — OK"

# No nself networks
docker network ls | grep nself || echo "no nself networks — OK"
```

All five lines should report "not found" or "no nself X" for a clean uninstall.

---

← [[Installation]] | [[Home]] →
