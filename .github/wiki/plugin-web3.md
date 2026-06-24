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
| `PORT` | `3128` | Port the service listens on |
| `WEB3_RPC_URL_<CHAIN_ID>` | — | RPC endpoint per chain (e.g. `WEB3_RPC_URL_1=https://mainnet.infura.io/v3/KEY`) |
| `WEB3_DEFAULT_CHAIN_ID` | `1` | Default EVM chain id (1 = Ethereum mainnet) |
| `WEB3_SUPPORTED_CHAINS` | — | Comma-separated chain ids to enable |
| `WEB3_GATE_CHECK_CACHE_TTL` | `300` | Seconds to cache token-gate results |
| `WEB3_API_KEY` | — | Infura / Alchemy API key |

## Ports

| Port | Purpose |
|------|---------|
| `3128` | Web3 REST API |

## Database Tables

12 tables added to your Postgres database:

- `np_web3_wallets` — verified wallet bindings per user
- `np_web3_nfts` — NFT holdings indexed per wallet
- `np_web3_collections` — NFT collection metadata
- `np_web3_tokens` — ERC-20 token registry
- `np_web3_token_balances` — wallet balances per token
- `np_web3_token_gates` — token-gate rule definitions
- `np_web3_gate_checks` — cached gate-check evaluations
- `np_web3_daos` — DAO registry
- `np_web3_proposals` — DAO proposals lifecycle
- `np_web3_votes` — votes cast on proposals
- `np_web3_transactions` — observed blockchain transactions
- `np_web3_events` — outbound event log

## How It Works

Wallet verification uses a signed-nonce flow. The client requests a nonce, signs it with their wallet (EIP-191, compatible with ethers.js and viem), and posts the signature. The plugin verifies the signature server-side and stores the binding without ever handling private keys. DID resolution is supported via the wallet address as a did:pkh identifier.

Token gates evaluate ERC-20 balance thresholds or ERC-721 collection membership against on-chain data fetched via the configured RPC endpoint. Results are cached by TTL to avoid repeated RPC calls.

DAO governance tracks proposals and votes on-chain. The plugin stores proposal metadata and vote records locally and exposes them via REST so apps do not need direct chain access.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/api/web3/` | `localhost:3128` |

---

← [[Plugin-Overview]] | [[Home]] →
