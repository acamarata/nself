package ports

import (
	"strings"
	"testing"
)

// TestFormatConflictMessage_NilHolder verifies the message when holder is nil.
func TestFormatConflictMessage_NilHolder(t *testing.T) {
	msg := FormatConflictMessage(5432, nil)
	if !strings.Contains(msg, "5432") {
		t.Errorf("message should contain port number: %q", msg)
	}
	if !strings.Contains(msg, "in use") {
		t.Errorf("message should indicate port is in use: %q", msg)
	}
}

// TestFormatConflictMessage_WithHolder verifies the message includes process info.
func TestFormatConflictMessage_WithHolder(t *testing.T) {
	h := &Holder{
		PID:     1234,
		Name:    "postgres",
		Command: "postgres -D /var/lib/pgsql/data",
		IsOurs:  false,
	}
	msg := FormatConflictMessage(5432, h)
	if !strings.Contains(msg, "5432") {
		t.Errorf("message should contain port: %q", msg)
	}
	if !strings.Contains(msg, "postgres") {
		t.Errorf("message should contain process name: %q", msg)
	}
	if !strings.Contains(msg, "1234") {
		t.Errorf("message should contain PID: %q", msg)
	}
}

// TestFormatConflictMessage_IsOurs verifies the message for our own containers.
func TestFormatConflictMessage_IsOurs(t *testing.T) {
	h := &Holder{
		PID:    5678,
		Name:   "docker-proxy",
		IsOurs: true,
	}
	msg := FormatConflictMessage(3821, h)
	// Our containers should produce a specific message about nself.
	if !strings.Contains(msg, "3821") {
		t.Errorf("message should contain port: %q", msg)
	}
}

// TestFormatConflictMessage_TableDriven verifies multiple port/holder combinations.
func TestFormatConflictMessage_TableDriven(t *testing.T) {
	cases := []struct {
		name    string
		port    int
		holder  *Holder
		wantSub string
	}{
		{"nil holder", 5432, nil, "5432"},
		{"postgres", 5432, &Holder{PID: 100, Name: "postgres"}, "postgres"},
		{"redis", 6379, &Holder{PID: 200, Name: "redis-server"}, "redis-server"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := FormatConflictMessage(tc.port, tc.holder)
			if !strings.Contains(msg, tc.wantSub) {
				t.Errorf("msg %q does not contain %q", msg, tc.wantSub)
			}
		})
	}
}

// TestHolder_Fields verifies that the Holder struct fields are accessible.
func TestHolder_Fields(t *testing.T) {
	h := Holder{
		PID:     42,
		Name:    "myprocess",
		Command: "/usr/bin/myprocess --port 1234",
		IsOurs:  true,
	}
	if h.PID != 42 {
		t.Errorf("PID: got %d, want 42", h.PID)
	}
	if h.Name != "myprocess" {
		t.Errorf("Name: got %q, want myprocess", h.Name)
	}
	if !h.IsOurs {
		t.Error("IsOurs should be true")
	}
}
