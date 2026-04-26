# Runbook: OTEL Tail Sampling — policy + tuning

**Component:** OTEL Collector tail_sampling processor (`cli/monitoring/otelcol-config.yaml`)
**Sprint:** G6-T07

## Sampling policy

Three policies evaluate in order. First match wins.

| Policy | Type | Sample rate | What it catches |
|---|---|---|---|
| `error-policy` | `status_code` | 100% | Any trace containing one or more spans with status `ERROR` |
| `high-volume-policy` | `and` (route + probabilistic) | 1% | Traces whose root span has `http.route` in `/health`, `/healthz`, `/readyz`, `/livez`, `/metrics`, `/ping` |
| `default-policy` | `probabilistic` | 10% | All other traces |

The combined effect: complete error capture, minimal health-probe noise, statistical baseline for everything else. Expect ~10–15% of total span volume retained on a typical workload.

## Tuning knobs

### `decision_wait` (default 10s)

How long the collector buffers a trace before deciding to sample it.

- Increase if you see late-arriving child spans missing from sampled error traces (slow downstream services, queued worker spans).
- Decrease if collector memory pressure is high and traces complete quickly.
- Cost: linear memory growth with `decision_wait * traces_per_sec`.

### `num_traces` (default 50000)

Maximum traces held in memory during the decision window.

- Increase if you see `tail_sampling_processor_traces_dropped` rising — buffer is overflowing.
- Decrease if collector RSS exceeds 80% of `memory_limiter.limit_mib`.
- Rule of thumb: `num_traces ≈ expected_new_traces_per_sec * decision_wait * 5`.

### `expected_new_traces_per_sec` (default 100)

Hint to the processor for sizing its hash table. Tune to roughly match real load — accuracy not critical.

### Per-policy `sampling_percentage`

- `default-policy`: raise to 25–50% in dev / staging for richer traces; lower to 1–5% in high-volume prod.
- `high-volume-policy`: 1% is usually too high for very chatty environments — drop to 0.1% if Prometheus / k8s probe traffic dominates.
- `error-policy`: keep at 100%. Errors are rare and high-value.

## Common issues

| Symptom | Cause | Fix |
|---|---|---|
| Errors missing from Tempo | `decision_wait` too short — child error spans arriving late | Bump to 15–20s |
| Collector OOM | `num_traces` × avg trace size exceeds `memory_limiter.limit_mib` | Lower `num_traces` or raise the limit |
| Sampled rate higher than expected | `default-policy` set too high, or many spans flagged ERROR | Audit error rate, lower probability |
| Health-probe spans still flooding Tempo | Routes don't match `http.route` exactly | Check service instrumentation; route may be stripped to `/` |

## Verification

```bash
# Inspect policy hits over the last 5m (collector internal metrics)
curl -s http://localhost:8888/metrics | grep otelcol_processor_tail_sampling_count_traces_sampled

# Tempo trace count by service over the last hour
# (run from Grafana Explore → Tempo)
```

## Related

- [[otel-collector-down.md]] — collector availability + ingest drop
- [[tempo-down.md]] — exporter backpressure caused by Tempo outage
- `cli/monitoring/otelcol-config.yaml` — canonical config
