# Checkout Plugin

> Checkout flow service — cart to payment to fulfillment pipeline. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install checkout
```

## What It Does

Manages the complete checkout flow from cart validation through payment capture to fulfillment. Handles address validation, coupon application, tax calculation, payment processing (via Stripe), and post-purchase fulfillment triggers. Exposes a simple REST API that your storefront frontend calls to complete purchases.

## Dependencies

Pairs with the `commerce` and `stripe` plugins.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CHECKOUT_PORT` | `3044` | Checkout service port |
| `CHECKOUT_CURRENCY` | `usd` | Store currency |
| `CHECKOUT_TAX_PROVIDER` | — | Tax provider: `taxjar`, `avalara`, or blank |
| `CHECKOUT_SUCCESS_URL` | — | Redirect URL after successful payment |
| `CHECKOUT_CANCEL_URL` | — | Redirect URL on cancellation |

## Ports

| Port | Purpose |
|------|---------|
| 3044 | Checkout REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_checkout_sessions` — checkout session state
- `np_checkout_addresses` — saved customer addresses
- `np_checkout_coupons` — coupon definitions
- `np_checkout_fulfillments` — fulfillment records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/checkout/` | Checkout flow API |
