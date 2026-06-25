package plugin

// arch.go — Platform architecture detection for binary plugin installs.
//
// Purpose: Map Go's runtime.GOOS/GOARCH values to the platform strings used in
//          binary plugin release archives, enabling nself plugin install to
//          select the correct platform tarball without the shell scripts
//          (plugin_install.sh / runtime.sh). Port of runtime.sh arch detection.
// Inputs:  none (reads runtime.GOOS + runtime.GOARCH at startup)
// Outputs: platform string e.g. "darwin-arm64"; error if platform unsupported
// Constraints: Must match the archive naming used by plugin CI (cross-compiled
//              via goreleaser or Cargo cross — see plugins-pro/Makefile).
// SPORT: F02 nself plugin install — binary plugin support (P4-E5-W2-S03-T12-B)

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// PlatformArch returns the platform string for binary plugin archive selection.
// Supported values: darwin-arm64, darwin-amd64, linux-amd64, linux-arm64,
// windows-amd64. Any other GOOS/GOARCH pair returns an error.
func PlatformArch() (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "darwin":
		switch goarch {
		case "arm64":
			return "darwin-arm64", nil
		case "amd64":
			return "darwin-amd64", nil
		}
	case "linux":
		switch goarch {
		case "amd64":
			return "linux-amd64", nil
		case "arm64":
			return "linux-arm64", nil
		}
	case "windows":
		switch goarch {
		case "amd64":
			return "windows-amd64", nil
		}
	}
	return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
}

// binaryPluginDownloadURL builds the GitHub Releases download URL for a
// binary plugin using the platform-arch string.
// URL format: https://github.com/<org>/<repo>/releases/download/v<version>/<name>-<version>-<platform>.tar.gz
func binaryPluginDownloadURL(repository, name, version, platform string) string {
	return fmt.Sprintf(
		"%s/releases/download/v%s/%s-%s-%s.tar.gz",
		repository, version, name, version, platform,
	)
}

// downloadBinaryPlugin fetches a platform-specific binary plugin tarball,
// verifies its SHA-256 checksum, extracts it to destDir, and chmods the binary
// to 0755. If checksum verification fails the partially-extracted content is
// removed and an error is returned.
//
// checksumURL is optional; if empty, checksum verification is skipped (not
// recommended — only acceptable for dev/testing).
func downloadBinaryPlugin(name, version, platform, archiveURL, checksumURL, destDir string) error {
	// Download archive to temp file.
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download binary plugin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download binary plugin: server returned %d for %s", resp.StatusCode, archiveURL)
	}

	tmp, err := os.CreateTemp("", "nself-plugin-binary-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("write download: %w", err)
	}
	tmp.Close()
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Verify checksum if provided.
	if checksumURL != "" {
		expected, err := fetchExpectedChecksum(client, checksumURL, filepath.Base(archiveURL))
		if err != nil {
			return fmt.Errorf("fetch checksum: %w", err)
		}
		if actualChecksum != expected {
			return fmt.Errorf(
				"binary plugin checksum mismatch for %s-%s-%s: got %s, want %s",
				name, version, platform, actualChecksum, expected,
			)
		}
	}

	// Extract tarball to destDir.
	if err := extractTarGz(tmp.Name(), destDir); err != nil {
		os.RemoveAll(destDir)
		return fmt.Errorf("extract binary plugin: %w", err)
	}

	// chmod +x the binary inside destDir.
	binary := filepath.Join(destDir, name)
	if err := os.Chmod(binary, 0755); err != nil {
		// Tolerate missing exact name — some plugins ship as <name>-<platform>.
		_ = err
	}

	return nil
}

// fetchExpectedChecksum retrieves checksums.txt from checksumURL and extracts
// the SHA-256 for the named file (matching the format produced by goreleaser:
// "<sha256>  <filename>").
func fetchExpectedChecksum(client *http.Client, checksumURL, filename string) (string, error) {
	resp, err := client.Get(checksumURL)
	if err != nil {
		return "", fmt.Errorf("fetch checksums.txt: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums.txt returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read checksums.txt: %w", err)
	}

	for _, line := range splitLines(string(data)) {
		// Format: "<sha256>  <filename>"
		if len(line) < 66 {
			continue
		}
		sha := line[:64]
		rest := line[64:]
		// Trim leading whitespace from rest.
		for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
			rest = rest[1:]
		}
		if rest == filename {
			return sha, nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s in checksums.txt", filename)
}

// splitLines splits a string by newlines (tolerates CRLF and LF).
func splitLines(s string) []string {
	var lines []string
	for len(s) > 0 {
		idx := -1
		for i, c := range s {
			if c == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			lines = append(lines, s)
			break
		}
		line := s[:idx]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
		s = s[idx+1:]
	}
	return lines
}
