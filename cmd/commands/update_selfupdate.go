package commands

// Purpose: The self-update entry points for `nself update` — resolving a
// binary/checksum URL pair or a release tag into a running self-replace of
// the current executable. Split out of update.go (CLI-R12) to separate the
// top-level self-update orchestration from the lower-level download/verify/
// extract primitives (update_binary_ops.go) and the GitHub/ping-API release
// lookups (update_release_api.go).
// Inputs: a binary/checksum URL pair, an expected SHA-256 sum, or a release
// tag string (empty means "latest").
// Outputs: replaces the running nself binary in place on success; returns
// *alreadyLatestError when the requested version matches the current one.
// Constraints: pure move — no behavior changes.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// selfUpdateFromURL downloads a binary directly from binaryURL, verifies its
// SHA-256 checksum using checksumURL (the URL to a checksums.txt file), and
// atomically replaces the running executable.
//
// This is the --binary-url path used by operators who need to pin to a
// specific build URL rather than always pulling the latest GitHub release.
// The same backup, checksum, and atomic-swap logic as selfUpdate applies.
func selfUpdateFromURL(binaryURL, checksumURL string) error {
	archiveName := filepath.Base(binaryURL)

	// Download checksum file.
	expectedSum, err := fetchExpectedChecksum(checksumURL, archiveName)
	if err != nil {
		return fmt.Errorf("fetching checksum for %s: %w", archiveName, err)
	}

	return selfUpdateFromURLInner(binaryURL, expectedSum)
}

// selfUpdateFromURLWithSHA is the same as selfUpdateFromURL but uses an
// operator-supplied SHA-256 digest instead of fetching a checksums.txt
// alongside the binary. Used when the mirror cannot host a checksums file
// (e.g. signed-URL mirror, air-gapped distribution channel) and the
// operator already has the digest from an out-of-band source.
func selfUpdateFromURLWithSHA(binaryURL, expectedSum string) error {
	return selfUpdateFromURLInner(binaryURL, expectedSum)
}

// selfUpdateFromURLInner is the shared implementation used by
// selfUpdateFromURL (checksums.txt path) and selfUpdateFromURLWithSHA
// (operator-supplied SHA path). It downloads, verifies, and atomically
// swaps the binary. The expectedSum must already be a 64-char lowercase
// hex digest; callers are responsible for validation.
func selfUpdateFromURLInner(binaryURL, expectedSum string) error {
	archiveName := filepath.Base(binaryURL)

	// Download binary/archive to a temp file.
	tmpFile, err := os.CreateTemp("", "nself-binary-url-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := downloadFile(binaryURL, tmpFile); err != nil {
		return fmt.Errorf("downloading %s: %w", binaryURL, err)
	}

	// Verify checksum.
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking temp file: %w", err)
	}
	if err := verifyChecksum(tmpFile, expectedSum); err != nil {
		return fmt.Errorf("checksum mismatch for downloaded binary: %w", err)
	}

	// If it looks like a tar.gz, extract the binary; otherwise treat the
	// downloaded file itself as the binary.
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking temp file for extraction: %w", err)
	}
	var tmpBinary string
	if strings.HasSuffix(strings.ToLower(archiveName), ".tar.gz") ||
		strings.HasSuffix(strings.ToLower(archiveName), ".tgz") {
		extracted, err := extractBinary(tmpFile, "nself")
		if err != nil {
			return fmt.Errorf("extracting binary from archive: %w", err)
		}
		defer os.Remove(extracted)
		tmpBinary = extracted
	} else {
		// Raw binary: copy to a new temp file so we can chmod it independently.
		rawTmp, err := os.CreateTemp("", "nself-raw-*")
		if err != nil {
			return fmt.Errorf("creating raw binary temp file: %w", err)
		}
		defer os.Remove(rawTmp.Name())
		if _, err := io.Copy(rawTmp, tmpFile); err != nil {
			rawTmp.Close()
			return fmt.Errorf("copying binary data: %w", err)
		}
		if err := rawTmp.Close(); err != nil {
			return fmt.Errorf("closing raw binary temp file: %w", err)
		}
		tmpBinary = rawTmp.Name()
	}

	// Determine the path of the running executable.
	// executablePath is the package-level hook (upgrade.go) so tests can
	// redirect the swap to a temp file rather than the real binary.
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("evaluating symlinks for executable: %w", err)
	}

	// Make the new binary executable.
	if err := os.Chmod(tmpBinary, 0755); err != nil {
		return fmt.Errorf("chmod on new binary: %w", err)
	}

	// Atomic replace (EXDEV-aware fallback, same as selfUpdate).
	if err := os.Rename(tmpBinary, exePath); err != nil {
		var linkErr *os.LinkError
		if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
			if copyErr := copyAndReplace(tmpBinary, exePath); copyErr != nil {
				return fmt.Errorf("replacing binary (copy fallback): %w", copyErr)
			}
		} else {
			return fmt.Errorf("replacing binary: %w", err)
		}
	}

	return nil
}

// selfUpdate downloads the release archive for the current OS/arch, verifies
// its SHA-256 checksum, extracts the binary, and atomically replaces the
// running executable.
func selfUpdate(tag string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	version := strings.TrimPrefix(tag, "v")
	archiveName := fmt.Sprintf("nself-%s-%s-%s.tar.gz", version, goos, goarch)
	archiveURL := fmt.Sprintf("%s/%s/%s", githubDownloadBaseURL, tag, archiveName)
	checksumURL := fmt.Sprintf("%s/%s/checksums.txt", githubDownloadBaseURL, tag)

	// Download checksum file.
	expectedSum, err := fetchExpectedChecksum(checksumURL, archiveName)
	if err != nil {
		return fmt.Errorf("fetching checksum for %s: %w", archiveName, err)
	}

	// Download archive to a temp file.
	tmpArchive, err := os.CreateTemp("", "nself-update-*.tar.gz")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpArchive.Name())
	defer tmpArchive.Close()

	if err := downloadFile(archiveURL, tmpArchive); err != nil {
		return fmt.Errorf("downloading %s: %w", archiveURL, err)
	}

	// Verify checksum.
	if _, err := tmpArchive.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking temp archive: %w", err)
	}
	if err := verifyChecksum(tmpArchive, expectedSum); err != nil {
		return fmt.Errorf("checksum mismatch for downloaded archive: %w", err)
	}

	// Extract binary from archive.
	if _, err := tmpArchive.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking temp archive for extraction: %w", err)
	}
	tmpBinary, err := extractBinary(tmpArchive, "nself")
	if err != nil {
		return fmt.Errorf("extracting binary from archive: %w", err)
	}
	defer os.Remove(tmpBinary)

	// Determine the path of the running executable.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	// Follow symlinks so we replace the real binary.
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("evaluating symlinks for executable: %w", err)
	}

	// Make the extracted binary executable.
	if err := os.Chmod(tmpBinary, 0755); err != nil {
		return fmt.Errorf("chmod on new binary: %w", err)
	}

	// Atomic replace: try os.Rename first; on cross-device (macOS /tmp vs /usr/local/bin),
	// fall back to copy-then-rename within the same directory.
	if err := os.Rename(tmpBinary, exePath); err != nil {
		var linkErr *os.LinkError
		if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
			if copyErr := copyAndReplace(tmpBinary, exePath); copyErr != nil {
				return fmt.Errorf("replacing binary (copy fallback): %w", copyErr)
			}
		} else {
			return fmt.Errorf("replacing binary: %w", err)
		}
	}

	return nil
}

// copyAndReplace copies src to a temp file in the same directory as dst, then
// renames the temp file to dst. This avoids EXDEV on cross-filesystem moves.
