# Security Advisory: CWE-214 Secret Exposure + Supply-Chain Installer Integrity

**Advisory ID:** nself-cli-2026-05-15  
**Date:** 2026-05-15  
**Affected component:** nself-org/cli  
**Severity:** High (both findings)  
**Fixed in:** v1.2.0  
**Chain ID:** a83c99d6  
**GitHub Security Advisory:** pending — published at tag time, not filed here

---

## Summary

Two independent security findings in the CLI were reported under chain ID `a83c99d6` and fixed in v1.2.0. The first exposed a process-visible secret via the Hasura admin secret passed as a command-line argument. The second allowed a man-in-the-middle attacker to substitute the Ollama installer script because its SHA-256 was not verified before execution.

---

## Finding 1: Hasura Admin Secret in Process Command Line (CWE-214) — High

### Affected versions

v1.1.x and earlier (any build before v1.2.0)

### Description

`cli/internal/backup/create.go` — `hasuraMetadataExportCmd()` — passed the Hasura admin secret as a `--admin-secret=<value>` flag in the argv array when shelling out to the Hasura CLI inside the backup Docker container. On Linux and macOS, any local user with read access to `/proc/<pid>/cmdline` (or `ps aux`) could read the secret while the subprocess ran. The window is short but deterministic: it exists for the full duration of the metadata export.

### Fix

The secret is now placed in the child process environment (`HASURA_GRAPHQL_ADMIN_SECRET`) via `cmd.Env = append(os.Environ(), "HASURA_GRAPHQL_ADMIN_SECRET="+cfg.Hasura.AdminSecret)`, and the Docker exec invocation passes `-e HASURA_GRAPHQL_ADMIN_SECRET` (the bare flag, not `=<value>`) so Docker reads the value from the client-side env. Nothing secret ever appears in an argv array.

A comment in the source (line 253) documents the invariant so future contributors do not reintroduce the pattern.

### CVSS base score

CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N — **7.1 (High)**

The attack requires local user access to the same machine running the backup command.

### Impact

An attacker with local read access to process state could obtain the Hasura admin secret during a `nself backup create` run. Possessing this secret grants full administrative access to the Hasura GraphQL engine, including unrestricted access to all database tables.

### Workaround (before upgrading)

None known for the argv exposure window. Operators who cannot upgrade immediately should restrict who can run `nself backup create` to accounts with database admin privileges anyway.

---

## Finding 2: Unsigned Installer Script Execution (Supply-Chain / MitM) — High

### Affected versions

v1.1.x and earlier (any build before v1.2.0)

### Description

`cli/internal/installer/ollama.go` downloaded the official Ollama installer script with `curl` and piped the result directly to `sh` without verifying the content. A network-level MitM attacker positioned between the CLI host and `https://ollama.com/install.sh` could substitute a malicious script. This is the classic "curl | sh" supply-chain attack vector.

### Fix

The installer no longer pipes curl output to sh. Instead:

1. The script is downloaded to a 0700 owner-only temporary directory using `DownloadAndVerify()` in the new `cli/internal/installer/verify.go`.
2. The temporary file is opened with `O_EXCL` to prevent symlink/TOCTOU race conditions.
3. The download body is capped at 2 MiB via `io.LimitReader` to prevent memory exhaustion from a malicious server.
4. The SHA-256 of the downloaded bytes is computed with `crypto/sha256` and compared against the pinned checksum returned by `ExpectedOllamaInstallChecksum()` (defined in `ollama_checksums.go`).
5. Execution only proceeds if the checksum matches. An empty expected checksum is unconditionally rejected.
6. The caller removes the temporary directory when done.

### CVSS base score

CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H — **8.1 (High)**

The High score reflects that code execution is possible, though the attack requires a network-level position (AC:H).

### Impact

A MitM attacker could execute arbitrary code as the user running `nself` during Ollama installer execution. In practice, most users run `nself` with a shell account that may have access to project secrets, SSH keys, and config files.

### Workaround (before upgrading)

There is no workaround for the `curl | sh` pattern short of not running `nself install ollama`. Operators who cannot upgrade should avoid that command and install Ollama out-of-band using the official signed package for their OS.

---

## Remediation

Upgrade to v1.2.0 or later. The fix ships in both findings with no configuration change required.

```sh
nself update
# or
brew upgrade nself
```

---

## Credit

Chain ID: a83c99d6 (external report via nSelf PCI inbox, 2026-05-15). No CVE has been assigned. A GitHub Security Advisory draft will be published at v1.2.0 tag time.

---

## See Also

- [[Supply-Chain]] — SBOM publication, Cosign signing, and installer verification procedures
- `cli/internal/installer/verify.go` — implementation of the download-and-verify helper
- `cli/internal/installer/ollama_checksums.go` — pinned SHA-256 registry
- `cli/internal/backup/create.go` — hasuraMetadataExportCmd with the env-passthrough pattern
