package gdpr

import (
	"testing"
	"time"
)

// TestRequestTypeConstants verifies the exported constants are the expected strings.
func TestRequestTypeConstants(t *testing.T) {
	tests := []struct {
		name string
		got  RequestType
		want string
	}{
		{"export", RequestTypeExport, "export"},
		{"delete", RequestTypeDelete, "delete"},
		{"restrict", RequestTypeRestrict, "restrict"},
	}
	for _, tc := range tests {
		if string(tc.got) != tc.want {
			t.Errorf("RequestType %s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestStatusConstants verifies the lifecycle state constants.
func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		got  RequestStatus
		want string
	}{
		{"pending", StatusPending, "pending"},
		{"processing", StatusProcessing, "processing"},
		{"complete", StatusComplete, "complete"},
		{"failed", StatusFailed, "failed"},
	}
	for _, tc := range tests {
		if string(tc.got) != tc.want {
			t.Errorf("RequestStatus %s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestDeadlineDays verifies the GDPR 30-day mandatory window.
func TestDeadlineDays(t *testing.T) {
	if DeadlineDays != 30 {
		t.Errorf("DeadlineDays = %d, want 30 (GDPR Art.17/20 mandate)", DeadlineDays)
	}
}

// TestAnonymizedUserID verifies deterministic pseudonymisation and prefix.
func TestAnonymizedUserID(t *testing.T) {
	tests := []struct {
		input    string
		wantHas  string
	}{
		{"user-abc123xyz", "gdpr-erased-"},
		{"short", "gdpr-erased-"},
		{"", "gdpr-erased-"},
	}
	for _, tc := range tests {
		got := anonymizedUserID(tc.input)
		if len(got) < len("gdpr-erased-") {
			t.Errorf("anonymizedUserID(%q) = %q: missing gdpr-erased- prefix", tc.input, got)
			continue
		}
		if got[:len("gdpr-erased-")] != tc.wantHas {
			t.Errorf("anonymizedUserID(%q) = %q: want prefix %q", tc.input, got, tc.wantHas)
		}
	}
}

// TestAnonymizedUserIDTruncation verifies IDs longer than 8 chars are truncated.
func TestAnonymizedUserIDTruncation(t *testing.T) {
	longID := "abcdefghijklmnop"
	got := anonymizedUserID(longID)
	want := "gdpr-erased-" + longID[:8]
	if got != want {
		t.Errorf("anonymizedUserID(%q) = %q, want %q", longID, got, want)
	}

	shortID := "abc"
	got2 := anonymizedUserID(shortID)
	want2 := "gdpr-erased-" + shortID
	if got2 != want2 {
		t.Errorf("anonymizedUserID(%q) = %q, want %q", shortID, got2, want2)
	}
}

// TestCoreUserErasureTables verifies the built-in core tables are present and valid.
func TestCoreUserErasureTables(t *testing.T) {
	tables := coreUserErasureTables()
	if len(tables) == 0 {
		t.Fatal("coreUserErasureTables: returned empty slice; expected at least 3 core tables")
	}

	required := map[string]bool{
		"np_users":         false,
		"np_user_sessions": false,
		"np_user_metadata": false,
	}
	for _, tbl := range tables {
		if tbl.Table == "" {
			t.Error("coreUserErasureTables: entry has empty Table field")
		}
		if tbl.UserCol == "" {
			t.Error("coreUserErasureTables: entry has empty UserCol field")
		}
		if _, known := required[tbl.Table]; known {
			required[tbl.Table] = true
		}
	}
	for table, found := range required {
		if !found {
			t.Errorf("coreUserErasureTables: required table %q not found", table)
		}
	}
}

// TestGDPRRequestDeadlineCalculation verifies the GDPRRequest struct holds
// a deadline that is roughly 30 days after RequestedAt (unit-level sanity check,
// does not require a DB).
func TestGDPRRequestDeadlineCalculation(t *testing.T) {
	now := time.Now()
	deadline := now.Add(DeadlineDays * 24 * time.Hour)

	r := &GDPRRequest{
		ID:          "test-id",
		RequestType: RequestTypeExport,
		SubjectType: SubjectTypeUser,
		SubjectID:   "user-123",
		RequestedAt: now,
		Deadline:    deadline,
		Status:      StatusPending,
	}

	diff := r.Deadline.Sub(r.RequestedAt)
	expectedMin := 29 * 24 * time.Hour
	expectedMax := 31 * 24 * time.Hour
	if diff < expectedMin || diff > expectedMax {
		t.Errorf("deadline diff = %v; want ~30 days (29-31 day range)", diff)
	}
}

// TestSubjectTypeConstants verifies SubjectType values.
func TestSubjectTypeConstants(t *testing.T) {
	if string(SubjectTypeUser) != "user" {
		t.Errorf("SubjectTypeUser = %q, want %q", SubjectTypeUser, "user")
	}
	if string(SubjectTypeTenant) != "tenant" {
		t.Errorf("SubjectTypeTenant = %q, want %q", SubjectTypeTenant, "tenant")
	}
}
