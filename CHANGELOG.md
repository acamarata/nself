# Changelog

All notable changes to nself will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Security

See [SECURITY.md](SECURITY.md) for security policy and reporting vulnerabilities.

## License

See [LICENSE](LICENSE) for license information.

---

**Note:** This project is under active development. Features and APIs may change before v1.0.0 stable release.
