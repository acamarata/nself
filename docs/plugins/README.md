# nself Plugins (v0.4.8)

Extend nself with third-party integrations through the plugin system.

---

## Overview

The nself plugin system allows you to extend your backend with pre-built integrations for popular services. Plugins provide database schemas, webhook handlers, CLI actions, and optional background services.

---

## Plugin Documentation

| Document | Description |
|----------|-------------|
| **[Plugin Overview](index.md)** | Plugin system introduction and architecture |
| **[Plugin Development](development.md)** | Creating custom plugins |
| **[Stripe Plugin](stripe.md)** | Payment processing integration |
| **[GitHub Plugin](github.md)** | Repository and workflow sync |
| **[Shopify Plugin](shopify.md)** | E-commerce integration |

---

## Quick Start

### Install a Plugin

```bash
# List available plugins
nself plugin list

# Install Stripe plugin
nself plugin install stripe

# Configure plugin
# Edit .env and add STRIPE_SECRET_KEY

# Rebuild and restart
nself build && nself restart

# Sync data
nself plugin stripe sync
```

---

## Available Plugins

### Stripe - Payment Processing

**Category:** Billing & Payments
**Version:** 1.0.0

Integrate Stripe payments, subscriptions, and billing:

- Payment processing and invoices
- Subscription management
- Customer portal
- Usage-based billing
- Webhook handling for real-time events
- Analytics and reporting

**[View Stripe Plugin Documentation](stripe.md)**

```bash
nself plugin install stripe
```

---

### GitHub - DevOps Integration

**Category:** DevOps & Development
**Version:** 1.0.0

Sync GitHub data to your database:

- Repository information and metadata
- Issues and pull requests
- Workflow runs and deployments
- Commits and contributors
- Webhook events for real-time updates

**[View GitHub Plugin Documentation](github.md)**

```bash
nself plugin install github
```

---

### Shopify - E-commerce Integration

**Category:** E-commerce
**Version:** 1.0.0

Integrate Shopify store data:

- Products and inventory
- Orders and customers
- Fulfillment tracking
- Webhook events
- Analytics views

**[View Shopify Plugin Documentation](shopify.md)**

```bash
nself plugin install shopify
```

---

## Plugin Features

### Database Schemas

Plugins create prefixed database tables automatically:

```sql
-- Stripe plugin creates:
stripe_customers
stripe_subscriptions
stripe_invoices
stripe_payment_intents

-- Tables are automatically tracked in Hasura
```

### Webhook Handlers

Automatic webhook endpoint configuration:

```
POST /webhooks/stripe
POST /webhooks/github
POST /webhooks/shopify
```

**Features:**
- Signature verification
- Event processing
- Database updates
- Error handling

### CLI Actions

Manage plugin data via CLI:

```bash
# Stripe
nself plugin stripe sync                # Sync all data
nself plugin stripe customers list      # List customers
nself plugin stripe subscriptions       # Manage subscriptions

# GitHub
nself plugin github sync                # Sync repos and issues
nself plugin github repos list          # List repositories
nself plugin github workflows           # View workflow runs

# Shopify
nself plugin shopify sync               # Sync products and orders
nself plugin shopify products           # Manage products
nself plugin shopify orders             # View orders
```

### Analytics Views

Pre-built SQL views for insights:

```sql
-- Stripe analytics
SELECT * FROM stripe_revenue_by_month;
SELECT * FROM stripe_churn_analysis;

-- GitHub analytics
SELECT * FROM github_commit_activity;
SELECT * FROM github_pr_metrics;
```

### Optional Services

Some plugins include Docker services for background processing:

```yaml
# Stripe webhook processor
stripe-webhook-processor:
  image: nself/stripe-webhook-processor
  environment:
    - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
```

---

## Plugin Management

### List Available Plugins

```bash
# List all plugins in registry
nself plugin list

# Show plugin details
nself plugin info stripe
```

### Install Plugin

```bash
# Install latest version
nself plugin install stripe

# Install specific version
nself plugin install stripe@1.2.0
```

### Update Plugins

```bash
# Check for updates
nself plugin updates

# Update specific plugin
nself plugin update stripe

# Update all plugins
nself plugin update --all
```

### Uninstall Plugin

```bash
# Uninstall (keeps data)
nself plugin uninstall stripe

# Uninstall and remove data
nself plugin uninstall stripe --remove-data
```

---

## Plugin Configuration

### Environment Variables

Each plugin requires configuration via `.env`:

**Stripe:**
```bash
STRIPE_ENABLED=true
STRIPE_SECRET_KEY=sk_test_PLACEHOLDER...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

**GitHub:**
```bash
GITHUB_ENABLED=true
GITHUB_TOKEN=ghp_...
GITHUB_WEBHOOK_SECRET=...
GITHUB_REPOS=owner/repo1,owner/repo2
```

**Shopify:**
```bash
SHOPIFY_ENABLED=true
SHOPIFY_SHOP_NAME=your-store
SHOPIFY_API_KEY=...
SHOPIFY_API_SECRET=...
SHOPIFY_ACCESS_TOKEN=...
```

---

## Plugin Registry

### Official Registry

**Primary:** [plugins.nself.org](https://plugins.nself.org)
**Fallback:** GitHub raw files

**Features:**
- Plugin discovery
- Version management
- Update notifications
- Download tracking

### Registry Caching

Plugin registry is cached locally:

```bash
# Refresh registry cache
nself plugin refresh

# Clear cache
nself plugin cache clear
```

---

## Creating Custom Plugins

### Plugin Structure

```
my-plugin/
├── plugin.yaml           # Plugin metadata
├── schema.sql           # Database schema
├── routes.yaml          # Nginx routes
├── cli.yaml            # CLI commands
├── services.yaml       # Docker services (optional)
├── views/              # Analytics views
│   └── revenue.sql
└── README.md           # Documentation
```

### Example: Simple Plugin

**plugin.yaml:**
```yaml
name: my-plugin
version: 1.0.0
description: My custom integration
author: Your Name
category: Integration

tables:
  - name: my_plugin_items
    prefix: my_plugin_
```

**schema.sql:**
```sql
CREATE TABLE IF NOT EXISTS my_plugin_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**[View Full Plugin Development Guide](development.md)**

---

## Plugin Architecture

### Data Flow

```
External Service (Stripe, GitHub, etc.)
    ↓ Webhook
Nginx (signature verification)
    ↓ Route to handler
Plugin Webhook Handler
    ↓ Process event
PostgreSQL (update tables)
    ↓ Auto-tracked by Hasura
GraphQL API (available to frontend)
```

### CLI Integration

```
nself plugin <plugin-name> <action>
    ↓
Plugin CLI Handler
    ↓ Execute action
API Call / Database Query
    ↓
Return Results
```

---

## Security

### Webhook Signature Verification

All webhook handlers verify signatures:

**Stripe:**
```javascript
const signature = req.headers['stripe-signature'];
const event = stripe.webhooks.constructEvent(
  req.body,
  signature,
  process.env.STRIPE_WEBHOOK_SECRET
);
```

**GitHub:**
```javascript
const signature = req.headers['x-hub-signature-256'];
const isValid = verifyGitHubSignature(
  req.body,
  signature,
  process.env.GITHUB_WEBHOOK_SECRET
);
```

### API Key Storage

```bash
# Store secrets in .secrets file (gitignored)
STRIPE_SECRET_KEY=sk_live_...

# Never commit secrets to git
echo ".secrets" >> .gitignore
```

---

## Troubleshooting

### Plugin Not Installing

```bash
# Refresh registry
nself plugin refresh

# Clear cache
nself plugin cache clear

# Try again
nself plugin install stripe
```

### Webhooks Not Working

```bash
# Check webhook secret is configured
echo $STRIPE_WEBHOOK_SECRET

# Test webhook locally with CLI
stripe listen --forward-to localhost:1337/webhooks/stripe

# Check nginx logs
docker logs myapp_nginx
```

### Data Not Syncing

```bash
# Check API keys
nself plugin stripe test-connection

# Manual sync
nself plugin stripe sync --force

# View sync logs
nself logs stripe-sync
```

---

## Plugin Roadmap

### Coming Soon

- **SendGrid** - Email delivery and tracking
- **Twilio** - SMS and voice integration
- **Auth0** - Advanced authentication
- **Algolia** - Advanced search
- **AWS S3** - Cloud storage
- **Mailgun** - Email service
- **Postmark** - Transactional email
- **Plaid** - Banking and financial data
- **Segment** - Customer data platform

**Request a plugin:** [GitHub Discussions](https://github.com/acamarata/nself/discussions)

---

## Related Documentation

- **[Commands: plugin](../commands/PLUGIN.md)** - Plugin CLI reference
- **[Architecture](../architecture/ARCHITECTURE.md)** - System architecture
- **[Services](../services/SERVICES.md)** - Available services
- **[Guides](../guides/README.md)** - Usage guides

---

**[Back to Documentation Home](../README.md)**
