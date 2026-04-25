package maintenance

import (
	"runtime"
	"testing"
)

// TestGetDiskUsageReturnsValidPercent verifies that GetDiskUsage returns a
// utilisation percentage in the range [0, 100] and non-negative GB values.
func TestGetDiskUsageReturnsValidPercent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows path not exercised here")
	}
	usage, err := GetDiskUsage()
	if err != nil {
		t.Fatalf("GetDiskUsage() error = %v", err)
	}
	if usage.UsedPercent < 0 || usage.UsedPercent > 100 {
		t.Errorf("UsedPercent = %d; want 0-100", usage.UsedPercent)
	}
	if usage.TotalGB <= 0 {
		t.Errorf("TotalGB = %.2f; want > 0", usage.TotalGB)
	}
	if usage.FreeGB < 0 {
		t.Errorf("FreeGB = %.2f; want >= 0", usage.FreeGB)
	}
	if usage.UsedGB < 0 {
		t.Errorf("UsedGB = %.2f; want >= 0", usage.UsedGB)
	}
}

// TestGetDiskUsageConsistency verifies that UsedGB + FreeGB roughly equals TotalGB.
func TestGetDiskUsageConsistency(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows path not exercised here")
	}
	usage, err := GetDiskUsage()
	if err != nil {
		t.Fatalf("GetDiskUsage() error = %v", err)
	}
	sum := usage.UsedGB + usage.FreeGB
	// Allow 5% tolerance for filesystem overhead reported differently by the kernel.
	tolerance := usage.TotalGB * 0.05
	if diff := sum - usage.TotalGB; diff < -tolerance || diff > tolerance {
		t.Errorf("UsedGB(%.2f) + FreeGB(%.2f) = %.2f, want ~%.2f (±5%%)",
			usage.UsedGB, usage.FreeGB, sum, usage.TotalGB)
	}
}

// TestScheduleStatusPlatformSet verifies that GetScheduleStatus always
// populates the Platform field.
func TestScheduleStatusPlatformSet(t *testing.T) {
	status := GetScheduleStatus()
	if status.Platform == "" {
		t.Error("GetScheduleStatus().Platform is empty; want a non-empty platform string")
	}
}

// TestRunCommandNonexistent verifies that runCommand returns an error for an
// unknown binary.
func TestRunCommandNonexistent(t *testing.T) {
	_, err := runCommand("nself-nonexistent-binary-xyz123")
	if err == nil {
		t.Error("runCommand() with nonexistent binary: expected error, got nil")
	}
}
