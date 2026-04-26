# Shopify Plugin

> Shopify store sync, orders, products, customers, and inventory. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install shopify
```

## What It Does

Syncs your Shopify store data into Postgres in real time via webhooks and the Shopify REST API. Tracks products, collections, customers, orders, fulfillments, and inventory across locations. Query Shopify data directly via Hasura GraphQL for custom storefronts, analytics dashboards, and fulfillment workflows.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SHOPIFY_PORT` | `3072` | Shopify plugin port |
| `SHOPIFY_SHOP_DOMAIN` | — | Your Shopify store domain |
| `SHOPIFY_ACCESS_TOKEN` | — | Shopify Admin API access token |
| `SHOPIFY_WEBHOOK_SECRET` | — | Shopify webhook HMAC secret |
| `SHOPIFY_API_VERSION` | `2024-01` | Shopify API version |

## Ports

| Port | Purpose |
|------|---------|
| 3072 | Shopify integration REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_shopify_products`, product catalog
- `np_shopify_variants`, product variants
- `np_shopify_collections`, product collections
- `np_shopify_customers`, customer records
- `np_shopify_orders`, order records
- `np_shopify_line_items`, order line items
- `np_shopify_fulfillments`, fulfillment records
- `np_shopify_inventory`, inventory levels
- `np_shopify_webhook_events`, raw event log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/shopify/webhook` | Shopify webhook receiver |
