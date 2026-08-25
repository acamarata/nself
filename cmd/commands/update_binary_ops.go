package commands

// Purpose: Low-level file operations used by the self-update flow: an
// EXDEV-safe copy-and-replace for swapping the running binary, checksum
// fetch/verify, a plain file download, and tar.gz binary extraction. Split
// out of update.go (CLI-R12) to separate these primitives from the
// higher-level self-update orchestration (update_selfupdate.go) that calls
// them.
// Inputs: source/destination paths, a checksum URL or expected sum string,
// a download URL, and an io.Reader/io.Writer for streaming archive data.
// Outputs: a replaced binary file, a verified checksum, or an extracted
// binary's temp file path.
// Constraints: pure move — no behavior changes.

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func copyAndReplace(src, dst string) error {
	dir := filepath.Dir(dst)
	tmpDst, err := os.CreateTemp(dir, ".nself-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	defer os.Remove(tmpDst.Name())

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source binary: %w", err)
	}
	defer srcFile.Close()

	if _, err := io.Copy(tmpDst, srcFile); err != nil {
		return fmt.Errorf("copying binary data: %w", err)
	}
	if err := tmpDst.Close(); err != nil {
		return fmt.Errorf("closing temp binary: %w", err)
	}
	if err := os.Chmod(tmpDst.Name(), 0755); err != nil {
		return fmt.Errorf("chmod on temp binary: %w", err)
	}
	if err := os.Rename(tmpDst.Name(), dst); err != nil {
		return fmt.Errorf("renaming temp binary to destination: %w", err)
	}
	return nil
}

// fetchExpectedChecksum downloads a checksum manifest and returns the hex
// SHA-256 for the named file.
func fetchExpectedChecksum(checksumURL, filename string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum file returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading checksum response: %w", err)
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Standard sha256sum format: "<hash>  <filename>" or "<hash> *<filename>"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if name == filename || filepath.Base(name) == filename {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("no checksum entry found for %s", filename)
}

// downloadFile fetches url and writes the response body to dst.
func downloadFile(url string, dst io.Writer) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return fmt.Errorf("writing download data: %w", err)
	}
	return nil
}

// verifyChecksum computes the SHA-256 of r and compares it to expected (hex).
func verifyChecksum(r io.Reader, expected string) error {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return fmt.Errorf("computing SHA-256: %w", err)
	}
	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != strings.ToLower(expected) {
		return fmt.Errorf("got %s, want %s", actual, expected)
	}
	return nil
}

// extractBinary reads a gzip-compressed tar archive from r and writes the
// entry named binaryName (or ending in /binaryName) to a temp file, returning
// the temp file path.
func extractBinary(r io.Reader, binaryName string) (string, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return "", fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar entry: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Match the bare binary name regardless of directory prefix.
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}

		tmpFile, err := os.CreateTemp("", "nself-binary-*")
		if err != nil {
			return "", fmt.Errorf("creating temp file for binary: %w", err)
		}
		if _, err := io.Copy(tmpFile, tr); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("extracting binary data: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("closing extracted binary: %w", err)
		}
		return tmpFile.Name(), nil
	}

	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

// fetchLatestRelease queries the GitHub API for the latest nSelf CLI release.
