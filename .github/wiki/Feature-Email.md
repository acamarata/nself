# Feature: Email

nSelf supports email delivery through built-in SMTP configuration and a growing list of provider integrations.

## What's Included

| Capability | Description |
|-----------|-------------|
| SMTP | Direct SMTP for any mail server |
| Auth email | Magic links, email verification, password reset |
| Mailpit (dev) | Local email capture UI for development |
| 16 provider integrations | Via the email plugin ecosystem |

## Built-in SMTP Configuration

Configure SMTP in `.env`:

```env
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=noreply@example.com
SMTP_SECURE=true
```

## Development Mode (Mailpit)

In development, all emails are captured by Mailpit. No emails leave the server. View captured emails at the Mailpit URL from `nself urls`.

## Provider Integrations

The following providers are supported via plugins:

| Provider | Plugin |
|---------|--------|
| SendGrid | `plugin-sendgrid` (pro) |
| Mailgun | `plugin-mailgun` (pro) |
| Postmark | `plugin-postmark` (pro) |

Install with:
```bash
nself plugin install sendgrid
```

## Key Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SMTP_HOST` | SMTP server hostname | Required |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USER` | SMTP username | Required |
| `SMTP_PASSWORD` | SMTP password | Required |
| `SMTP_FROM` | From address | Required |
| `SMTP_SECURE` | Use TLS | `true` |

## See Also

- [[Config-Env-Vars]] — full env var reference
- [[Feature-Auth]] — how auth uses email for magic links

---
← [[Home]] | [[_Sidebar]]
