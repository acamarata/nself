package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ── Registration ─────────────────────────────────────────────────────────────

// TestCostsCmd_Registered verifies costsCmd is on the global RootCmd.
func TestCostsCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Use == "costs" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("costsCmd is not registered on RootCmd")
	}
}

// TestCostsCmd_FlagsRegistered verifies --server-type and --format flags exist.
func TestCostsCmd_FlagsRegistered(t *testing.T) {
	for _, flag := range []string{"server-type", "format"} {
		if costsCmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag --%s not registered on costsCmd", flag)
		}
	}
}

// ── JSON output ──────────────────────────────────────────────────────────────

// TestCostsCmd_JSONOutputValid is the primary acceptance criterion:
// `nself costs --format json` must produce output parseable by jq (valid JSON array).
func TestCostsCmd_JSONOutputValid(t *testing.T) {
	buf := &bytes.Buffer{}
	costsCmd.SetOut(buf)

	// Use a known server type so the output is deterministic.
	t.Setenv("HETZNER_SERVER_TYPE", "cx23")
	costsFormat = "json"
	t.Cleanup(func() { costsFormat = "table" })

	err := costsCmd.RunE(costsCmd, []string{})
	if err != nil {
		t.Fatalf("costsCmd --format json: %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Fatal("expected JSON output, got empty string")
	}

	// Must be a valid JSON array (parseable like jq would).
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("costs --format json output is not valid JSON array: %v\noutput was:\n%s", err, out)
	}

	if len(arr) == 0 {
		t.Error("expected at least one cost line in JSON output, got empty array")
	}

	// Every entry must have required fields.
	for i, entry := range arr {
		for _, field := range []string{"category", "item", "monthly_usd", "notes"} {
			if _, ok := entry[field]; !ok {
				t.Errorf("entry[%d] missing field %q", i, field)
			}
		}
	}
}

// TestCostsCmd_JSONContainsTOTAL verifies the TOTAL entry is present in JSON output.
func TestCostsCmd_JSONContainsTOTAL(t *testing.T) {
	buf := &bytes.Buffer{}
	costsCmd.SetOut(buf)

	costsFormat = "json"
	t.Cleanup(func() { costsFormat = "table" })
	costsServerType = "cx23"
	t.Cleanup(func() { costsServerType = "" })

	if err := costsCmd.RunE(costsCmd, []string{}); err != nil {
		t.Fatalf("runCosts: %v", err)
	}

	out := buf.String()
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	foundTotal := false
	for _, entry := range arr {
		if cat, _ := entry["category"].(string); cat == "TOTAL" {
			foundTotal = true
			// Monthly USD for known cx23 must be non-negative.
			if v, ok := entry["monthly_usd"].(float64); ok && v < 0 {
				t.Errorf("TOTAL monthly_usd should be >= 0, got %v", v)
			}
		}
	}
	if !foundTotal {
		t.Error("TOTAL line not found in JSON output")
	}
}

// ── Table output ─────────────────────────────────────────────────────────────

// TestCostsCmd_TableOutputHasHeaders verifies table mode includes header columns.
func TestCostsCmd_TableOutputHasHeaders(t *testing.T) {
	buf := &bytes.Buffer{}
	costsCmd.SetOut(buf)

	costsFormat = "table"
	costsServerType = "cx23"
	t.Cleanup(func() { costsServerType = "" })

	if err := costsCmd.RunE(costsCmd, []string{}); err != nil {
		t.Fatalf("runCosts table: %v", err)
	}

	out := buf.String()
	for _, col := range []string{"CATEGORY", "ITEM", "MONTHLY", "NOTES"} {
		if !strings.Contains(out, col) {
			t.Errorf("table output missing column %q", col)
		}
	}
}

// TestCostsCmd_UnknownServerType verifies unknown server type still produces
// valid output (with a note) and exits 0.
func TestCostsCmd_UnknownServerType(t *testing.T) {
	buf := &bytes.Buffer{}
	costsCmd.SetOut(buf)

	costsFormat = "table"
	t.Cleanup(func() { costsFormat = "table" })
	costsServerType = "cx99999-unknown"
	t.Cleanup(func() { costsServerType = "" })

	err := costsCmd.RunE(costsCmd, []string{})
	if err != nil {
		t.Fatalf("unknown server type should not error, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "cx99999-unknown") {
		t.Error("expected unknown server type to appear in output")
	}
}

// TestCostsCmd_JSONUnknownServerType verifies JSON output is still valid for
// unknown server types (the "unknown type" row must be well-formed JSON).
func TestCostsCmd_JSONUnknownServerType(t *testing.T) {
	buf := &bytes.Buffer{}
	costsCmd.SetOut(buf)

	costsFormat = "json"
	t.Cleanup(func() { costsFormat = "table" })
	costsServerType = "unknowntype999"
	t.Cleanup(func() { costsServerType = "" })

	if err := costsCmd.RunE(costsCmd, []string{}); err != nil {
		t.Fatalf("runCosts JSON unknown type: %v", err)
	}

	var arr []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &arr); err != nil {
		t.Fatalf("output not valid JSON for unknown server type: %v\noutput:\n%s", err, buf.String())
	}
}
