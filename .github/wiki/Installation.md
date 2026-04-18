# Installation

## Contents

- [Prerequisites](#prerequisites)
- [macOS](#macos)
- [Linux](#linux)
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

## Manual Install

1. Download the binary for your platform from [GitHub Releases](https://github.com/nself-org/cli/releases/latest).
2. Make it executable and place it on your `PATH`:

```bash
chmod +x nself-linux-amd64
sudo mv nself-linux-amd64 /usr/local/bin/nself
```

## Verify Installation

```bash
nself version
```

Expected output:

```
nself v1.0.9 (linux/amd64)
```

## Uninstall

**Homebrew:**
```bash
brew uninstall nself
```

**Linux / manual:**
```bash
sudo rm /usr/local/bin/nself
rm -rf ~/.nself
```

This removes the CLI binary and the local state directory. Your project `.env` files and Docker volumes are not touched.

---
← [[Home]] | [[Quick-Start]] →
