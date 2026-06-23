package plugin

// Purpose: Rust/binary plugin install — arch detection, tarball download from
//          GitHub Releases, SHA256 checksum verification, extraction, and chmod.
//          Called from installLocked when manifest.Language == "rust" (binary plugin).
//          Replaces the legacy plugin_install.sh + runtime.sh shell script flow.
// Inputs:  context.Context, *PluginManifest (name/version/checksum/repository),
//          pluginDir path string.
// Outputs: error on any failure; on success the plugin binary is at
//          {pluginDir}/{name}/{name}[.exe] with mode 0755.
// Constraints: Binary is quarantined (removed) on checksum failure.
//              Arch must be one of the 5 entries in archTable or install fails.
//              All HTTP is done via the shared downloadFromURL helper.
// SPORT: binary install path; callers: installLocked in installer.go

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// binaryReleaseURL constructs the GitHub Releases download URL for a binary
// plugin asset. The URL pattern matches the Rust plugin release convention:
//
//	{repository}/releases/download/v{version}/{name}-v{version}-{arch}.tar.gz
//
// The repository field in the manifest is used when present; otherwise the
// default nself-org/plugins-pro GitHub org is assumed.
func binaryReleaseURL(name, version, repository, arch string) string {
	repoBase := repository
	if repoBase == "" {
		repoBase = fmt.Sprintf("https://github.com/nself-org/plugins-pro")
	}
	repoBase = strings.TrimSuffix(repoBase, ".git")
	return fmt.Sprintf("%s/releases/download/v%s/%s-v%s-%s.tar.gz",
		repoBase, version, name, version, arch)
}

// verifySHA256 computes the SHA-256 digest of the file at path and compares it
// to expectedHex (lowercase hex). Returns an error when they do not match.
func verifySHA256(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing file: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != strings.ToLower(expectedHex) {
		return fmt.Errorf("SHA-256 mismatch: got %s, expected %s", got, expectedHex)
	}
	return nil
}

// binaryName returns the plugin binary file name, adding .exe on Windows.
func binaryName(pluginName string) string {
	if runtime.GOOS == "windows" {
		return pluginName + ".exe"
	}
	return pluginName
}

// extractBinaryTarGz extracts a gzipped tarball at archivePath into destDir.
// It enforces path safety (no symlinks, no traversal) and makes the plugin
// binary executable (mode 0755).
func extractBinaryTarGz(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating plugin binary directory: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening binary archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// Safety: reject symlinks and path traversal.
		if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
			return fmt.Errorf("binary archive contains a symlink — rejected for security: %s", hdr.Name)
		}
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("binary archive path traversal detected: %s", hdr.Name)
		}

		destPath := filepath.Join(destDir, cleanName)

		// Ensure the destination is still under destDir.
		if !strings.HasPrefix(destPath+string(filepath.Separator), destDir+string(filepath.Separator)) {
			return fmt.Errorf("binary archive path escape detected: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("creating directory %s: %w", destPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", destPath, err)
			}
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("creating %s: %w", destPath, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("writing %s: %w", destPath, err)
			}
			out.Close()
		}
	}
	return nil
}

// chmodBinary makes the plugin binary at {destDir}/{binaryName} executable.
// Returns an error if the binary file is not present after extraction.
func chmodBinary(destDir, pluginName string) error {
	binPath := filepath.Join(destDir, binaryName(pluginName))
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return fmt.Errorf("binary %s not found in archive after extraction", binaryName(pluginName))
	}
	if err := os.Chmod(binPath, 0o755); err != nil {
		return fmt.Errorf("chmod +x %s: %w", binPath, err)
	}
	return nil
}

// installBinaryPlugin downloads, verifies, and extracts a Rust/binary plugin.
// It is called from installLocked when manifest.Language == "rust".
// On checksum failure the downloaded archive is removed and an error is returned.
func installBinaryPlugin(ctx context.Context, manifest *PluginManifest, pluginDir string) error {
	// Step 1: Detect the running platform arch.
	arch, err := currentArch()
	if err != nil {
		return fmt.Errorf("binary plugin install for %q: %w", manifest.Name, err)
	}

	// Step 2: Build the download URL.
	url := binaryReleaseURL(manifest.Name, manifest.Version, manifest.Repository, arch)

	// Step 3: Download the tarball to a temp file.
	archivePath, err := downloadFromURL(ctx, url)
	if err != nil {
		return fmt.Errorf("downloading binary plugin %q (%s): %w", manifest.Name, arch, err)
	}
	// quarantine on any failure after this point
	defer os.Remove(archivePath)

	// Step 4: Verify SHA-256 checksum. For stable binary plugins a missing
	// checksum is a hard error (mirrors the source plugin verifyChecksum policy).
	if manifest.Checksum == "" {
		if manifest.PublishStatus == "stable" || manifest.PublishStatus == "" {
			return fmt.Errorf("no checksum in registry for stable binary plugin %q — refusing to install", manifest.Name)
		}
		fmt.Fprintf(os.Stderr, "warning: no checksum in registry for binary plugin %q, skipping verification\n", manifest.Name)
	} else {
		if err := verifySHA256(archivePath, manifest.Checksum); err != nil {
			// quarantine: archivePath is removed by deferred os.Remove above.
			return fmt.Errorf("checksum verification for binary plugin %q: %w", manifest.Name, err)
		}
	}

	// Step 5: Extract to {pluginDir}/{name}/.
	destDir := filepath.Join(pluginDir, manifest.Name)
	if err := extractBinaryTarGz(archivePath, destDir); err != nil {
		// Remove partially extracted dir on failure.
		os.RemoveAll(destDir)
		return fmt.Errorf("extracting binary plugin %q: %w", manifest.Name, err)
	}

	// Step 6: chmod +x the plugin binary.
	if err := chmodBinary(destDir, manifest.Name); err != nil {
		os.RemoveAll(destDir)
		return fmt.Errorf("setting binary plugin %q executable: %w", manifest.Name, err)
	}

	return nil
}
