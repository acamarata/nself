# Security Hardening Checklist

Use this checklist before going live in production. For step-by-step instructions, see [[Guide-Security-Hardening]].

## Pre-Launch Checklist

### Firewall
- [ ] Firewall enabled (`ufw status` shows active)
- [ ] Only ports 22, 80, 443 open inbound
- [ ] Port 5432 (Postgres) blocked from external access
- [ ] Port 8080 (Hasura) blocked from external access
- [ ] Port 4000 (Auth) blocked from external access

### Secrets
- [ ] `POSTGRES_PASSWORD` changed from default (32+ char random)
- [ ] `HASURA_GRAPHQL_ADMIN_SECRET` changed from default (32+ char random)
- [ ] `HASURA_JWT_KEY` changed from default (32+ char random)
- [ ] `AUTH_JWT_SECRET` set (32+ char random)
- [ ] `.env.secrets` added to `.gitignore`
- [ ] `.env.secrets` not committed to git (`git log --all -- .env.secrets` shows nothing)

### Hasura
- [ ] `HASURA_GRAPHQL_ENABLE_CONSOLE=false` in `.env.prod`
- [ ] `HASURA_GRAPHQL_DEV_MODE=false` in `.env.prod`

### TLS
- [ ] Valid TLS certificate installed (not self-signed in production)
- [ ] Certificate expiry date noted (`nself ssl status`)
- [ ] Calendar reminder set for certificate renewal

### Monitoring
- [ ] Monitoring plugin installed (`nself plugin install monitoring`)
- [ ] Grafana admin password changed from default
- [ ] Alert rules configured in Alertmanager

### Backups
- [ ] Backup plugin installed or manual backup scheduled
- [ ] Backup restoration tested on a staging instance
- [ ] Backup destination (local or S3) verified

### Updates
- [ ] ɳSelf CLI at latest version (`nself update`)
- [ ] Schedule for regular updates established

## Ongoing Maintenance

| Task | Frequency | Command |
|------|-----------|---------|
| Update CLI | Weekly | `nself update` |
| Check service health | Daily | `nself health` |
| Review logs for errors | Weekly | `nself logs` |
| Verify backup success | Weekly | Check backup logs |
| Review Grafana alerts | Daily | Check Alertmanager |

## Quick Reference

Generate a strong secret:
```bash
openssl rand -hex 32
```

Check current TLS certificate:
```bash
nself ssl status
```

Run the ɳSelf health check:
```bash
nself health
```

## See Also

- [[Guide-Security-Hardening]], step-by-step instructions for each item above
- [[Security-Architecture]], how ɳSelf security is designed
- [[Security-Policy]], reporting vulnerabilities

---
← [[Home]] | [[_Sidebar]]
