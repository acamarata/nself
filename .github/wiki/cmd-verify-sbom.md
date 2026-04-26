# nself verify-sbom

Download and verify the CycloneDX SBOM cosign bundle for a CLI release.

## Usage

```bash
nself verify-sbom --version <v>
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--version` | required | Release version to verify (e.g. `v1.0.9`) |
| `--repo` | `nself-org/cli` | GitHub repo (`owner/name`) |

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
nself verify-sbom --version v1.0.9
# Downloading SBOM for v1.0.9...
# Verifying cosign bundle signature...
# SBOM for v1.0.9: VERIFIED
```

## Related

- [[cmd-secrets]], secret management and rotation
- [[security/Supply-Chain]], supply-chain security baseline
- [[Home]]
