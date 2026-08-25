# nself verify-sbom

<!-- BEGIN PROSE:summary -->
> Verify the SBOM signature for a CLI release.
<!-- END PROSE:summary -->

## Synopsis

```
nself verify-sbom [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Download and verify the CycloneDX SBOM cosign bundle for a CLI release.

## Usage

```bash
nself verify-sbom --version <v>
```

## What It Does

1. Downloads `sbom-cli-{version}.cdx.json` from the GitHub Release.
2. Downloads `sbom-cli-{version}.cdx.json.bundle` (cosign bundle).
3. Runs `cosign verify-blob` with the OIDC certificate from GitHub Actions.
4. Prints `VERIFIED` on success or exits 1 on failure.

## Requirements

- `cosign` must be in PATH: `brew install cosign`
- The version must correspond to a published GitHub Release.

## Example

```bash
nself verify-sbom --version v1.1.1
# Downloading SBOM for v1.1.1...
# Verifying cosign bundle signature...
# SBOM for v1.1.1: VERIFIED
```

## Related

- [[cmd-secrets]], secret management and rotation
- [[security/Supply-Chain]], supply-chain security baseline
- [[Home]]
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | `nself-org/cli` | GitHub repo (owner/name) |
| `--version` | `""` | Release version to verify (e.g. v1.0.9) |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself verify-sbom --version v1.0.9
  nself verify-sbom --version v1.0.10 --repo nself-org/cli
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
