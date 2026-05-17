package compose

import (
	"strings"
	"testing"
)

// service_toggle tests: compose-aware plugin enable/disable.

const sampleCompose = `services:
  postgres:
    image: postgres:16
    ports:
      - "5432:5432"
  plugin-claw:
    image: nself/claw:latest
    environment:
      - FOO=bar
  hasura:
    image: hasura/graphql-engine:latest

networks:
  default:
    driver: bridge
`

func TestToggleServiceDisable(t *testing.T) {
	out, changed, err := ToggleService(sampleCompose, "plugin-claw", false)
	if err != nil {
		t.Fatalf("ToggleService disable: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true on first disable")
	}
	if !strings.Contains(out, "# nself-disabled: plugin-claw") {
		t.Errorf("expected disabled marker in output, got:\n%s", out)
	}
	if !strings.Contains(out, "#   plugin-claw:") {
		t.Errorf("expected commented service header, got:\n%s", out)
	}
	// Surrounding services must remain intact and uncommented.
	if !strings.Contains(out, "  postgres:") {
		t.Errorf("postgres service was unexpectedly modified:\n%s", out)
	}
	if !strings.Contains(out, "  hasura:") {
		t.Errorf("hasura service was unexpectedly modified:\n%s", out)
	}
}

func TestToggleServiceEnable(t *testing.T) {
	disabled, _, _ := ToggleService(sampleCompose, "plugin-claw", false)
	out, changed, err := ToggleService(disabled, "plugin-claw", true)
	if err != nil {
		t.Fatalf("ToggleService enable: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true on enable")
	}
	if strings.Contains(out, "# nself-disabled: plugin-claw") {
		t.Errorf("expected marker removed after enable, got:\n%s", out)
	}
	if strings.Contains(out, "#   plugin-claw:") {
		t.Errorf("expected service uncommented, got:\n%s", out)
	}
	if !strings.Contains(out, "  plugin-claw:") {
		t.Errorf("expected uncommented plugin-claw service, got:\n%s", out)
	}
}

func TestToggleServiceIdempotent(t *testing.T) {
	// Disabling twice should not double-comment or duplicate markers.
	once, _, _ := ToggleService(sampleCompose, "plugin-claw", false)
	twice, changed, err := ToggleService(once, "plugin-claw", false)
	if err != nil {
		t.Fatalf("second disable: %v", err)
	}
	if changed {
		t.Error("second disable should not report a change")
	}
	if once != twice {
		t.Error("second disable should be a no-op")
	}

	// Enabling an already-enabled service is also a no-op.
	out, changed, err := ToggleService(sampleCompose, "plugin-claw", true)
	if err != nil {
		t.Fatalf("enable when already enabled: %v", err)
	}
	if changed {
		t.Error("enable on already-enabled should not report a change")
	}
	if out != sampleCompose {
		t.Error("enable on already-enabled should be a no-op")
	}
}

func TestToggleServiceMissingService(t *testing.T) {
	out, changed, err := ToggleService(sampleCompose, "nonexistent-plugin", false)
	if err != nil {
		t.Fatalf("ToggleService for missing service: %v", err)
	}
	if changed {
		t.Error("expected changed=false for missing service")
	}
	if out != sampleCompose {
		t.Error("expected unchanged output for missing service")
	}
}

func TestToggleServiceEmptyName(t *testing.T) {
	if _, _, err := ToggleService(sampleCompose, "", false); err == nil {
		t.Error("expected error on empty service name")
	}
}

func TestRoundTripDisableEnable(t *testing.T) {
	disabled, _, _ := ToggleService(sampleCompose, "plugin-claw", false)
	enabled, _, _ := ToggleService(disabled, "plugin-claw", true)
	if enabled != sampleCompose {
		t.Errorf("disable -> enable did not round-trip\nGot:\n%s\nWant:\n%s", enabled, sampleCompose)
	}
}

func TestIsTwoSpaceServiceHeader(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"  postgres:", true},
		{"  hasura:", true},
		{"   postgres:", false},     // three spaces — too deep
		{"postgres:", false},        // no indent — top-level key
		{"  postgres: foo", false},  // value present — not a header
		{"  # comment", false},      // comment line
		{"    image: nginx", false}, // four-space sub-key
		{"", false},
	}
	for _, tt := range tests {
		got := isTwoSpaceServiceHeader(tt.in)
		if got != tt.want {
			t.Errorf("isTwoSpaceServiceHeader(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
