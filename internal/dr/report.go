package dr

import (
	"encoding/json"
	"fmt"
	"time"
)

// DrillReport is the persisted JSON schema for a monthly DR drill run.
//
// It is inserted into the `dr_drill_report` PG table on nclaw-prod via an
// authenticated API call, and is also the payload posted to the #ops-dr
// Telegram channel. Field names match the spec in p88-block-g section 4.4.
type DrillReport struct {
	DrillID       string          `json:"drill_id"`
	StartedAt     time.Time       `json:"started_at"`
	FinishedAt    time.Time       `json:"finished_at"`
	VMID          string          `json:"vm_id"`
	BackupID      string          `json:"backup_id"`
	RTOActualSec  int64           `json:"rto_actual_sec"`
	RPOActualSec  int64           `json:"rpo_actual_sec"`
	Result        string          `json:"result"` // pass | fail
	Scenarios     DrillScenarios  `json:"scenarios"`
	CostEUR       float64         `json:"cost_eur"`
}

// DrillScenarios captures per-check pass/fail across the smoke suite. All
// boolean fields are true on success. PluginHealth maps plugin name to health
// status and MUST include claw, ai, and mux at minimum.
type DrillScenarios struct {
	PGRestore     bool            `json:"pg_restore"`
	HasuraUp      bool            `json:"hasura_up"`
	MinIOMetadata bool            `json:"minio_metadata"`
	PluginHealth  map[string]bool `json:"plugin_health"`
}

// RequiredPluginChecks lists plugins whose health MUST be present in every
// drill report. Additional installed plugins are appended at runtime.
var RequiredPluginChecks = []string{"claw", "ai", "mux"}

// NewDrillReport builds a new DrillReport with zero values for all scenario
// checks and a pre-populated PluginHealth map seeded with required plugins.
func NewDrillReport(drillID, vmID, backupID string, startedAt time.Time) *DrillReport {
	plugins := make(map[string]bool, len(RequiredPluginChecks))
	for _, p := range RequiredPluginChecks {
		plugins[p] = false
	}
	return &DrillReport{
		DrillID:   drillID,
		StartedAt: startedAt,
		VMID:      vmID,
		BackupID:  backupID,
		Result:    "fail",
		Scenarios: DrillScenarios{PluginHealth: plugins},
	}
}

// Finalize stamps the finish time and computes pass/fail across all scenarios.
// A report passes iff every scenario check and every plugin health check is
// true, and both RTO/RPO values were recorded (> 0).
func (r *DrillReport) Finalize(finishedAt time.Time) {
	r.FinishedAt = finishedAt
	if r.RTOActualSec <= 0 || r.RPOActualSec < 0 {
		r.Result = "fail"
		return
	}
	if !r.Scenarios.PGRestore || !r.Scenarios.HasuraUp || !r.Scenarios.MinIOMetadata {
		r.Result = "fail"
		return
	}
	for _, healthy := range r.Scenarios.PluginHealth {
		if !healthy {
			r.Result = "fail"
			return
		}
	}
	r.Result = "pass"
}

// Validate ensures the report has every required field populated. It is used
// by the storage layer to reject malformed reports before they hit PG.
func (r *DrillReport) Validate() error {
	if r.DrillID == "" {
		return fmt.Errorf("drill_id is required")
	}
	if r.StartedAt.IsZero() || r.FinishedAt.IsZero() {
		return fmt.Errorf("started_at and finished_at are required")
	}
	if r.VMID == "" {
		return fmt.Errorf("vm_id is required")
	}
	if r.BackupID == "" {
		return fmt.Errorf("backup_id is required")
	}
	if r.Result != "pass" && r.Result != "fail" {
		return fmt.Errorf("result must be \"pass\" or \"fail\", got %q", r.Result)
	}
	for _, p := range RequiredPluginChecks {
		if _, ok := r.Scenarios.PluginHealth[p]; !ok {
			return fmt.Errorf("scenarios.plugin_health missing required plugin %q", p)
		}
	}
	return nil
}

// MarshalJSON renders the report in the exact field order defined by the spec.
func (r *DrillReport) Marshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// DrillReportTableDDL is the idempotent Postgres DDL for the report table.
// It is applied once on nclaw-prod during cron install.
const DrillReportTableDDL = `
CREATE TABLE IF NOT EXISTS dr_drill_report (
    drill_id         TEXT PRIMARY KEY,
    started_at       TIMESTAMPTZ NOT NULL,
    finished_at      TIMESTAMPTZ NOT NULL,
    vm_id            TEXT NOT NULL,
    backup_id        TEXT NOT NULL,
    rto_actual_sec   BIGINT NOT NULL,
    rpo_actual_sec   BIGINT NOT NULL,
    result           TEXT NOT NULL CHECK (result IN ('pass','fail')),
    scenarios        JSONB NOT NULL,
    cost_eur         NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS dr_drill_report_started_at_idx
    ON dr_drill_report (started_at DESC);
`
