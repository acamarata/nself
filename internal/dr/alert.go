package dr

// DrillAlertRuleYAML is the exact Alertmanager/Prometheus rule that fires when
// a monthly drill produces a non-"pass" result. The content is deployed to
// `web/backend/nself/monitoring/alerts/dr.rules.yml` and evaluated by the
// Prometheus instance on nclaw-prod.
const DrillAlertRuleYAML = `groups:
  - name: nself-dr
    rules:
      - alert: DRDrillFailed
        expr: nself_dr_drill_result{result!="pass"} == 1
        for: 0m
        labels:
          severity: critical
          service: dr
        annotations:
          summary: "DR drill {{ $labels.drill_id }} failed"
          runbook: "https://nself.org/docs/ops/dr-drill-failure"
`

// DrillResultMetricName is the Prometheus metric name emitted by a drill run.
// Labels: drill_id (unique per run), result ("pass" or "fail"), rto_sec, rpo_sec.
const DrillResultMetricName = "nself_dr_drill_result"
