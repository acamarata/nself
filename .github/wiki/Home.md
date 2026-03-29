# ɳSelf

> Self-hosted backend in five minutes. Postgres, GraphQL, Auth, Nginx. No cloud required.

nSelf is an open-source CLI that spins up a production-grade backend stack on any server or local machine. PostgreSQL, Hasura GraphQL, Auth, and Nginx reverse-proxy out of the box. Fully self-hosted, MIT licensed, no vendor lock-in.

```bash
brew install nself-org/nself/nself
nself init myapp
nself start
```

**Three commands. Complete backend. Your infrastructure.**

## What's included

| Layer | Service | Details |
|-------|---------|---------|
| Database | PostgreSQL | pgvector, PostGIS, TimescaleDB extensions |
| API | Hasura GraphQL | Instant API with permissions, subscriptions, remote schemas |
| Auth | nHost Auth | JWT, 13 OAuth providers, MFA, magic links |
| Proxy | Nginx | Automatic SSL, rate limiting, security headers |
| Storage | MinIO | S3-compatible object storage (optional) |
| Plugins | 84 total | 25 free + 59 Pro — AI, video, commerce, CMS, and more |

## Start here

- [[Installation]] — Homebrew, curl, or manual install
- [[Quick-Start]] — Running in under 5 minutes
- [[First-Project]] — Build a real backend step by step
- [[Commands]] — All 295+ commands
- [[FAQ]] — Common questions

## Go deeper

- [[Plugin-Overview]] — 84 plugins for every use case
- [[Guide-Production-Deployment]] — Deploy to a real server
- [[Guide-Security-Hardening]] — Lock it down
- [[Architecture]] — How the pieces fit together
