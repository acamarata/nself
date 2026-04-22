# Errors

Quick reference for user-journey errors from the nSelf CLI. Every error printed
to your terminal includes a stable code, a plain-language explanation, and steps
to fix it.

---

## How errors appear

```
Error: [ERR-INSTALL-001] Docker not found in PATH
  Why:  The docker binary is not installed or is not on your system PATH.
  Fix:  Install Docker Desktop from https://docs.docker.com/get-docker/
        then restart your terminal.
        Need help? https://nself.org/support · Discord: https://discord.gg/nself
        · GitHub: https://github.com/orgs/nself-org/discussions
  Docs: https://github.com/nself-org/cli/wiki/Errors#err-install-001
```

The code in brackets is stable across CLI versions. Search this page (Ctrl+F / Cmd+F) by code for the cause and fix.

---

## Install errors (ERR-INSTALL-*)

### ERR-INSTALL-001

**Docker not found in PATH**

| Field | Value |
|-------|-------|
| Why | The docker binary is not installed or is not on your system PATH. |
| Fix | Install Docker Desktop from https://docs.docker.com/get-docker/ then restart your terminal. |

### ERR-INSTALL-002

**Docker daemon is not running**

| Field | Value |
|-------|-------|
| Why | The Docker daemon did not respond within the check timeout. |
| Fix | macOS: open Docker Desktop. Linux: `sudo systemctl start docker` |

### ERR-INSTALL-003

**Required dependency missing**

| Field | Value |
|-------|-------|
| Why | A binary required by nself was not found on your PATH. |
| Fix | macOS: `brew install <dep>`. Ubuntu/Debian: `sudo apt install <dep>`. Arch: `sudo pacman -S <dep>` |

### ERR-INSTALL-004

**Permission denied during install**

| Field | Value |
|-------|-------|
| Why | The install target directory is not writable by the current user. |
| Fix | Run the install command with sudo, or change the install prefix to a user-writable path. |

### ERR-INSTALL-005

**Insufficient disk space**

| Field | Value |
|-------|-------|
| Why | The target filesystem does not have enough free space. |
| Fix | Free at least 2 GB of disk space then retry. Run `df -h` to check free space. |

---

## License errors (ERR-LICENSE-*)

### ERR-LICENSE-001

**License key is invalid**

| Field | Value |
|-------|-------|
| Why | The key format was not recognised or ping.nself.org rejected it. |
| Fix | Check the key at https://nself.org/account/license. Keys start with `nself_pro_`. |

### ERR-LICENSE-002

**License key has expired**

| Field | Value |
|-------|-------|
| Why | The subscription associated with this key is no longer active. |
| Fix | Renew at https://nself.org/account/license or run `nself license info` to check status. |

### ERR-LICENSE-003

**Cannot reach license server**

| Field | Value |
|-------|-------|
| Why | ping.nself.org is not reachable and no valid cached license was found. |
| Fix | Check your internet connection. For offline mode, set `NSELF_LICENSE_OFFLINE=1`. Cached licenses remain valid for 7 days. |

---

## Plugin errors (ERR-PLUGIN-*)

### ERR-PLUGIN-001

**Plugin not found in registry**

| Field | Value |
|-------|-------|
| Why | No plugin with that name is registered in the nSelf plugin registry. |
| Fix | Run `nself plugin list` to see available plugins. Check spelling. |

### ERR-PLUGIN-002

**Plugin config is stale**

| Field | Value |
|-------|-------|
| Why | The installed plugin version differs from what was used to generate docker-compose.yml. |
| Fix | Run `nself build` to regenerate the compose file, then `nself start`. |

---

## Network errors (ERR-NETWORK-*)

### ERR-NETWORK-001

**Network unavailable**

| Field | Value |
|-------|-------|
| Why | A required outbound connection could not be established. |
| Fix | Check your internet connection. If behind a proxy, set `HTTP_PROXY` / `HTTPS_PROXY` environment variables. |

---

## Permission errors (ERR-PERMISSION-*)

### ERR-PERMISSION-001

**Permission denied**

| Field | Value |
|-------|-------|
| Why | The current user does not have the required file or directory access rights. |
| Fix | Check ownership with `ls -la`. Change permissions or run as the correct user. |

---

## Port conflict errors (ERR-PORT-*)

### ERR-PORT-001

**Port already in use**

| Field | Value |
|-------|-------|
| Why | The port required by nself is occupied by another process. |
| Fix | Stop the conflicting process or set the port override in `.env.local` (e.g. `POSTGRES_PORT=5433`). Run `nself doctor` to identify the occupying process. |

---

## Config errors (ERR-CONFIG-*)

### ERR-CONFIG-001

**.env file is malformed**

| Field | Value |
|-------|-------|
| Why | The environment file contains a syntax error or an invalid key. |
| Fix | Each line must be `KEY=value`. Remove blank lines, quotes around keys, or spaces around `=`. |

---

## Need help?

- Support portal: https://nself.org/support
- Community Discord: https://discord.gg/nself
- GitHub Discussions: https://github.com/orgs/nself-org/discussions

For the legacy E001-E399 Docker/config/plugin error codes, see [[error-codes]].

---

← [[Home]] | [[Support]] | [[cmd-doctor]] | [[_Sidebar]]
