# Support Plugin

> Helpdesk ticketing with SLA management, canned responses, and team routing. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install support
```

## What It Does

A complete helpdesk ticketing system for managing customer support. Users submit tickets via API or email, tickets are routed to teams based on rules, SLA timers track response and resolution deadlines, and agents reply using canned responses. Includes escalation rules, internal notes, and customer satisfaction (CSAT) surveys.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SUPPORT_PORT` | `3111` | Support service port |
| `SUPPORT_DEFAULT_SLA_HOURS` | `24` | Default first response SLA |
| `SUPPORT_EMAIL_ENABLED` | `false` | Enable email ticket submission |
| `SUPPORT_CSAT_ENABLED` | `true` | Send CSAT surveys on close |

## Ports

| Port | Purpose |
|------|---------|
| 3111 | Support REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_support_tickets` — ticket records
- `np_support_replies` — ticket replies
- `np_support_teams` — support team definitions
- `np_support_routing_rules` — ticket routing rules
- `np_support_canned_responses` — reusable reply templates
- `np_support_sla_policies` — SLA policy definitions
- `np_support_escalations` — escalation records
- `np_support_csat` — customer satisfaction responses
- `np_support_notes` — internal agent notes

## Nginx Routes

| Route | Target |
|-------|--------|
| `/support/` | Support ticket API |
