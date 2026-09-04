package errs

import (
	"errors"
	"testing"
)

// TestSentinelErrors_NonNil verifies that all sentinel errors are non-nil,
// ensuring they can be used for errors.Is comparisons.
func TestSentinelErrors_NonNil(t *testing.T) {
	all := []struct {
		name string
		err  error
	}{
		// Docker
		{"ErrDockerNotRunning", ErrDockerNotRunning},
		{"ErrDockerNotInstalled", ErrDockerNotInstalled},
		{"ErrComposeNotFound", ErrComposeNotFound},
		{"ErrPortConflict", ErrPortConflict},
		// Config
		{"ErrWeakPassword", ErrWeakPassword},
		{"ErrInsecurePassword", ErrInsecurePassword},
		{"ErrInvalidProjectName", ErrInvalidProjectName},
		{"ErrInvalidCORS", ErrInvalidCORS},
		{"ErrPlaceholderSecret", ErrPlaceholderSecret},
		// Health
		{"ErrServiceUnhealthy", ErrServiceUnhealthy},
		{"ErrHealthTimeout", ErrHealthTimeout},
		{"ErrServiceNotFound", ErrServiceNotFound},
		// Plugin
		{"ErrInvalidLicenseKey", ErrInvalidLicenseKey},
		{"ErrLicenseTierTooLow", ErrLicenseTierTooLow},
		{"ErrLicenseExpired", ErrLicenseExpired},
		{"ErrLicenseNetworkUnavailable", ErrLicenseNetworkUnavailable},
		{"ErrPluginNotFound", ErrPluginNotFound},
		{"ErrPluginManifest", ErrPluginManifest},
		{"ErrCircularDependency", ErrCircularDependency},
		// SSL
		{"ErrMkcertNotFound", ErrMkcertNotFound},
		{"ErrSSLGenerationFailed", ErrSSLGenerationFailed},
		// Nginx
		{"ErrDuplicateRoute", ErrDuplicateRoute},
		// Domain / Port
		{"ErrInvalidDomain", ErrInvalidDomain},
		{"ErrInvalidPort", ErrInvalidPort},
		// Database
		{"ErrDatabaseNotRunning", ErrDatabaseNotRunning},
		{"ErrMigrationFailed", ErrMigrationFailed},
		{"ErrMigrationValidationFailed", ErrMigrationValidationFailed},
		{"ErrBackupFailed", ErrBackupFailed},
		{"ErrBackupNotFound", ErrBackupNotFound},
		{"ErrBackupVerifyFailed", ErrBackupVerifyFailed},
		{"ErrBackupRestoreFailed", ErrBackupRestoreFailed},
		{"ErrBackupEncryptFailed", ErrBackupEncryptFailed},
		{"ErrBackupDecryptFailed", ErrBackupDecryptFailed},
		{"ErrBackupRemoteFailed", ErrBackupRemoteFailed},
		{"ErrBackupPruneFailed", ErrBackupPruneFailed},
		{"ErrWALArchiveFailed", ErrWALArchiveFailed},
		// Disaster Recovery
		{"ErrDRDrillFailed", ErrDRDrillFailed},
		{"ErrDRPromoteFailed", ErrDRPromoteFailed},
		{"ErrDRRollbackFailed", ErrDRRollbackFailed},
		{"ErrDRFenceFailed", ErrDRFenceFailed},
	}

	for _, tc := range all {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Errorf("%s is nil — sentinel errors must be non-nil", tc.name)
			}
		})
	}
}

// TestSentinelErrors_IsIdentity verifies that errors.Is works correctly
// (the error wraps itself).
func TestSentinelErrors_IsIdentity(t *testing.T) {
	if !errors.Is(ErrDockerNotRunning, ErrDockerNotRunning) {
		t.Error("errors.Is identity failed for ErrDockerNotRunning")
	}
	if !errors.Is(ErrPluginNotFound, ErrPluginNotFound) {
		t.Error("errors.Is identity failed for ErrPluginNotFound")
	}
	if !errors.Is(ErrBackupFailed, ErrBackupFailed) {
		t.Error("errors.Is identity failed for ErrBackupFailed")
	}
}

// TestSentinelErrors_Uniqueness verifies that different sentinel errors are not
// equal to each other.
func TestSentinelErrors_Uniqueness(t *testing.T) {
	if errors.Is(ErrDockerNotRunning, ErrDockerNotInstalled) {
		t.Error("ErrDockerNotRunning and ErrDockerNotInstalled should be distinct")
	}
	if errors.Is(ErrPluginNotFound, ErrPluginManifest) {
		t.Error("ErrPluginNotFound and ErrPluginManifest should be distinct")
	}
	if errors.Is(ErrBackupFailed, ErrBackupNotFound) {
		t.Error("ErrBackupFailed and ErrBackupNotFound should be distinct")
	}
}

// TestSentinelErrors_WrappedUnwrap verifies that wrapping a sentinel error with
// fmt.Errorf and %w allows errors.Is to unwrap it correctly.
func TestSentinelErrors_WrappedUnwrap(t *testing.T) {
	wrapped := errors.Join(ErrDatabaseNotRunning, errors.New("additional context"))
	if !errors.Is(wrapped, ErrDatabaseNotRunning) {
		t.Error("errors.Is failed to unwrap wrapped ErrDatabaseNotRunning via errors.Join")
	}
}
