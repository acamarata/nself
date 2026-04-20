# Installation

## Contents

- [Prerequisites](#prerequisites)
- [macOS](#macos)
- [Linux](#linux)
- [Windows (WSL2)](#windows-wsl2)
- [Manual Install](#manual-install)
- [Verify Installation](#verify-installation)
- [Uninstall](#uninstall)

## Prerequisites

- **Docker** 24 or later
- **Docker Compose** v2 (included with Docker Desktop)
- **curl** and **bash** (for Linux installer)

## macOS

Install via Homebrew using the official tap:

```bash
brew install nself-org/nself/nself
```

This installs the latest stable release and keeps it up to date with `brew upgrade`.

## Linux

Install with the official installer script:

```bash
curl -sSL https://install.nself.org | bash
```

The script detects your architecture (amd64 or arm64), downloads the appropriate binary, and places it in `/usr/local/bin`.

## Windows (WSL2)

Windows requires WSL2. See [[windows-wsl2]] for the full step-by-step guide covering WSL2 setup, Docker integration, Linuxbrew install, and troubleshooting.

## Manual Install

1. Download the binary for your platform from [GitHub Releases](https://github.com/nself-org/cli/releases/latest).
2. Make it executable and place it on your `PATH`:

```bash
chmod +x nself-linux-amd64
sudo mv nself-linux-amd64 /usr/local/bin/nself
```

## Verify your install

Run these two commands after installation to confirm everything is working:

```bash
nself version
```

Expected output:

```
nself v1.0.9 (linux/amd64)
```

```bash
nself doctor
```

Expected output (all green):

```
✓ Docker: running (v26.1.0)
✓ Docker Compose: v2.27.0
✓ curl: available
✓ bash: available
✓ Architecture: amd64
```

If `nself doctor` reports a problem, see [[Errors]] for remediation steps.

## Uninstall

See [[install/uninstall]] for full per-OS instructions including Docker cleanup steps.

**Quick uninstall:**

```bash
# macOS (Homebrew)
brew uninstall nself

# Linux / manual
sudo rm /usr/local/bin/nself
rm -rf ~/.nself
```

This removes the CLI binary and local state. Your project `.env` files and Docker volumes are preserved.

---
← [[Home]] | [[Quick-Start]] →
