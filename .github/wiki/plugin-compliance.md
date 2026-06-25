# Compliance Plugin

> GDPR, CCPA, HIPAA, SOC 2, and PCI compliance, DSARs, consent, and audit logging. **Pro plugin.**

> **Requires:** Business license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install compliance
```

## What It Does

Adds a compliance management layer covering GDPR, CCPA, HIPAA, SOC 2, and PCI-DSS requirements. Manages consent records, processes Data Subject Access Requests (DSARs) including data export and right-to-erasure, maintains an immutable compliance audit log, and generates compliance reports. Provides a consent API for cookie banners and privacy settings UIs.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `COMPLIANCE_PORT` | `3211` | Compliance service port |
| `COMPLIANCE_REGULATIONS` | `gdpr,ccpa` | Active regulations |
| `COMPLIANCE_DSAR_RESPONSE_DAYS` | `30` | DSAR response deadline in days |
| `COMPLIANCE_DATA_RETENTION_DAYS` | `2555` | Default data retention (7 years) |
| `COMPLIANCE_PII_DETECTION` | `true` | Detect PII in requests |

## Ports

| Port | Purpose |
|------|---------|
| 3211 | Compliance REST API |

## Database Tables

17 tables added to your Postgres database:
- `np_compliance_consent_records`, user consent decisions
- `np_compliance_consent_versions`, consent version history
- `np_compliance_dsar_requests`, DSAR request queue
- `np_compliance_data_exports`, exported user data packages
- `np_compliance_erasure_requests`, right-to-erasure requests
- `np_compliance_audit_log`, immutable compliance events
- `np_compliance_retention_policies`, data retention rules
- And 10 more for regulations, reports, violations, etc.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/compliance/` | Compliance management API |
| `/compliance/consent` | Consent collection endpoint |
| `/compliance/dsar` | DSAR submission endpoint |

## Distinct From the `hipaa` Plugin

This plugin handles multi-regulation consent and DSAR workflows across GDPR, CCPA, HIPAA, SOC 2, and PCI. The separate `hipaa` plugin is narrower: it focuses on the PHI column registry and protected-health-information handling. Install `compliance` for cross-regulation consent, retention, and DSAR management; install `hipaa` when you specifically need PHI column-level tracking.

## See Also

- [[plugin-hipaa]], PHI column registry
- [[Plugin-Overview]]
- [[Home]]
