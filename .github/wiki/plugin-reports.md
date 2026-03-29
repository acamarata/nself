> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Reports Plugin

> Automated report generation — scheduled queries to PDF/CSV/email delivery. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install reports
```

## What It Does

Defines and schedules reports that run SQL queries against your Postgres database, formats results as PDF, CSV, or Excel, and delivers them via email on a schedule. Supports parameterized queries, multiple recipients, and custom report templates. Results are archived in Postgres for historical access.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `REPORTS_PORT` | `3048` | Reports service port |
| `REPORTS_SMTP_HOST` | — | SMTP host for email delivery |
| `REPORTS_FROM_EMAIL` | — | Sender email address |
| `REPORTS_STORAGE_DIR` | `/reports` | Local report archive directory |
| `REPORTS_MAX_ROWS` | `10000` | Max rows per report |

## Ports

| Port | Purpose |
|------|---------|
| 3048 | Reports REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_reports_definitions` — report definitions and SQL queries
- `np_reports_schedules` — delivery schedules
- `np_reports_archive` — generated report history

## Nginx Routes

| Route | Target |
|-------|--------|
| `/reports/` | Reports management API |
