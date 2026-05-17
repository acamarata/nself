package gdpr

import (
	"bytes"
	"strings"
	"testing"
)

// TestWriteCSV_EmptyRows returns nil and writes nothing.
func TestWriteCSV_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCSV(&buf, nil); err != nil {
		t.Errorf("writeCSV(nil): unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("writeCSV(nil): wrote %d bytes, want 0", buf.Len())
	}
	if err := writeCSV(&buf, []map[string]interface{}{}); err != nil {
		t.Errorf("writeCSV(empty): unexpected error: %v", err)
	}
}

// TestWriteCSV_SingleRow verifies header + data row output.
func TestWriteCSV_SingleRow(t *testing.T) {
	rows := []map[string]interface{}{
		{"email": "alice@example.com", "id": "u-001", "name": "Alice"},
	}
	var buf bytes.Buffer
	if err := writeCSV(&buf, rows); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("writeCSV: got %d lines, want 2 (header + 1 data row)\noutput: %q", len(lines), output)
	}
	// Header must be alphabetically sorted.
	header := lines[0]
	if header != "email,id,name" {
		t.Errorf("header: got %q, want %q (alphabetical)", header, "email,id,name")
	}
	data := lines[1]
	if !strings.Contains(data, "alice@example.com") {
		t.Errorf("data row missing email: %q", data)
	}
}

// TestWriteCSV_AlphabeticalColumns verifies consistent column ordering.
func TestWriteCSV_AlphabeticalColumns(t *testing.T) {
	rows := []map[string]interface{}{
		{"zzz": "last", "aaa": "first", "mmm": "middle"},
	}
	var buf bytes.Buffer
	if err := writeCSV(&buf, rows); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if lines[0] != "aaa,mmm,zzz" {
		t.Errorf("header ordering: got %q, want %q", lines[0], "aaa,mmm,zzz")
	}
}

// TestWriteCSV_NilValueBecomesEmpty verifies nil values are rendered as empty fields.
func TestWriteCSV_NilValueBecomesEmpty(t *testing.T) {
	rows := []map[string]interface{}{
		{"email": "bob@example.com", "notes": nil},
	}
	var buf bytes.Buffer
	if err := writeCSV(&buf, rows); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("writeCSV: too few lines: %q", output)
	}
	// data row: "bob@example.com," or ",bob@example.com" depending on col order
	data := lines[1]
	if !strings.Contains(data, "bob@example.com") {
		t.Errorf("data row missing email: %q", data)
	}
	// nil field must be empty (two adjacent commas or trailing comma)
	if !strings.Contains(data, ",") {
		t.Errorf("data row missing comma (nil field should produce empty): %q", data)
	}
}

// TestWriteCSV_MultipleRows verifies all rows are emitted.
func TestWriteCSV_MultipleRows(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": "u-001", "name": "Alice"},
		{"id": "u-002", "name": "Bob"},
		{"id": "u-003", "name": "Carol"},
	}
	var buf bytes.Buffer
	if err := writeCSV(&buf, rows); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("writeCSV: got %d lines, want 4 (header + 3 data rows)\noutput: %q", len(lines), buf.String())
	}
}

// TestWriteCSV_IntAndBoolValues verifies numeric and boolean formatting.
func TestWriteCSV_IntAndBoolValues(t *testing.T) {
	rows := []map[string]interface{}{
		{"active": true, "count": 42},
	}
	var buf bytes.Buffer
	if err := writeCSV(&buf, rows); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "true") {
		t.Errorf("writeCSV: bool value 'true' not found in output: %q", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("writeCSV: int value '42' not found in output: %q", output)
	}
}
