//go:build windows

package maintenance

import "fmt"

// HasuraMetadataCronStatus reports the state of the Hasura metadata backup timer.
type HasuraMetadataCronStatus struct {
	Enabled  bool
	Platform string
	Detail   string
}

// InstallHasuraMetadataCron is not supported on Windows.
func InstallHasuraMetadataCron() error {
	return fmt.Errorf("hasura metadata cron is not supported on Windows — use Task Scheduler manually")
}

// RemoveHasuraMetadataCron is not supported on Windows.
func RemoveHasuraMetadataCron() error {
	return fmt.Errorf("hasura metadata cron removal is not supported on Windows")
}

// GetHasuraMetadataCronStatus returns a stub status on Windows.
func GetHasuraMetadataCronStatus() HasuraMetadataCronStatus {
	return HasuraMetadataCronStatus{Platform: "windows", Enabled: false, Detail: "not supported"}
}
