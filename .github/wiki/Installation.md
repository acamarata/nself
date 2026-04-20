# Installation

## OS Support

| Platform | Architecture | Status | Guide |
|---|---|---|---|
| macOS | Apple Silicon (arm64) | Supported | [macOS Apple Silicon](install/macos-apple-silicon) |
| macOS | Intel (amd64) | Supported | [macOS Intel](install/macos-intel) |
| Linux | x86_64 (amd64) | Supported | [Linux x86_64](install/linux-x86_64) |
| Linux | arm64 (aarch64) | Supported | [Linux arm64](install/linux-arm64) |
| Windows | WSL2 | Supported (WSL2 only) | [Windows WSL2](install/windows-wsl2) |
| Windows | Native (win64) | Planned v1.1.0 | — |
| Raspberry Pi | arm64 | Supported | [Raspberry Pi](install/raspberry-pi) |

**Windows native binary** (`windows-amd64.zip` + `install.ps1`) is planned for v1.1.0.
WSL2 + Docker Desktop is the supported path until then. See [[install/windows-wsl2]] for setup.

## Platform Guides

| Platform | Architecture | Description |
|---|---|---|
| [macOS Apple Silicon](install/macos-apple-silicon) | darwin/arm64 | M1/M2/M3/M4 — OrbStack vs Docker Desktop, VirtioFS |
| [macOS Intel](install/macos-intel) | darwin/amd64 | Intel Mac — Homebrew, binary download |
| [Linux x86_64](install/linux-x86_64) | linux/amd64 | Ubuntu, Debian, Fedora, Rocky, Amazon Linux |
| [Linux arm64](install/linux-arm64) | linux/arm64 | Graviton, Ampere Altra, Hetzner CAX, Pi |
| [Windows WSL2](install/windows-wsl2) | linux/amd64 | WSL2 + Docker Desktop; native Windows v1.1.0 |
| [Raspberry Pi](install/raspberry-pi) | linux/arm64 | Pi 4/5; RAM budgeting; minimal preset |

## Prerequisites

- **Docker** 24 or later
- **Docker Compose** v2 (included with Docker Desktop)
- **curl** and **bash** (for Linux/macOS installer)

## Quick Install

### macOS

```bash
brew install nself-org/nself/nself
```

See [[install/macos-apple-silicon]] or [[install/macos-intel]] for platform-specific notes.

### Linux

```bash
curl -sSL https://install.nself.org | bash
```

The script detects your architecture (amd64 or arm64) and downloads the correct binary.
See [[install/linux-x86_64]] or [[install/linux-arm64]] for manual install and verification.

### Windows (WSL2)

Windows requires WSL2. See [[install/windows-wsl2]] for the full step-by-step guide
covering WSL2 setup, Docker integration, and troubleshooting.

### Raspberry Pi

See [[install/raspberry-pi]] for RAM budgeting, SSD recommendations, and the minimal
services preset.

## Manual Install

1. Download the binary for your platform from [GitHub Releases](https://github.com/nself-org/cli/releases/latest).
2. Make it executable and place it on your `PATH`:

```bash
chmod +x nself-linux-amd64
sudo mv nself-linux-amd64 /usr/local/bin/nself
```

## Verify Your Install

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

This removes the CLI binary and local state. Your project `.env` files and Docker volumes
are preserved.

---
← [[Home]] | [[Quick-Start]] →
