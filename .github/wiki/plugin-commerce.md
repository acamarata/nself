# Commerce Plugin

> E-commerce foundation — products, orders, cart, and inventory management. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install commerce
```

## What It Does

Adds a complete e-commerce data layer to your Postgres database. Manages product catalogs, inventory, shopping carts, and orders. Designed to be payment-processor agnostic — pair with the `stripe`, `paypal`, or `shopify` plugins for payment processing. Exposes a GraphQL API via Hasura for storefront frontends.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `COMMERCE_PORT` | `3042` | Commerce service port |
| `COMMERCE_CURRENCY` | `usd` | Default store currency |
| `COMMERCE_TAX_ENABLED` | `false` | Enable tax calculation |
| `COMMERCE_INVENTORY_TRACKING` | `true` | Track stock levels |

## Ports

| Port | Purpose |
|------|---------|
| 3715 | Commerce REST API |

## Database Tables

8 tables added to your Postgres database:
- `np_commerce_products` — product catalog
- `np_commerce_variants` — product variants (size, color, etc.)
- `np_commerce_inventory` — stock levels per variant
- `np_commerce_carts` — shopping cart sessions
- `np_commerce_cart_items` — items in each cart
- `np_commerce_orders` — order records
- `np_commerce_order_items` — line items per order
- `np_commerce_categories` — product categories

## Nginx Routes

| Route | Target |
|-------|--------|
| `/commerce/` | Commerce API |
