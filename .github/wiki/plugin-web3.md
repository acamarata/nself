# Web3 Plugin

> NFT management, token-gating for content, DAO governance, wallet management, and blockchain transaction tracking. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install web3
```

## What It Does

The web3 plugin integrates blockchain capabilities into your nself backend, providing wallet management, NFT and token tracking, and token-gating so you can restrict content access to holders of specific assets. It also includes DAO governance primitives (proposals and voting) and a full transaction and event log synced from your configured RPC endpoint.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `WEB3_PORT` | `3128` | Port the service listens on |
| `WEB3_PROVIDER_URL` | — | RPC endpoint URL (e.g. Infura, Alchemy, or self-hosted node) |
| `WEB3_CHAIN_ID` | — | Chain ID of the target network (e.g. `1` for Ethereum mainnet) |

## Ports

| Port | Purpose |
|------|---------|
| `3128` | Web3 REST API |

## Database Tables

12 tables added to your Postgres database:

- `np_web3_wallets`
- `np_web3_nfts`
- `np_web3_tokens`
- `np_web3_gates`
- `np_web3_transactions`
- `np_web3_daos`
- `np_web3_proposals`
- `np_web3_votes`
- `np_web3_collections`
- `np_web3_contracts`
- `np_web3_events`
- `np_web3_sync_log`

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/web3/` | `localhost:3128` |
