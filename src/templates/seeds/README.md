# nself Seed Templates

This directory contains SQL seed templates for initializing database data.

## Available Templates

### 001_auth_users.sql.template

Seeds authentication users using nHost auth schema structure.

**Placeholders:**
- `{{TIMESTAMP}}` - Current timestamp
- `{{PROJECT_NAME}}` - Project name
- `{{DEFAULT_PASSWORD}}` - Default password (bcrypt hashed)
- `{{OWNER_EMAIL}}` - Owner email address
- `{{ADMIN_EMAIL}}` - Admin email address
- `{{SUPPORT_EMAIL}}` - Support email address

**Default values:**
- Password: `npass123` (development only!)
- Owner: `owner@nself.org`
- Admin: `admin@nself.org`
- Support: `support@nself.org`

## Usage

Seeds can be applied using:

```bash
# Apply all seeds
nself db seed apply

# Apply specific seed
nself db seed apply path/to/seed.sql

# List seed status
nself db seed list

# Create new seed from template
nself db seed create my_seed
```

## nHost Auth Schema

The auth users seed follows the nHost authentication schema:

1. **auth.providers** - Authentication providers (email, google, github, etc.)
2. **auth.users** - User accounts with hashed passwords
3. **auth.user_providers** - Links users to their provider identities (email addresses)

### Important Notes

- Passwords are hashed using **bcrypt** (cost factor 10) via PostgreSQL's `pgcrypto` extension
- Access tokens for seeded users are **dummy tokens** (`seed_token_<uuid>`)
- Seeds are **idempotent** using `ON CONFLICT` clauses
- UUIDs are deterministic for default users (11111..., 22222..., 33333...)

## Creating Custom Seeds

1. Copy template: `cp 001_auth_users.sql.template my_custom_seed.sql`
2. Replace placeholders with your values
3. Place in `nself/seeds/common/` or environment-specific directory
4. Apply with `nself db seed apply`

## Environment-Specific Seeds

Organize seeds by environment:

```
nself/seeds/
├── common/         # Applied in all environments
├── local/          # Local development only
├── staging/        # Staging environment
└── production/     # Production environment
```

## Security Best Practices

- **Never commit** production passwords to seed files
- Use **environment variables** for sensitive values
- **Regenerate passwords** after initial setup
- Consider using `nself auth create-user` for production users instead of seeds
