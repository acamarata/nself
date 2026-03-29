> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# DRM Plugin

> Digital Rights Management for video — Widevine and FairPlay license server. **Pro plugin.**

> **Requires:** Pro license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install drm
```

## What It Does

Runs a DRM license server supporting Google Widevine (L1/L3) and Apple FairPlay Streaming. Issues time-limited decryption keys to authorized viewers. Integrates with the `transcoder` plugin to encrypt HLS content during encoding, and the `tokens` plugin to verify viewer entitlements before issuing DRM licenses.

## Dependencies

Requires `transcoder` for content encryption and `tokens` for entitlement checks.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DRM_PORT` | `3060` | DRM license server port |
| `DRM_WIDEVINE_ENABLED` | `true` | Enable Widevine DRM |
| `DRM_FAIRPLAY_ENABLED` | `true` | Enable FairPlay DRM |
| `DRM_WIDEVINE_KEY_SEED` | *(auto-generated)* | Widevine encryption key seed |
| `DRM_FAIRPLAY_CERT_FILE` | — | FairPlay certificate file path |
| `DRM_LICENSE_DURATION` | `3600` | License duration in seconds |

## Ports

| Port | Purpose |
|------|---------|
| 3060 | DRM license server |

## Database Tables

3 tables added to your Postgres database:
- `np_drm_content_keys` — per-content encryption key store
- `np_drm_licenses` — issued license records
- `np_drm_audit` — license request audit log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/drm/widevine` | Widevine license endpoint |
| `/drm/fairplay` | FairPlay license endpoint |
