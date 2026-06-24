# Compliance Plugin

> GDPR, CCPA, HIPAA, SOC 2, and PCI compliance, DSARs, consent, and audit logging. **Pro plugin — requires ɳSelf+ license.**

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install compliance
nself build
```

Requires an active ɳSelf+ subscription ($3.99/mo or $39.99/yr). The license key is validated against `ping.nself.org/license/validate` before download.

## What It Does

Adds a multi-regulation compliance management layer. Toggle GDPR, CCPA, HIPAA, SOC 2, and PCI-DSS per project. Once active, the plugin manages consent records, processes Data Subject Access Requests (DSARs), enforces data retention schedules, sends breach notifications, and maintains an immutable audit log with optional SIEM forwarding.

This plugin is distinct from the `hipaa` plugin. The `hipaa` plugin focuses on PHI column-level encryption, the 6-year PHI access log, de-identification helpers, and Business Associate Agreement (BAA) workflows. The `compliance` plugin handles the broader GDPR/CCPA consent and DSAR lifecycle, multi-framework retention schedules, and cross-regulation audit logging.

## GDPR DSAR Workflow

1. A data subject submits a request via `POST /compliance/dsar` (type: `access`, `erasure`, or `portability`).
2. The plugin creates a record in `np_compliance_dsar_requests` with a deadline computed from `COMPLIANCE_DSAR_RESPONSE_DAYS` (default: 30 days, per GDPR Article 12).
3. For `access` and `portability` requests, the plugin assembles a data export package stored in `np_compliance_data_exports` and returns a signed download URL.
4. For `erasure` (right-to-be-forgotten) requests, an erasure record is created in `np_compliance_erasure_requests`. Your app queries this table before serving data.
5. All DSAR activity is written to `np_compliance_audit_log` for regulatory evidence.

## CCPA Opt-Out

Enable CCPA with `COMPLIANCE_CCPA_ENABLED=true`. The consent API at `/compliance/consent` accepts a `ccpa_do_not_sell` flag. Consent decisions are stored per subject in `np_compliance_consent_records` with version history in `np_compliance_consent_versions`. Query the consent API before any data sale or sharing operation.

## Consent Records

Consent is captured per subject and purpose:

```bash
curl -X POST https://api.example.com/compliance/consent \
  -H 'Authorization: Bearer $TOKEN' \
  -d '{"subject_id":"u_xxx","purpose":"marketing","granted":true,"regulation":"gdpr"}'
```

Records include timestamp, version, and source IP. The consent API returns the current consent state for a given subject and purpose, making it suitable for server-side gating.

## Data Retention Schedules

Retention policies are stored in `np_compliance_retention_policies`. Each policy specifies:

- `table_name`: which `np_*` table to age out
- `retention_days`: how long to keep records
- `regulation`: the driving framework (gdpr, ccpa, hipaa, soc2, pci)

The plugin evaluates policies on a schedule and writes results to `np_compliance_retention_executions`. Trigger a manual run:

```bash
curl -X POST https://api.example.com/compliance/retention/run \
  -H 'Authorization: Bearer $TOKEN' \
  -d '{"policy_id":"ret_xxx"}'
```

Default retention is 7 years (`COMPLIANCE_DATA_RETENTION_DAYS=2555`), suitable for SOC 2 and PCI audit requirements.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `COMPLIANCE_PLUGIN_PORT` | `3211` | Compliance service port |
| `COMPLIANCE_GDPR_ENABLED` | — | Enable GDPR framework |
| `COMPLIANCE_CCPA_ENABLED` | — | Enable CCPA / opt-out tracking |
| `COMPLIANCE_HIPAA_ENABLED` | — | Enable HIPAA-adjacent audit (use `hipaa` plugin for PHI) |
| `COMPLIANCE_SOC2_ENABLED` | — | Enable SOC 2 audit log tagging |
| `COMPLIANCE_PCI_ENABLED` | — | Enable PCI-DSS retention rules |
| `COMPLIANCE_DSAR_DEADLINE_DAYS` | `30` | DSAR response deadline in days |
| `COMPLIANCE_DATA_RETENTION_DAYS` | `2555` | Default data retention (7 years) |
| `COMPLIANCE_CONSENT_EXPIRY_DAYS` | — | Auto-expire consent records after N days |
| `COMPLIANCE_BREACH_NOTIFICATION_HOURS` | — | Breach notification deadline in hours |
| `COMPLIANCE_AUDIT_RETENTION_DAYS` | — | Audit log retention override |
| `AUDIT_SIEM_SPLUNK_HEC_URL` | — | Splunk HEC endpoint for SIEM forwarding |
| `AUDIT_SIEM_ELK_URL` | — | Elasticsearch endpoint |
| `AUDIT_SIEM_DATADOG_API_KEY` | — | Datadog API key |

Reference `~/.claude/vault.env` for secrets. Never hardcode credentials.

## Ports

| Port | Purpose |
|------|---------|
| 3211 | Compliance REST API (bound to 127.0.0.1, reached via Nginx) |

## Database Tables

17 tables added to your Postgres database (prefix `np_`, all with `source_account_id` for multi-app isolation):

- `np_compliance_dsars`, DSAR request queue
- `np_compliance_dsar_activities`, DSAR activity log
- `np_compliance_consents`, current consent decisions
- `np_compliance_consent_history`, consent version history
- `np_compliance_privacy_policies`, policy versions
- `np_compliance_policy_acceptances`, user policy acceptance records
- `np_compliance_retention_policies`, retention rules per table
- `np_compliance_retention_executions`, retention run results
- `np_compliance_processing_records`, lawful basis records
- `np_compliance_data_processors`, third-party processor registry
- `np_compliance_data_breaches`, breach incident records
- `np_compliance_breach_notifications`, notification dispatch log
- `np_compliance_audit_log`, immutable compliance event log
- `np_compliance_audit_events`, structured audit events
- `np_compliance_audit_retention_policies`, audit log retention config
- `np_compliance_audit_alert_rules`, alert trigger rules
- `np_compliance_audit_webhook_events`, outbound webhook deliveries

## Nginx Routes

| Route | Target |
|-------|--------|
| `/compliance/` | Compliance management API |
| `/compliance/consent` | Consent collection and query |
| `/compliance/dsar` | DSAR submission endpoint |
| `/compliance/retention/run` | Manual retention policy trigger |

## Docker Hub

```bash
docker pull nself/plugin-compliance:latest
```

Image: [`nself/plugin-compliance`](https://hub.docker.com/r/nself/plugin-compliance)

## See Also

- [[plugin-hipaa]] — PHI column registry and HIPAA-specific controls
- [[Commands]] — full CLI command reference
- [[Home]]
