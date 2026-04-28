# Chocolatey Packaging (stub)

Status: **stub** — publish target is v1.1.0.

This directory holds the Chocolatey package source for the `nself` CLI on Windows. It is wired up so that when v1.1.0 ships, the release workflow can:

1. Render `nself.nuspec` with the live version + asset URL + SHA256.
2. Render `tools/chocolateyInstall.ps1` (download zip, extract to `$env:ChocolateyInstall\lib\nself\tools`, drop a shim).
3. Run `choco pack nself.nuspec` to produce `nself.<version>.nupkg`.
4. Run `choco push nself.<version>.nupkg --api-key $CHOCO_API_KEY` to publish.

Until then, the canonical Windows install paths are:

- `iwr -useb https://install.nself.org/install.ps1 | iex` (native Windows binary)
- WSL2 + Linuxbrew: `brew tap nself-org/nself && brew install nself` (full functionality)

## Publish-day checklist (do not run yet)

- [ ] Bump `<version>` in `nself.nuspec` to the release version (no leading `v`).
- [ ] Author `tools/chocolateyInstall.ps1` (download from GitHub release, verify SHA256, extract).
- [ ] Author `tools/chocolateyUninstall.ps1`.
- [ ] Add `LICENSE.txt` and `VERIFICATION.txt` under `tools/`.
- [ ] Run `choco pack` locally and test install in a Windows sandbox.
- [ ] Submit to https://community.chocolatey.org/packages — manual review takes 1-3 days.
- [ ] Register `nself-org` Chocolatey publisher account (carry to user — see G3-T06 blocker).

## Blockers (carry to user)

- Chocolatey publisher account for `nself-org`: not registered. User action required.
- API key for unattended `choco push`: not in vault.
