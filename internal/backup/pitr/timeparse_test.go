package pitr

import (
	"strings"
	"testing"
	"time"
)

func TestParseTargetTimeRFC3339(t *testing.T) {
	input := "2026-04-22T15:30:00Z"
	got, err := ParseTargetTime(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := time.Date(2026, 4, 22, 15, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseTargetTimeRelativeMinutes(t *testing.T) {
	before := time.Now().UTC()
	got, err := ParseTargetTime("30 minutes ago")
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be approximately 30 minutes in the past.
	low := before.Add(-31 * time.Minute)
	high := after.Add(-29 * time.Minute)
	if got.Before(low) || got.After(high) {
		t.Errorf("relative time %v not in expected range [%v, %v]", got, low, high)
	}
}

func TestParseTargetTimeRelativeHours(t *testing.T) {
	_, err := ParseTargetTime("2 hours ago")
	if err != nil {
		t.Fatalf("unexpected error for '2 hours ago': %v", err)
	}
}

func TestParseTargetTimeRelativeDays(t *testing.T) {
	_, err := ParseTargetTime("1 day ago")
	if err != nil {
		t.Fatalf("unexpected error for '1 day ago': %v", err)
	}
}

func TestParseTargetTimePluralUnits(t *testing.T) {
	_, err := ParseTargetTime("5 minutes ago")
	if err != nil {
		t.Fatalf("unexpected error for plural unit: %v", err)
	}
}

func TestParseTargetTimeEmpty(t *testing.T) {
	_, err := ParseTargetTime("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseTargetTimeInvalid(t *testing.T) {
	cases := []string{
		"not a time",
		"2026-04-22",        // date only (no time)
		"yesterday",         // unsupported relative form
		"foo hours ago",     // non-numeric
		"5 fortnights ago",  // unknown unit
	}
	for _, c := range cases {
		_, err := ParseTargetTime(c)
		if err == nil {
			t.Errorf("expected error for %q", c)
		}
		_ = strings.Contains(err.Error(), c) // silence unused variable
	}
}
