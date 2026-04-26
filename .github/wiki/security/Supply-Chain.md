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

## Related pages

- [[Security-Architecture]], threat model and hardening decisions
- [[Security-Hardening]], operator hardening guide
- [[Security-Policy]], reporting vulnerabilities
- [[Home]]
