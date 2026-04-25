# Benchmark Methodology

## Infrastructure

- **Server type:** Hetzner CX23 (2 vCPU AMD, 4 GB RAM, 40 GB SSD, NVMe)
- **Region:** Falkenstein, Germany (fsn1)
- **OS:** Ubuntu 24.04 LTS, fresh image, no custom tuning
- **Network:** no VPN, direct Hetzner network

## nSelf Setup

```bash
curl -fsSL https://install.nself.org/cli | bash
nself init --name bench-test --no-monitoring
nself start
```

Timing starts from `curl` start and ends when `nself doctor` reports all-green.

## Competitor Setup

Each competitor follows its official quickstart exactly, using the latest version at
benchmark date. No custom configuration is applied. Timing ends when the official
healthcheck returns 200.

## Load Test

```bash
# Install oha (HTTP benchmark tool)
brew install oha  # or cargo install oha

# Seed 10,000 rows
./seed.sh

# Run load test
oha --no-tui -n 10000 -c 50 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  --data '{"query":"{ users(where: {id: {_eq: \"$RANDOM_ID\"}}) { id name } }"}' \
  http://localhost:8080/v1/graphql
```

## Cost Per 10k Requests

Calculated at benchmark date using public pricing:
- nSelf: VPS cost per hour ÷ requests per hour × 10,000
- Competitors: published API pricing per request × 10,000 (where applicable)

Self-hosted competitors: same VPS cost formula as nSelf.

## Feature Coverage Matrix

Manually evaluated. Each row is checked against official documentation. Contested
entries link to the documentation source. Community corrections accepted via PR.

## Disclosure

Results are honest: if a competitor wins on a metric, we publish it. We update quarterly
and re-run whenever a competitor ships a major version. See `results/` for raw data.
