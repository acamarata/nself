# P93 S49 — 48h production soak scripts

Gate scripts for the P93 production soak. See `.claude/docs/operations/p93-soak-protocol.md` for the full protocol.

| Script | Gate | Runs |
|---|---|---|
| `00-pre-flight.sh` | Pre-flight tag verification | Once, before deploy |
| `01-deploy-verify.sh` | T01 all 11 repos deployed | Every 5 min |
| `02-clock.sh` | T02 48h clock | Every 5 min (status) |
| `03-critical-events.sh` | T03 zero CRITICAL events | Every 5 min |
| `04-mux-classification.sh` | T04 100% mux rate | Every 5 min |
| `05-null-topics.sh` | T05 zero NULL topics | Every 5 min |
| `06-nself-ai-commands.sh` | T06 14/14 ai commands | Every 30 min |
| `07-pool-keys.sh` | T07 4 pool keys | Every 5 min |
| `08-oauth-reauth.sh` | T08 OAuth refresh | Every 30 min |
| `09-fresh-install.sh` | T09 fresh CX23 install | Every 6h (expensive) |
| `10-ollama-fallback.sh` | T10 Ollama fallback | Every 30 min |
| `run-all.sh` | Orchestrator | Every 5 min via CI |

## Start the soak

```bash
chmod +x cli/scripts/soak/*.sh
cli/scripts/soak/00-pre-flight.sh v1.0.9
# deploy via deploy-runbook.md
cli/scripts/soak/02-clock.sh start
# CI workflow p93-soak-monitor.yml runs run-all.sh every 5 min
```

## Environment

| Var | Purpose |
|---|---|
| `TARGET_VERSION` | Expected prod version (default 1.0.9) |
| `CLOCK_FILE` | Clock state file (default /var/lib/nself/soak-clock) |
| `EVENT_LOG` | JSONL event log (default /var/log/nself/soak-events.jsonl) |
| `PROM_URL` | Prometheus query API |
| `ALERTMANAGER_URL` | Alertmanager API |
| `PG_CONN` | Postgres connection for T05 |
| `MUX_BASE` | mux plugin base URL |
| `POOL_BASE` | AI pool base URL |
| `HETZNER_NSELF_TOKEN` | Hetzner API token for T09 |
| `SOAK_GOOGLE_REFRESH_TOKEN` | Long-lived refresh token for T08 |
| `SOAK_RUN_T09` | Set to `1` to include T09 in run-all |
