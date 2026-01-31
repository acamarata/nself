# Changelog

All notable changes to nself will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.0] - 2026-01-29

### Added - Phase 3: Multi-Tenancy & Enterprise (100%)

Complete multi-tenancy implementation with tenant isolation, resource quotas, billing integration, and enterprise collaboration features. All 8 sprints completed with 280 points.

#### Sprint 21: Multi-Tenancy Foundation (40 points) ✓
**Tenant Management:**
- Tenant CRUD operations with schema isolation
- Automatic schema creation per tenant (tenant_*)
- Tenant metadata and settings
- Tenant status management (active/suspended/trial)
- Tenant provisioning and cleanup
- Complete tenant CLI interface

**Database Migrations:**
- `001_tenant_foundation.sql` - Tenant management tables and schemas
- Schema-based tenant isolation
- Per-tenant user management
- Tenant-specific configuration

**CLI Commands:**
- `nself tenant create` - Create new tenant
- `nself tenant list` - List all tenants
- `nself tenant get` - Get tenant details
- `nself tenant update` - Update tenant settings
- `nself tenant delete` - Delete tenant (soft/hard)
- `nself tenant status` - Change tenant status

**Documentation:**
- Multi-tenancy architecture guide
- Tenant isolation patterns
- Schema design and best practices

#### Sprint 22: Tenant Isolation (40 points) ✓
**Data Isolation:**
- Row-level security (RLS) policies
- Tenant context propagation via JWT claims
- Cross-tenant access prevention
- Secure tenant data boundaries

**Resource Isolation:**
- Per-tenant resource limits
- Storage quota enforcement
- API rate limiting per tenant
- Database connection pooling per tenant

**Database Migrations:**
- `002_tenant_isolation.sql` - RLS policies and security
- Tenant context functions
- Cross-tenant protection

**Security Features:**
- Tenant ID validation middleware
- Context switching protection
- Audit logging per tenant
- Tenant data encryption

**Documentation:**
- Tenant isolation security model
- RLS implementation guide
- Multi-tenant security best practices

#### Sprint 23: Tenant Users & Teams (35 points) ✓
**Tenant User Management:**
- User-tenant associations
- Per-tenant user roles
- User invitation system
- Tenant user permissions
- User transfer between tenants

**Team Management:**
- Team CRUD operations
- Team-based access control
- Team roles and permissions
- Team member management
- Team ownership and delegation

**Database Migrations:**
- `003_tenant_teams.sql` - Team and membership tables
- User-tenant associations
- Team-based RBAC

**CLI Commands:**
- `nself tenant users` - Manage tenant users
- `nself tenant teams` - Team management
- `nself tenant invite` - User invitations
- `nself tenant roles` - Tenant-specific roles

**Documentation:**
- Multi-tenant user management
- Team collaboration patterns
- Invitation workflows

#### Sprint 24: Resource Quotas & Billing (45 points) ✓
**Resource Quotas:**
- Configurable tenant quotas
- Real-time usage tracking
- Quota enforcement
- Quota alerts and notifications
- Usage analytics per tenant

**Billing Integration:**
- Stripe integration for subscriptions
- Plan management (free/basic/pro/enterprise)
- Usage-based billing
- Invoice generation
- Payment method management
- Billing history and reports

**Database Migrations:**
- `004_tenant_quotas_billing.sql` - Quotas and billing tables
- Usage tracking tables
- Subscription management
- Invoice storage

**Quota Types:**
- Storage limits
- User count limits
- API request limits
- Custom service limits
- Bandwidth limits

**CLI Commands:**
- `nself tenant quota` - Manage tenant quotas
- `nself tenant usage` - View usage statistics
- `nself tenant billing` - Billing management
- `nself tenant plans` - Subscription plans
- `nself tenant invoice` - Invoice operations

**Documentation:**
- Resource quota configuration
- Billing integration guide
- Stripe setup and testing
- Usage tracking implementation

#### Sprint 25: Tenant Analytics & Monitoring (35 points) ✓
**Analytics System:**
- Per-tenant analytics tracking
- User activity metrics
- Resource usage analytics
- Custom event tracking
- Analytics data retention policies

**Monitoring Dashboards:**
- Tenant health monitoring
- Real-time metrics per tenant
- Performance monitoring
- Error tracking
- Alert system

**Database Migrations:**
- `005_tenant_analytics.sql` - Analytics tables
- Event tracking schema
- Metrics storage

**Metrics Tracked:**
- User login/logout events
- API usage patterns
- Resource consumption
- Feature adoption
- Error rates

**CLI Commands:**
- `nself tenant analytics` - View analytics
- `nself tenant metrics` - Access metrics
- `nself tenant events` - Event tracking
- `nself tenant health` - Health checks

**Prometheus Integration:**
- Per-tenant metrics export
- Custom metric definitions
- Grafana dashboard templates

**Documentation:**
- Analytics implementation guide
- Monitoring setup
- Custom metrics creation

#### Sprint 26: Tenant Backup & Restore (30 points) ✓
**Backup System:**
- Automated tenant backups
- Point-in-time recovery
- Backup scheduling (hourly/daily/weekly)
- Backup retention policies
- Cross-region backup storage

**Restore Operations:**
- Full tenant restore
- Selective data restore
- Restore to new tenant
- Restore validation
- Rollback capabilities

**Database Migrations:**
- `006_tenant_backups.sql` - Backup metadata tables
- Restore tracking
- Backup verification

**Backup Types:**
- Full database backup
- Schema-only backup
- Data-only backup
- Incremental backups

**CLI Commands:**
- `nself tenant backup create` - Create backup
- `nself tenant backup list` - List backups
- `nself tenant backup restore` - Restore from backup
- `nself tenant backup schedule` - Configure auto-backup
- `nself tenant backup verify` - Verify backup integrity

**Storage Options:**
- Local filesystem
- S3-compatible storage (MinIO)
- Azure Blob Storage
- Google Cloud Storage

**Documentation:**
- Backup and restore guide
- Disaster recovery procedures
- Backup best practices

#### Sprint 27: Tenant Migration Tools (35 points) ✓
**Migration System:**
- Tenant data export
- Tenant data import
- Cross-environment migration (dev→staging→prod)
- Tenant cloning
- Data transformation pipelines

**Export Formats:**
- JSON export
- CSV export
- SQL dump
- Custom format support

**Import Features:**
- Schema validation
- Data mapping
- Conflict resolution
- Dry-run mode
- Import rollback

**Database Migrations:**
- `007_tenant_migrations.sql` - Migration tracking tables
- Migration history
- Migration validation

**CLI Commands:**
- `nself tenant export` - Export tenant data
- `nself tenant import` - Import tenant data
- `nself tenant clone` - Clone tenant
- `nself tenant migrate` - Cross-env migration
- `nself tenant validate` - Validate migration data

**Migration Features:**
- Zero-downtime migrations
- Data consistency checks
- Foreign key preservation
- Index rebuilding
- Migration progress tracking

**Documentation:**
- Migration guide
- Data transformation examples
- Troubleshooting migrations

#### Sprint 28: Enterprise Collaboration (40 points) ✓
**Collaboration Features:**
- Real-time notifications
- Activity feeds per tenant
- Comment system
- @mentions and tagging
- Notification preferences

**Workspace Management:**
- Multiple workspaces per tenant
- Workspace permissions
- Cross-workspace collaboration
- Workspace templates

**Communication:**
- In-app messaging
- Notification channels (email, SMS, webhook)
- Activity streams
- Read receipts
- Notification batching

**Database Migrations:**
- `008_tenant_collaboration.sql` - Collaboration tables
- Notification system
- Workspace management

**Features:**
- User presence indicators
- Typing indicators
- Activity timestamps
- Notification badges
- Email digests

**CLI Commands:**
- `nself tenant notify` - Send notifications
- `nself tenant activity` - View activity feed
- `nself tenant workspace` - Workspace management
- `nself tenant messages` - Messaging operations

**Integration:**
- Slack notifications
- Discord webhooks
- Email templates
- SMS via Twilio
- Push notifications

**Documentation:**
- Collaboration features guide
- Notification system setup
- Workspace management best practices

### Technical Improvements
- Complete schema isolation per tenant
- Row-level security enforcement
- Per-tenant connection pooling
- Tenant context propagation
- Resource quota enforcement
- Real-time analytics processing
- Automated backup system
- Zero-downtime migrations
- Enterprise-grade collaboration

### Security
- Tenant isolation via RLS policies
- Cross-tenant access prevention
- Encrypted tenant data at rest
- Per-tenant audit logging
- Secure tenant provisioning
- Tenant data deletion compliance (GDPR)
- API key scoping per tenant

### Database Schema
**tenants schema:**
- tenants (core tenant management)
- tenant_users (user-tenant associations)
- tenant_teams (team management)
- tenant_invitations (user invitations)
- tenant_quotas (resource limits)
- tenant_usage (usage tracking)
- tenant_subscriptions (billing)
- tenant_invoices (invoice history)
- tenant_analytics_events (event tracking)
- tenant_metrics (metrics storage)
- tenant_backups (backup metadata)
- tenant_migrations (migration tracking)
- tenant_notifications (notification system)
- tenant_workspaces (workspace management)
- tenant_activity (activity feeds)

### CLI Commands Added
- `nself tenant` - Complete tenant management suite
- `nself tenant users` - Tenant user management
- `nself tenant teams` - Team collaboration
- `nself tenant quota` - Resource quotas
- `nself tenant billing` - Billing operations
- `nself tenant analytics` - Analytics and metrics
- `nself tenant backup` - Backup and restore
- `nself tenant migrate` - Migration tools
- `nself tenant notify` - Notification system
- `nself tenant workspace` - Workspace management

### Performance
- Optimized per-tenant queries
- Connection pooling per tenant
- Analytics data aggregation
- Efficient backup strategies
- Indexed tenant lookups
- Cached quota checks
- Async notification delivery

### Documentation
- Multi-tenancy architecture guide: `/docs/architecture/multi-tenancy.md`
- Tenant isolation security model: `/docs/security/tenant-isolation.md`
- Resource quota configuration: `/docs/configuration/quotas.md`
- Billing integration guide: `/docs/integrations/billing.md`
- Backup and restore procedures: `/docs/operations/backup-restore.md`
- Migration guide: `/docs/operations/migrations.md`
- Collaboration features: `/docs/features/collaboration.md`

### Statistics - Phase 3
- **Total Points Completed:** 280/280 (100%)
- **Database Migrations:** 8 migrations
- **New CLI Commands:** 10+ command groups
- **Database Tables Added:** 15+ tables
- **Lines of Code:** ~8,000 lines
- **Git Commits:** 40+ commits

## [0.7.0] - 2026-01-29

### Added - Phase 2: Advanced Backend Features (100%)

Complete advanced backend implementation with real-time collaboration, performance optimization, developer tools, migration utilities, and enhanced security. All sprints completed with 270 points.

#### Sprint 16: Real-Time Collaboration (70 points) ✓
- WebSocket server implementation
- Real-time presence tracking
- Live cursor sharing
- Collaborative editing
- Pub/sub messaging
- Room management
- Complete websocket CLI

#### Sprint 17: Security Hardening (25 points) ✓
- Security audit checklist
- Firewall configuration
- SSL/TLS automation with Let's Encrypt
- Secrets scanning
- Vulnerability management
- Security headers configuration
- Compliance reporting

#### Sprint 18: Performance & Optimization (45 points) ✓
- Query optimization tools
- Database indexing strategies
- Caching layer (Redis)
- CDN integration
- Load testing framework
- Performance monitoring
- Bottleneck analysis

#### Sprint 19: Developer Tools (30 points) ✓
- API documentation generator
- Schema introspection
- GraphQL playground integration
- Development environment setup
- Debugging utilities
- Testing frameworks

#### Sprint 20: Migration Tools (40 points) ✓
- Database migration system
- Schema versioning
- Data transformation utilities
- Rollback capabilities
- Migration validation
- Import/export tools

### Database Migrations
- Real-time collaboration schema
- Security audit tables
- Performance metrics tables
- Migration tracking system

### CLI Commands Added
- `nself websocket` - WebSocket management
- `nself security` - Security operations
- `nself performance` - Performance tools
- `nself dev` - Developer utilities
- `nself migrate` - Migration management

### Documentation
- Real-time collaboration guide
- Security hardening checklist
- Performance optimization guide
- Developer tools documentation
- Migration system guide

## [0.6.0] - 2026-01-29

### Added - Phase 1: Enterprise Authentication & Security (91.5%)

#### Sprint 1: Core Authentication (100%) ✓
- Password authentication with bcrypt hashing
- Email/password signup and login flows
- Password reset with token expiration
- Email verification system
- Account linking (multiple auth methods per user)
- Secure session management
- CLI commands: signup, login, verify, reset

#### Sprint 2: OAuth & MFA (100%) ✓
**OAuth Providers (14 total):**
- Google OAuth 2.0
- GitHub OAuth 2.0
- Facebook OAuth 2.0
- Discord OAuth 2.0
- Microsoft Azure AD OAuth 2.0
- LinkedIn OAuth 2.0
- Slack OAuth v2
- Twitch OAuth 2.0
- Custom OIDC provider with auto-discovery
- Apple Sign In with JWT client secret
- Twitter/X OAuth 2.0 with PKCE
- GitLab OAuth 2.0 (supports self-hosted instances)
- Bitbucket OAuth 2.0

**MFA Methods:**
- TOTP (Time-based One-Time Password) with QR code generation
- SMS MFA (Twilio, AWS SNS, dev mode)
- Email MFA with customizable templates
- Backup codes (10 one-time recovery codes)
- MFA policies (global, role-based, user exemptions)
- WebAuthn/FIDO2 support (YubiKey, TouchID, Windows Hello)
- Complete MFA CLI interface

**User Management:**
- User CRUD operations with soft delete
- User profiles (avatar, bio, custom fields)
- User import/export (JSON, CSV formats)
- User metadata with versioning and history
- User search and filtering

#### Sprint 3: RBAC & Auth Hooks (81.5%)
**Role-Based Access Control:**
- Role CRUD operations (create, update, delete)
- System vs custom roles
- Default role management
- User-role assignments and revocation
- Comprehensive role CLI with permissions management

**Permission Management:**
- Fine-grained permissions (resource:action format)
- Role-permission associations
- User permission aggregation from multiple roles
- Permission checking and validation
- Permission inheritance

**Auth Hooks System:**
- Pre/post signup hooks
- Pre/post login hooks
- Custom claims generation hooks
- Pre/post MFA hooks
- Priority-based hook execution
- Hook logging and audit trail
- Pluggable architecture for custom logic

**JWT Management:**
- JWT configuration (algorithm, TTL, issuer)
- RS256 key pair generation with OpenSSL
- Automatic key rotation (configurable interval)
- Multiple keys support for gradual rotation
- Key storage in PostgreSQL

**Session Management:**
- Session lifecycle management
- Refresh token rotation for enhanced security
- Session revocation (single, all, all-except-current)
- Last activity tracking
- Automatic cleanup of expired sessions
- Session listing per user

**Custom Claims:**
- Generate custom JWT claims from user roles/permissions
- Hasura-compatible JWT claims format
- Claims caching (5-minute TTL for performance)
- Claims validation and refresh

#### Sprint 4: API Keys & Secrets Vault (100%) ✓
**API Key Management:**
- Secure API key generation with prefix support
- SHA-256 hashing for key storage
- Scope-based permissions (resource:action format)
- Key expiration and automatic rotation
- Usage tracking (request count + last used timestamp)
- Keys shown only once on creation for security
- Key revocation and management

**Encrypted Secrets Vault:**
- AES-256-CBC encryption with OpenSSL
- Encryption key generation and rotation (90-day default)
- Encrypted secret storage in PostgreSQL
- Secret versioning with full history
- Rollback to previous versions
- Comprehensive audit trail for compliance
- Environment separation (default/dev/staging/prod)
- Secret sync and promotion workflows (dev→staging→prod)
- Suspicious activity detection
- Secrets comparison across environments
- Complete vault CLI interface

#### Sprint 5: Rate Limiting & Throttling (72.6%)
**Rate Limiting Algorithms:**
- Token bucket (allows bursts, flexible)
- Leaky bucket (smooth rate, no bursts)
- Fixed window (simple, fast)
- Sliding window (accurate, fair)
- Sliding log (most accurate, storage intensive)
- Adaptive rate limiting (adjusts based on success rate)
- Burst protection (detects and blocks traffic spikes)

**Rate Limiting Types:**
- IP-based rate limiting with whitelist/blocklist
- User-based rate limiting with tier support
- Endpoint-based rate limiting with regex rules engine
- Combined limiting (IP+endpoint, user+endpoint)
- Method-based limits (GET/POST/PUT/DELETE)

**Rate Limit Management:**
- IP whitelist and blocklist
- Rule-based endpoint rate limits with priority
- User quota management
- Tier-based limits (free/basic/pro/enterprise/unlimited)
- Rate limit statistics and monitoring
- Comprehensive audit logging
- Rate limit headers (X-RateLimit-*)
- Complete rate-limit CLI interface

### Added - Phase 2: Advanced Features

#### Webhook System
- Webhook endpoint management (create, list, delete)
- Event subscriptions (11 core events)
- HMAC signature verification (sha256)
- Async webhook delivery with retries (3 attempts, 60s delay)
- Delivery status tracking
- Custom headers support
- Complete webhooks CLI

**Webhook Events:**
- user.created, user.updated, user.deleted
- user.login, user.logout
- session.created, session.revoked
- mfa.enabled, mfa.disabled
- role.assigned, role.revoked

#### Device Management
- Device registration and tracking
- Device fingerprinting
- OS, browser, IP detection
- Trusted device management (skip MFA on trusted devices)
- Last seen tracking
- Device revocation
- Device CLI
- Multi-device session support

#### Audit Logging
- Comprehensive event tracking for all auth actions
- Actor, resource, and action tracking
- Result tracking (success/failure)
- Metadata support (JSON)
- IP address and user agent tracking
- Queryable audit trail with filters
- Audit CLI for investigation
- Compliance ready (SOC 2, ISO 27001, GDPR)

### Technical Improvements
- Cross-platform compatibility (Bash 3.2+, macOS/Linux)
- PostgreSQL-backed with proper schema design
- Modular architecture with exported functions
- Comprehensive CLI tooling (10+ commands)
- Security-first approach (bcrypt, SHA-256, AES-256-CBC)
- Docker-based deployment
- Clean error handling and validation
- Extensive documentation

### Security
- OWASP Top 10 mitigations
- CSRF protection
- SQL injection prevention (parameterized queries)
- XSS mitigation
- Secure password storage (bcrypt with salt)
- Encrypted secrets at rest (AES-256-CBC)
- Rate limiting against brute force and DoS
- Audit logging for compliance and forensics
- Refresh token rotation
- JWT key rotation
- Session security best practices

### Database Schema
**auth schema:**
- users (with soft delete)
- sessions
- mfa_secrets, mfa_codes
- roles, permissions, user_roles, role_permissions
- oauth_accounts, email_verifications
- jwt_keys, jwt_config
- password_reset_tokens
- user_metadata, user_metadata_history
- webauthn_credentials, webauthn_challenges
- devices

**secrets schema:**
- encryption_keys
- vault, vault_versions
- audit_log

**rate_limit schema:**
- buckets
- rules
- log
- whitelist, blocklist
- user_quotas

**webhooks schema:**
- endpoints
- deliveries

**audit schema:**
- events

### CLI Commands
- `nself auth` - Authentication management
- `nself mfa` - Multi-factor authentication
- `nself roles` - Role and permission management
- `nself vault` - Encrypted secrets management
- `nself rate-limit` - Rate limiting configuration
- `nself webhooks` - Webhook management
- `nself devices` - Device management
- `nself audit` - Audit log queries
- `nself secrets` - Production secrets generation (separate from vault)

### Performance
- Token bucket algorithm O(1) time complexity
- PostgreSQL indexing on all critical queries
- Claims caching for JWT generation
- Async webhook delivery (non-blocking)
- Connection pooling ready

### Documentation
- Comprehensive README
- API documentation
- CLI usage guides
- Security best practices
- Deployment guides
- Integration examples
- Phase 1 progress summary

## Statistics

- **Total Files Created:** 60+ files
- **Lines of Code:** ~14,000 lines
- **CLI Commands:** 10 commands
- **OAuth Providers:** 14 providers
- **MFA Methods:** 6 methods
- **Rate Limit Strategies:** 7 algorithms
- **Webhook Events:** 11 events
- **Database Tables:** 35+ tables
- **Git Commits:** 120+ commits

## Completion Status

### Phase 1: 91.5% (269/294 points)
- Sprint 1: 100% ✓
- Sprint 2: 100% ✓
- Sprint 3: 81.5%
- Sprint 4: 100% ✓
- Sprint 5: 72.6%

### Phase 2: Started
- Webhook System ✓
- Device Management ✓
- Audit Logging ✓

## Next Release (v0.7.0)

### Planned
- Complete Sprint 3 tests
- Distributed rate limiting with Redis
- Admin dashboard API
- Email templates system
- Analytics and metrics
- More documentation
- Integration guides

### Under Consideration
- SAML 2.0 support
- LDAP/Active Directory integration
- Advanced monitoring integrations
- Developer SDKs (JS, Python, Go)
- Mobile app support
- Multi-tenancy

## Contributing

See [CONTRIBUTING.md](../contributing/CONTRIBUTING.md) for development guidelines.

## Security

See [SECURITY.md](../guides/SECURITY.md) for security policy and reporting vulnerabilities.

## License

See [LICENSE](LICENSE) for license information.

---

**Note:** This project is under active development. Features and APIs may change before v1.0.0 stable release.
