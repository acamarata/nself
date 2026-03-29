# Entitlements Plugin

> Feature gating, subscription plans, usage quotas, and metered billing. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install entitlements
```

## What It Does

Manages what each user or tenant can access based on their subscription plan. Define feature flags per plan tier, enforce usage quotas (API calls, storage, seats), track metered usage for billing, and gate access to premium features. Integrates with the `stripe` plugin for plan state sync. Provides a fast entitlement check API that your application calls before serving features.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ENTITLEMENTS_PORT` | `3715` | Entitlements service port |
| `ENTITLEMENTS_CACHE_TTL` | `300` | Entitlement check cache TTL |
| `ENTITLEMENTS_METERED_ENABLED` | `false` | Enable metered usage billing |
| `ENTITLEMENTS_OVERAGE_ACTION` | `block` | On quota exceeded: `block` or `allow` |

## Ports

| Port | Purpose |
|------|---------|
| 3715 | Entitlements REST API |

## Database Tables

8 tables added to your Postgres database:
- `np_entitlements_plans` — subscription plan definitions
- `np_entitlements_features` — feature flag definitions
- `np_entitlements_plan_features` — plan-feature mappings
- `np_entitlements_subscriptions` — user plan assignments
- `np_entitlements_quotas` — quota definitions
- `np_entitlements_usage` — metered usage counters
- `np_entitlements_overages` — quota overage records
- `np_entitlements_grants` — manual feature grants

## Nginx Routes

| Route | Target |
|-------|--------|
| `/entitlements/` | Entitlements management API |
| `/entitlements/check` | Fast entitlement check endpoint |
