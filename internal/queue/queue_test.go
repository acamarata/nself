package queue

import (
	"strings"
	"testing"
)

// TestValidateQueueName_Valid verifies well-formed queue names pass validation.
func TestValidateQueueName_Valid(t *testing.T) {
	valid := []string{
		"my-queue",
		"my_queue",
		"my.queue",
		"MyQueue",
		"queue123",
		"a",
		strings.Repeat("a", 128),
	}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if err := validateQueueName(name); err != nil {
				t.Errorf("validateQueueName(%q): unexpected error: %v", name, err)
			}
		})
	}
}

// TestValidateQueueName_Invalid verifies malformed queue names are rejected.
func TestValidateQueueName_Invalid(t *testing.T) {
	invalid := []string{
		"",                      // empty
		strings.Repeat("a", 129), // too long
		"has space",             // space
		"has/slash",             // slash
		"has$dollar",            // dollar sign
		"inject'; DROP TABLE",   // SQL injection attempt
	}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := validateQueueName(name); err == nil {
				t.Errorf("validateQueueName(%q): expected error, got nil", name)
			}
		})
	}
}

// TestValidateJobID_Valid verifies well-formed job IDs pass validation.
func TestValidateJobID_Valid(t *testing.T) {
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"job-abc-123",
		"ABC123",
		"a",
	}
	for _, id := range valid {
		t.Run(id, func(t *testing.T) {
			if err := validateJobID(id); err != nil {
				t.Errorf("validateJobID(%q): unexpected error: %v", id, err)
			}
		})
	}
}

// TestValidateJobID_Invalid verifies malformed job IDs are rejected.
func TestValidateJobID_Invalid(t *testing.T) {
	invalid := []string{
		"",
		strings.Repeat("a", 65), // too long
		"has space",
		"has/slash",
		"sql'injection",
	}
	for _, id := range invalid {
		t.Run(id, func(t *testing.T) {
			if err := validateJobID(id); err == nil {
				t.Errorf("validateJobID(%q): expected error, got nil", id)
			}
		})
	}
}
