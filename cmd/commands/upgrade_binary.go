package commands

// Purpose: The lower-level binary/URL helpers used by `nself upgrade`:
// detecting how the CLI was installed (homebrew/direct/system), backing up
// and rolling back the current binary around a swap, and validating a
// user-supplied --binary-url and its checksum. Split out of upgrade.go
// (CLI-R12) to separate these primitives from the channel-config helpers
// and the upgradeCmd cobra wiring that remain in upgrade.go /
// upgrade_cmd.go.
// Inputs: no arguments (detectInstallMethod), or a binary URL / expected
// SHA-256 string (the validate* functions).
// Outputs: an install-method string, a backup file path, a restored
// binary (rollbackBinary), or a validation error.
// Constraints: pure move — no behavior changes.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"
)

// detectInstallMethod returns how nself was installed: "homebrew", "direct", or "system".
func detectInstallMethod() string {
	exePath, err := os.Executable()
	if err != nil {
		return "direct"
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	// Homebrew detection: binary lives under a Homebrew prefix
	homebrewPrefixes := []string{
		"/usr/local/Cellar",
		"/opt/homebrew/Cellar",
		"/home/linuxbrew/.linuxbrew/Cellar",
	}
	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(exePath, prefix) {
			return "homebrew"
		}
	}

	// System package manager detection (apt, rpm, etc.)
	systemPaths := []string{
		"/usr/bin/nself",
		"/usr/sbin/nself",
	}
	for _, sp := range systemPaths {
		if exePath == sp {
			return "system"
		}
	}

	return "direct"
}

// backupBinary copies the current binary to ~/.nself/bin/nself.prev for rollback.
func backupBinary() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}

	backupDir := filepath.Join(home, ".nself", "bin")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup dir: %w", err)
	}

	backupPath := filepath.Join(backupDir, "nself.prev")
	src, err := os.ReadFile(exePath)
	if err != nil {
		return "", fmt.Errorf("reading current binary: %w", err)
	}
	if err := os.WriteFile(backupPath, src, 0755); err != nil {
		return "", fmt.Errorf("writing backup: %w", err)
	}

	return backupPath, nil
}

// rollbackBinary restores the previous binary from backup.
func rollbackBinary() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	backupPath := filepath.Join(home, ".nself", "bin", "nself.prev")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", backupPath)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	src, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}
	if err := os.WriteFile(exePath, src, 0755); err != nil {
		return fmt.Errorf("writing restored binary: %w", err)
	}

	ui.Success("Rolled back to previous version")
	return nil
}

// validateBinaryURL returns an error if url is not safe to use as a binary
// download target.  Only HTTPS is accepted (plain HTTP is forbidden — all
// downloads must be encrypted in transit).  The host must be on the operator
// allowlist so that an accidental or injected URL cannot redirect the binary
// swap to an arbitrary server.  The path must end in .tar.gz or .tgz so the
// extractor knows how to handle it; raw-binary downloads are rejected here
// to keep the operator-facing contract narrow.
func validateBinaryURL(url string) error {
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("--binary-url must use HTTPS (got %q)", url)
	}
	allowed := []string{
		"github.com",
		"objects.githubusercontent.com",
		"install.nself.org",
		"ping.nself.org",
	}
	// Strip the scheme, then extract the host (everything up to the first /).
	rest := strings.TrimPrefix(url, "https://")
	host := rest
	pathPart := ""
	if idx := strings.IndexByte(rest, '/'); idx != -1 {
		host = rest[:idx]
		pathPart = rest[idx:]
	}
	// Strip port if present.
	if idx := strings.LastIndexByte(host, ':'); idx != -1 {
		host = host[:idx]
	}
	hostOK := false
	for _, a := range allowed {
		if host == a || strings.HasSuffix(host, "."+a) {
			hostOK = true
			break
		}
	}
	if !hostOK {
		return fmt.Errorf(
			"--binary-url host %q is not on the allowed list; accepted hosts: %s",
			host,
			strings.Join(allowed, ", "),
		)
	}
	// Strip query string before extension check so URLs with auth tokens or
	// signed-URL params still pass.
	cleanPath := pathPart
	if idx := strings.IndexAny(cleanPath, "?#"); idx != -1 {
		cleanPath = cleanPath[:idx]
	}
	lower := strings.ToLower(cleanPath)
	if !strings.HasSuffix(lower, ".tar.gz") && !strings.HasSuffix(lower, ".tgz") {
		return fmt.Errorf(
			"--binary-url path must end in .tar.gz or .tgz (got %q)",
			pathPart,
		)
	}
	return nil
}

// validateBinarySHA256 returns nil when sum is a 64-char lowercase hex
// SHA-256 digest. Operators may pass it with or without surrounding
// whitespace; uppercase is normalized at the call site.
func validateBinarySHA256(sum string) error {
	if len(sum) != 64 {
		return fmt.Errorf("--binary-sha256 must be a 64-char hex digest (got length %d)", len(sum))
	}
	for _, r := range sum {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
		if !isHex {
			return fmt.Errorf("--binary-sha256 must be lowercase hex (got %q)", sum)
		}
	}
	return nil
}

// checksumURLFromBinaryURL derives the expected checksums.txt URL from a
// binary or archive URL.  The convention follows the goreleaser layout:
//
//	https://example.com/path/to/nself-linux-amd64          -> .../checksums.txt
//	https://example.com/path/to/v1.0.9/nself-1.0.9-linux.tar.gz -> .../checksums.txt
//
// The checksum file is expected to live in the same directory as the binary.
func checksumURLFromBinaryURL(binaryURL string) string {
	idx := strings.LastIndexByte(binaryURL, '/')
	if idx == -1 {
		return binaryURL + "/checksums.txt"
	}
	return binaryURL[:idx+1] + "checksums.txt"
}
