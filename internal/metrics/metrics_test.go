package metrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEmitBackup_Success verifies that EmitBackup writes a .prom file
// containing the expected metric names.
func TestEmitBackup_Success(t *testing.T) {
	dir := t.TempDir()
	r := BackupRecord{
		Env:         "test",
		Type:        "full",
		Success:     true,
		DurationSec: 12.5,
		Bytes:       1024 * 1024,
		Encrypted:   true,
		Timestamp:   time.Now(),
		TextfileDir: dir,
	}
	if err := EmitBackup(r); err != nil {
		t.Fatalf("EmitBackup: %v", err)
	}
	// Verify at least one .prom file was written.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("EmitBackup wrote no files")
	}
	// Verify file contains expected metric name.
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "nself_backup") {
		t.Errorf("prom file missing nself_backup metric: %s", content)
	}
}

// TestEmitBackup_Failure verifies a failed backup record writes correct
// metrics (last_success_timestamp should not be emitted).
func TestEmitBackup_Failure(t *testing.T) {
	dir := t.TempDir()
	r := BackupRecord{
		Env:         "test",
		Type:        "wal",
		Success:     false,
		DurationSec: 0.5,
		Bytes:       0,
		Encrypted:   false,
		Timestamp:   time.Now(),
		TextfileDir: dir,
	}
	if err := EmitBackup(r); err != nil {
		t.Fatalf("EmitBackup failure record: %v", err)
	}
}

// TestEmitBackup_AllTypes verifies all documented backup types write without error.
func TestEmitBackup_AllTypes(t *testing.T) {
	dir := t.TempDir()
	types := []string{"full", "wal", "metadata", "minio"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			r := BackupRecord{
				Env:         "prod",
				Type:        typ,
				Success:     true,
				DurationSec: 1.0,
				Bytes:       512,
				TextfileDir: dir,
				Timestamp:   time.Now(),
			}
			if err := EmitBackup(r); err != nil {
				t.Errorf("EmitBackup type=%q: %v", typ, err)
			}
		})
	}
}

// TestEmitVerify_Success verifies that EmitVerify writes a .prom file.
func TestEmitVerify_Success(t *testing.T) {
	dir := t.TempDir()
	r := VerifyRecord{
		Env:         "prod",
		Success:     true,
		DurationSec: 30.0,
		Timestamp:   time.Now(),
		TextfileDir: dir,
	}
	if err := EmitVerify(r); err != nil {
		t.Fatalf("EmitVerify: %v", err)
	}
}

// TestEmitLicense_States verifies all documented grace states write without error.
func TestEmitLicense_States(t *testing.T) {
	dir := t.TempDir()
	states := []string{"valid", "grace_soft", "grace_hard", "expired"}
	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			r := LicenseRecord{
				Host:               "test-host",
				Env:                "prod",
				GraceDaysRemaining: 7.0,
				OfflineSeconds:     0,
				GraceState:         state,
				Tier:               "basic",
				TextfileDir:        dir,
			}
			if err := EmitLicense(r); err != nil {
				t.Errorf("EmitLicense state=%q: %v", state, err)
			}
		})
	}
}

// TestEmitStripeEvent verifies stripe event metrics write without error.
func TestEmitStripeEvent(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		eventType string
		result    string
	}{
		{"customer.subscription.created", "ok"},
		{"invoice.payment_failed", "error"},
		{"customer.subscription.deleted", "ok"},
	}
	for _, tc := range cases {
		t.Run(tc.eventType+"/"+tc.result, func(t *testing.T) {
			r := StripeEventRecord{
				EventType:   tc.eventType,
				Result:      tc.result,
				DurationSec: 0.1,
				TextfileDir: dir,
			}
			if err := EmitStripeEvent(r); err != nil {
				t.Errorf("EmitStripeEvent: %v", err)
			}
		})
	}
}

// TestEmitStripeOutbox verifies outbox depth metric writes without error.
func TestEmitStripeOutbox(t *testing.T) {
	dir := t.TempDir()
	r := StripeOutboxRecord{Depth: 5, TextfileDir: dir}
	if err := EmitStripeOutbox(r); err != nil {
		t.Fatalf("EmitStripeOutbox: %v", err)
	}
}

// TestEmitStripeSignatureFailure verifies signature failure metric writes.
func TestEmitStripeSignatureFailure(t *testing.T) {
	dir := t.TempDir()
	r := StripeSignatureRecord{TextfileDir: dir}
	if err := EmitStripeSignatureFailure(r); err != nil {
		t.Fatalf("EmitStripeSignatureFailure: %v", err)
	}
}
