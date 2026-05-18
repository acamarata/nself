package dr

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var testStart = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

// newPassingReport builds a fully passing DrillReport for use in tests.
func newPassingReport() *DrillReport {
	r := NewDrillReport("drill-001", "vm-abc", "backup-xyz", testStart)
	r.Scenarios.PGRestore = true
	r.Scenarios.HasuraUp = true
	r.Scenarios.MinIOMetadata = true
	for p := range r.Scenarios.PluginHealth {
		r.Scenarios.PluginHealth[p] = true
	}
	r.RTOActualSec = 120
	r.RPOActualSec = 30
	return r
}

// TestNewDrillReport_Fields verifies NewDrillReport populates required fields.
func TestNewDrillReport_Fields(t *testing.T) {
	r := NewDrillReport("drill-abc", "vm-001", "bkp-001", testStart)

	if r.DrillID != "drill-abc" {
		t.Errorf("DrillID: got %q, want drill-abc", r.DrillID)
	}
	if r.VMID != "vm-001" {
		t.Errorf("VMID: got %q, want vm-001", r.VMID)
	}
	if r.BackupID != "bkp-001" {
		t.Errorf("BackupID: got %q, want bkp-001", r.BackupID)
	}
	if !r.StartedAt.Equal(testStart) {
		t.Errorf("StartedAt: got %v, want %v", r.StartedAt, testStart)
	}
	// Default result is fail until Finalize.
	if r.Result != "fail" {
		t.Errorf("Result: got %q, want fail (default before Finalize)", r.Result)
	}
}

// TestNewDrillReport_RequiredPlugins verifies all required plugins are pre-seeded.
func TestNewDrillReport_RequiredPlugins(t *testing.T) {
	r := NewDrillReport("drill-req", "vm-001", "bkp-001", testStart)

	for _, p := range RequiredPluginChecks {
		if healthy, ok := r.Scenarios.PluginHealth[p]; !ok {
			t.Errorf("plugin %q missing from PluginHealth", p)
		} else if healthy {
			t.Errorf("plugin %q: default health should be false", p)
		}
	}
}

// TestFinalize_Pass verifies a fully successful report gets result=pass.
func TestFinalize_Pass(t *testing.T) {
	r := newPassingReport()
	finished := testStart.Add(10 * time.Minute)
	r.Finalize(finished)

	if r.Result != "pass" {
		t.Errorf("Result: got %q, want pass", r.Result)
	}
	if !r.FinishedAt.Equal(finished) {
		t.Errorf("FinishedAt: got %v, want %v", r.FinishedAt, finished)
	}
}

// TestFinalize_FailOnRTO verifies RTO=0 causes fail.
func TestFinalize_FailOnRTO(t *testing.T) {
	r := newPassingReport()
	r.RTOActualSec = 0
	r.Finalize(testStart.Add(5 * time.Minute))
	if r.Result != "fail" {
		t.Errorf("Result with RTO=0: got %q, want fail", r.Result)
	}
}

// TestFinalize_FailOnNegativeRTO verifies negative RTO causes fail.
func TestFinalize_FailOnNegativeRTO(t *testing.T) {
	r := newPassingReport()
	r.RTOActualSec = -1
	r.Finalize(testStart.Add(5 * time.Minute))
	if r.Result != "fail" {
		t.Errorf("Result with negative RTO: got %q, want fail", r.Result)
	}
}

// TestFinalize_FailOnScenario verifies a single false scenario causes fail.
func TestFinalize_FailOnScenario(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*DrillReport)
	}{
		{"PGRestore=false", func(r *DrillReport) { r.Scenarios.PGRestore = false }},
		{"HasuraUp=false", func(r *DrillReport) { r.Scenarios.HasuraUp = false }},
		{"MinIOMetadata=false", func(r *DrillReport) { r.Scenarios.MinIOMetadata = false }},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := newPassingReport()
			tc.mut(r)
			r.Finalize(testStart.Add(5 * time.Minute))
			if r.Result != "fail" {
				t.Errorf("Result: got %q, want fail when %s", r.Result, tc.name)
			}
		})
	}
}

// TestFinalize_FailOnPluginUnhealthy verifies unhealthy plugin causes fail.
func TestFinalize_FailOnPluginUnhealthy(t *testing.T) {
	r := newPassingReport()
	// Mark the first required plugin unhealthy.
	r.Scenarios.PluginHealth[RequiredPluginChecks[0]] = false
	r.Finalize(testStart.Add(5 * time.Minute))
	if r.Result != "fail" {
		t.Errorf("Result with unhealthy plugin: got %q, want fail", r.Result)
	}
}

// TestValidate_Pass verifies a valid finalized report passes validation.
func TestValidate_Pass(t *testing.T) {
	r := newPassingReport()
	r.Finalize(testStart.Add(10 * time.Minute))

	if err := r.Validate(); err != nil {
		t.Errorf("Validate on passing report: unexpected error: %v", err)
	}
}

// TestValidate_MissingFields verifies Validate catches each missing field.
func TestValidate_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*DrillReport)
	}{
		{"DrillID empty", func(r *DrillReport) { r.DrillID = "" }},
		{"StartedAt zero", func(r *DrillReport) { r.StartedAt = time.Time{} }},
		{"FinishedAt zero", func(r *DrillReport) { r.FinishedAt = time.Time{} }},
		{"VMID empty", func(r *DrillReport) { r.VMID = "" }},
		{"BackupID empty", func(r *DrillReport) { r.BackupID = "" }},
		{"Result invalid", func(r *DrillReport) { r.Result = "unknown" }},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := newPassingReport()
			r.Finalize(testStart.Add(5 * time.Minute))
			tc.mut(r)
			if err := r.Validate(); err == nil {
				t.Errorf("Validate: expected error when %s, got nil", tc.name)
			}
		})
	}
}

// TestValidate_MissingRequiredPlugin verifies missing plugin entry is caught.
func TestValidate_MissingRequiredPlugin(t *testing.T) {
	r := newPassingReport()
	r.Finalize(testStart.Add(5 * time.Minute))
	// Remove a required plugin.
	delete(r.Scenarios.PluginHealth, RequiredPluginChecks[0])

	if err := r.Validate(); err == nil {
		t.Error("Validate: expected error for missing required plugin, got nil")
	}
}

// TestMarshal_ValidJSON verifies Marshal produces parseable JSON with required fields.
func TestMarshal_ValidJSON(t *testing.T) {
	r := newPassingReport()
	r.Finalize(testStart.Add(5 * time.Minute))

	data, err := r.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundtrip DrillReport
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("json.Unmarshal after Marshal: %v", err)
	}
	if roundtrip.DrillID != r.DrillID {
		t.Errorf("roundtrip DrillID: got %q, want %q", roundtrip.DrillID, r.DrillID)
	}
	if roundtrip.Result != r.Result {
		t.Errorf("roundtrip Result: got %q, want %q", roundtrip.Result, r.Result)
	}
	// Verify pretty-print indentation (2 spaces).
	if !strings.Contains(string(data), "  ") {
		t.Error("Marshal: output does not appear to be indented")
	}
}
