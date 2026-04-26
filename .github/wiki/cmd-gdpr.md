# nself gdpr

Manage GDPR data portability (Art. 20) and right-to-erasure (Art. 17) requests for your ɳSelf instance.

All operations write an entry to `np_gdpr_requests`, which is the append-only audit trail required by GDPR Art. 30. That table is never deleted.

---

## Subcommands

### nself gdpr export

Export all personal data held for a user as a ZIP archive.

```bash
nself gdpr export --user <user_id> [--format json|csv] [--output <path>] [--dry-run]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--user` | required | User ID to export |
| `--format` | `json` | Archive format: `json` or `csv` |
| `--output` | `gdpr-export-<id>.zip` | Destination path for the archive |
| `--dry-run` | `false` | Print what would be exported without generating an archive |
| `--notify` | — | Email address to notify on completion |

The archive contains one file per plugin/table. Each file lists the rows belonging to the user.

---

### nself gdpr delete

Delete or anonymize all data for a user across every plugin-registered table and core ɳSelf tables.

```bash
nself gdpr delete --user <user_id> [--dry-run]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--user` | required | User ID to erase |
| `--dry-run` | `false` | Show affected row counts without deleting |

Tables configured with strategy `delete` have rows removed. Tables configured with strategy `anonymize` have PII columns replaced with pseudonymous values (`gdpr-erased-<prefix>`, `deleted@gdpr.invalid`, `Deleted User`).

---

### nself gdpr status

Check the status of a specific GDPR request.

```bash
nself gdpr status --request <request_id>
```

---

### nself gdpr list-requests

List all GDPR requests, optionally filtered by status.

```bash
nself gdpr list-requests [--status pending|processing|complete|failed]
```

---

## Plugin registry

Third-party plugins register their tables by calling `POST /gdpr/registry` on the gdpr plugin service, or by implementing the `GDPRProvider` Go interface. Registered tables are automatically included in export and delete cascades.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_GDPR_EXPORT_BUCKET` | `gdpr-exports` | MinIO bucket for export archives |
| `NSELF_GDPR_EXPORT_TTL` | `604800` | Presigned URL TTL in seconds (7 days) |
| `NSELF_GDPR_DEADLINE_DAYS` | `30` | Response deadline (never increase past 30) |
| `NSELF_GDPR_DEADLINE_ENFORCE` | `true` | Warn at T-7d, fail at T+0 |
| `NSELF_GDPR_NOTIFY_EMAIL` | — | Optional completion notification |
| `NSELF_GDPR_TENANT_DELETE` | `false` | Enable full tenant-level purge (Enterprise) |

---

## Related

- [[cmd-security]] - Security audit and hardening
- [[Home]]
