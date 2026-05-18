# Supply Chain Security

Every ɳSelf CLI release publishes a signed Software Bill of Materials (SBOM) alongside the release tarballs. This page explains what is shipped, how to verify it, and how to query it for dependency exposure.

---

## What is published on every release

| Artifact | Format | Description |
|---|---|---|
| `sbom.cdx.json` | CycloneDX 1.5 JSON | Primary SBOM: all Go module dependencies with versions and purls |
| `sbom.cdx.json.bundle` | Cosign bundle | Sigstore keyless signature for the CycloneDX SBOM |
| `sbom.cdx.json.sig` | Cosign sig | Legacy per-artifact signature (same key material) |
| `sbom.spdx.json` | SPDX 2.3 JSON | Secondary SBOM: same dependency data in SPDX format |
| `sbom.spdx.json.sig` | Cosign sig | Sigstore keyless signature for the SPDX SBOM |
| `checksums.txt` | SHA-256 | SHA-256 of tarballs + both SBOMs |

All artifacts are published to the GitHub Release page for each `v*` tag.

---

## How signing works

Signatures use [Cosign keyless signing](https://docs.sigstore.dev/cosign/signing/overview/) via the Sigstore ecosystem. There are no long-lived private keys.

The signature is bound to the GitHub Actions OIDC token issued during the release workflow run:

| Field | Value |
|---|---|
| Signing tool | `cosign sign-blob` (Cosign v2+) |
| OIDC issuer | `https://token.actions.githubusercontent.com` |
| Identity | `https://github.com/nself-org/cli/.github/workflows/release.yml@refs/tags/v*` |
| Transparency log | [Rekor](https://rekor.sigstore.dev) (public, immutable) |
| Key rotation | None required (keyless , OIDC token is the ephemeral key) |

---

## Verifying a release SBOM

### Quick verify (downloads automatically)

```bash
bash tools/sbom/verify.sh v1.0.9
```

### Manual verify with cosign

```bash
# Download the SBOM and its bundle from the GitHub Release page
curl -fsSL https://github.com/nself-org/cli/releases/download/v1.0.9/sbom.cdx.json \
  -o sbom.cdx.json
curl -fsSL https://github.com/nself-org/cli/releases/download/v1.0.9/sbom.cdx.json.bundle \
  -o sbom.cdx.json.bundle

# Verify — exits 0 on success, 1 on failure
cosign verify-blob sbom.cdx.json \
  --bundle sbom.cdx.json.bundle \
  --certificate-identity-regexp "^https://github.com/nself-org/cli/.github/workflows/release\.yml@refs/tags/v" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --rekor-url "https://rekor.sigstore.dev"
```

A successful verify prints `Verified OK`. Failure prints `Error: verifying blob` and exits 1, this means the SBOM was tampered or was not produced by the ɳSelf release CI.

### Install cosign

```bash
# macOS
brew install cosign

# Linux
curl -sSfL https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64 \
  -o /usr/local/bin/cosign && chmod +x /usr/local/bin/cosign

# Verify install
cosign version
```

---

## Querying the SBOM

Use `tools/sbom/query.sh` to search for a package or check CVE exposure:

```bash
# Does release v1.0.9 include package "cobra"?
bash tools/sbom/query.sh v1.0.9 --pkg cobra

# What does CVE-2024-45337 affect in v1.0.9?
# Requires grype: https://github.com/anchore/grype
bash tools/sbom/query.sh v1.0.9 --cve CVE-2024-45337

# Query a locally-downloaded SBOM
bash tools/sbom/query.sh --local sbom.cdx.json --pkg spf13
```

---

## Generating an SBOM locally

```bash
# Requires syft: https://github.com/anchore/syft
make sbom
```

This writes `sbom.spdx.json` and `sbom.cdx.json` in the repo root (not committed).

---

## Checking SHA-256 integrity

The `checksums.txt` file in each release includes SHA-256 hashes of all tarballs and both SBOMs. To verify an SBOM has not been replaced after signing:

```bash
# Download checksums.txt and sbom.cdx.json
sha256sum --check --ignore-missing checksums.txt
```

---

## SLSA compliance status

SBOM + cosign keyless signing moves ɳSelf CLI toward SLSA Level 2:

| SLSA requirement | Status |
|---|---|
| Source code on version control | Done (GitHub) |
| Build by CI (not developer laptop) | Done (GitHub Actions) |
| Provenance: build instructions | Done (`provenance.intoto.jsonl` via slsa-github-generator) |
| Provenance: hermetic build | Partial (vendored deps; CGO_ENABLED=0) |
| Signed provenance | Done (cosign keyless) |
| SBOM attached to release | Done (CycloneDX + SPDX, Q04) |

---

## Ollama installer SHA pinning

The `nself install ollama` command downloads the official Ollama install script from
`https://ollama.com/install.sh` and executes it. To prevent supply-chain tampering
(MitM or compromised CDN), the CLI pins the expected SHA-256 of the installer
script and refuses to execute it if the hash does not match.

The pinned checksum lives in `internal/installer/ollama_checksums.go` as
`PinnedOllamaInstallSHA256["<version>"]`. Verification is **unconditional**:
there is no flag or code path that skips it.

### TOCTOU protection

The verified script is written into a private `0700` owner-only temporary
directory (not the shared system `$TMPDIR`). The script file itself is
created `0600`. This prevents a same-uid process from replacing the file
between the checksum verification and the `sh` exec call.

The parent directory (not just the file) is removed with `os.RemoveAll`
after execution.

### Body size cap

Downloads are capped at **2 MiB** via `io.LimitReader`. A response body
larger than 2 MiB is truncated before hashing. Because the pinned hash
covers the full script content, a truncated body produces a checksum
mismatch and `DownloadAndVerify` returns an error. This means an
oversized response cannot be silently accepted; it fails loudly at the
verification step.

### Updating the pin

When Ollama ships a new installer:

```bash
# Download and compute the new hash
curl -fsSL https://ollama.com/install.sh | sha256sum

# Update the map entry in internal/installer/ollama_checksums.go,
# then run the test suite to confirm the pin is non-empty.
go test -mod=vendor ./internal/installer/...
```

The `TestExpectedOllamaInstallChecksum_NonEmpty` test enforces that the pinned
value is always a non-empty 64-character hex string and will fail CI if the entry
is accidentally blanked out.

### Automated drift monitor

`.github/workflows/ollama-pin-drift.yml` runs every Monday at 08:00 UTC and
on `workflow_dispatch`. It fetches the live installer, computes its SHA-256,
and compares it to the pinned value. On mismatch it opens a GitHub issue
tagged `security` and `ollama-pin-drift` with remediation instructions.

The monitor is informational — it does not block PRs — but any open
`ollama-pin-drift` issue must be resolved before the next CLI release.

---

## Related pages

- [[Security-Architecture]], threat model and hardening decisions
- [[Security-Hardening]], operator hardening guide
- [[Security-Policy]], reporting vulnerabilities
- [Security Advisory 2026-05-15](../SECURITY-ADVISORIES/2026-05-15-rce-and-secrets.md) — supply-chain installer integrity fix (finding 2, High, fixed v1.2.0)
- [[Home]]
