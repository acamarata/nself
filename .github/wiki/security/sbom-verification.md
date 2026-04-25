# SBOM Verification

Every nSelf CLI release ships with a Software Bill of Materials (SBOM) in both
CycloneDX and SPDX formats, signed with cosign via GitHub Actions OIDC.

## Verifying a Release SBOM

```bash
# Install cosign
brew install cosign

# Verify the SBOM for a specific release
nself verify-sbom --version v1.0.9
```

Expected output:

```
Downloading SBOM for v1.0.9...
Verifying cosign bundle signature...
SBOM for v1.0.9: VERIFIED
```

## What Is Verified

1. The CycloneDX SBOM (`sbom-cli-{version}.cdx.json`) is downloaded from the GitHub Release.
2. Its cosign bundle (`sbom-cli-{version}.cdx.json.bundle`) is downloaded.
3. `cosign verify-blob` confirms the bundle was signed by a GitHub Actions workflow
   in the `nself-org/cli` repository.
4. Any tampering with the SBOM file will cause verification to fail.

## Manual Verification (Without the CLI)

```bash
cosign verify-blob sbom-cli-v1.0.9.cdx.json \
  --bundle sbom-cli-v1.0.9.cdx.json.bundle \
  --certificate-identity-regexp "^https://github.com/nself-org/cli/" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

## SBOM Contents

Each SBOM includes the NTIA-mandated minimum elements:

- Supplier name and component name
- Component version
- Unique identifier (Package URL / PURL)
- Dependency relationships
- Author identity and timestamp

## Release Pipeline

SBOMs are generated on every `release-published` event by `.github/workflows/sbom.yml`:

1. `syft scan` generates `sbom-cli-{version}.cdx.json` (CycloneDX) and `.spdx.json` (SPDX).
2. `cosign sign-blob` creates a keyless cosign bundle via GitHub Actions OIDC.
3. Both SBOMs and the bundle are attached to the GitHub Release as assets.
4. The Docker image `nself/nself-admin:{version}` is also signed with `cosign sign`.

A dry-run validation runs on every PR via the `sbom.yml` (Q04) workflow.

## Compliance References

- Executive Order 14028 — US federal software supply chain
- EU CRA (Cyber Resilience Act, effective 2027)
- NTIA Minimum SBOM Elements

## Related

- [[cmd-verify-sbom]] — CLI reference for SBOM verification
- [[security/Supply-Chain]] — full supply-chain security baseline
- [[Home]]
