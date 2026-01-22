# nself Planning Documentation

Detailed planning documents for each upcoming release.

---

## Current Release

| Version | Status | Focus |
|---------|--------|-------|
| **v0.4.2** | Released | Service & Monitoring management |

---

## Upcoming Releases

| Version | Plan Document | Focus | Target |
|---------|---------------|-------|--------|
| **v0.4.3** | [v0.4.3-PLAN.md](v0.4.3-PLAN.md) | Deployment Pipeline (local → staging → prod) | Q1 2026 |
| **v0.4.4** | [v0.4.4-PLAN.md](v0.4.4-PLAN.md) | Database, Backup & Restore | Q1-Q2 2026 |
| **v0.4.5** | [v0.4.5-PLAN.md](v0.4.5-PLAN.md) | Mock Data & Seeding System | Q2 2026 |
| **v0.4.6** | [v0.4.6-PLAN.md](v0.4.6-PLAN.md) | Scaling & Performance | Q2-Q3 2026 |
| **v0.4.7** | [v0.4.7-PLAN.md](v0.4.7-PLAN.md) | Multi-Cloud Providers | Q3 2026 |
| **v0.4.8** | [v0.4.8-PLAN.md](v0.4.8-PLAN.md) | Kubernetes Support | Q3-Q4 2026 |
| **v0.4.9** | [v0.4.9-PLAN.md](v0.4.9-PLAN.md) | Polish & nself-admin Integration | Q4 2026 |
| **v0.5.0** | [v0.5.0-PLAN.md](v0.5.0-PLAN.md) | Full Production Release + nself-admin v0.1 | Q4 2026 / Q1 2027 |

---

## Quick Reference

### New Commands by Version

```
v0.4.3: env, deploy, prod, staging
v0.4.4: db, backup, restore
v0.4.5: seed, mock, data
v0.4.6: scale, perf, migrate, bench
v0.4.7: cloud, provision
v0.4.8: k8s, helm

Total new commands: 18
Total commands at v0.5.0: 46
```

### Key Notes

#### v0.4.3 - Deployment Access Control
- SSH key auto-detection for server access
- Not everyone may have access to staging/prod
- Support for username/password credentials
- Direct VPS/IP address deployment supported

#### v0.4.5 - Environment-Aware Mock Data
- **Local**: Full mock data, reset on demand
- **Staging**: Mock data with realistic volumes
- **Production**: Real data only, no mocks ever
- Automatic detection - no configuration needed

#### v0.4.7 - Supported Cloud Providers
- AWS (EC2, RDS, S3, EKS)
- Google Cloud (Compute, Cloud SQL, GKE)
- Azure (VMs, Azure DB, AKS)
- DigitalOcean (Droplets, Managed DB, DOKS)
- Linode (Linodes, Managed DB, LKE)
- Vultr (Compute, Managed DB)
- Hetzner (Cloud Servers, Volumes)

#### v0.5.0 - Release Includes
- nself CLI v0.5.0 (all v0.4.x features polished)
- nself-admin v0.1.0 (first UI release)

---

## Dependencies Between Versions

```
v0.4.2 (current)
    ↓
v0.4.3 (environments, deployment)
    ↓
v0.4.4 (database uses environments)
    ↓
v0.4.5 (mock data uses environments + db)
    ↓
v0.4.6 (scaling builds on deployment)
    ↓
v0.4.7 (cloud uses deployment foundation)
    ↓
v0.4.8 (k8s extends cloud provisioning)
    ↓
v0.4.9 (polish all features)
    ↓
v0.5.0 (production release)
```

---

## Contributing to Planning

To suggest changes to the roadmap:

1. Open a GitHub Discussion
2. Label it as "roadmap-feedback"
3. Describe your use case
4. Propose specific features or changes

All feedback is reviewed and incorporated where appropriate.

---

*Last Updated: January 22, 2026*
