# nself gdpr

<!-- BEGIN PROSE:summary -->
> GDPR data portability and right-to-erasure (Art. 20, 17).
<!-- END PROSE:summary -->

## Synopsis

```
nself gdpr <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Manage GDPR data portability (Art. 20) and right-to-erasure (Art. 17) requests for your ɳSelf instance.

All operations write an entry to `np_gdpr_requests`, which is the append-only audit trail required by GDPR Art. 30. That table is never deleted.

---

### nself gdpr export
Export all personal data held for a user as a ZIP archive.
```bash
nself gdpr export --user <user_id> [--format json|csv] [--output <path>] [--dry-run]
```
The archive contains one file per plugin/table. Each file lists the rows belonging to the user.
---
### nself gdpr delete
Delete or anonymize all data for a user across every plugin-registered table and core ɳSelf tables.
```bash
nself gdpr delete --user <user_id> [--dry-run]
```
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
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `delete` | Delete or anonymize all data for a user (GDPR Art. 17 erasure) |
| `export` | Export all data for a user (GDPR Art. 20 portability) |
| `forget` | Alias for 'delete' — right to be forgotten (GDPR Art. 17) |
| `list-requests` | List all GDPR requests |
| `status` | Show the status of a GDPR request |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself gdpr
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
