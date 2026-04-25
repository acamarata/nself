# nSelf Benchmark Harness

Comparative benchmarks: nSelf vs Supabase vs Nhost vs PocketBase.

**Methodology:** All benchmarks run on a fresh Hetzner CX23 (2 vCPU, 4 GB RAM) with
Ubuntu 24.04. nSelf uses a default `nself init && nself start` configuration.
Competitor results are collected by a third-party reviewer following the methodology
in `METHODOLOGY.md`. Raw results are archived in `results/`.

**Third-party validation:** We invite any community member to re-run this harness and
submit a PR with their results. Independent replications strengthen the data.
Contradictory results are published as-is with the submitter's notes.

---

## Dimensions

| Dimension | Tool | Notes |
|-----------|------|-------|
| Setup time | `time` + shell script | From `curl install.sh` to first successful healthcheck |
| RPS at 1 vCPU | `oha -n 10000 -c 50` | GraphQL select-by-PK against a table with 10k rows |
| Cost per 10k req | Calculator | Based on published pricing at benchmark date |
| Feature coverage | Manual | 15-row matrix vs Supabase, Nhost, PocketBase |

---

## Running locally

```bash
.github/benchmarks/run.sh
```

The script outputs results to `results/YYYY-MM-DD-local.json`. CI runs it on a clean
Hetzner runner and archives to `results/YYYY-MM-DD-ci.json`.

---

## Quarterly re-run

CI runs `.github/workflows/benchmarks.yml` every quarter (first Monday of March, June,
September, December) and archives results in `results/`. A PR is auto-opened if any
nSelf metric degrades by >10% vs the previous quarter.

---

See [nself.org/benchmarks](https://nself.org/benchmarks) for the published results.
