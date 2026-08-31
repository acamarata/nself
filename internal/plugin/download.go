package plugin

// Purpose: Plugin archive download, extraction, and install rollback helpers.
// Inputs:  context.Context, plugin name/version/repository strings, archive file path.
// Outputs: string temp file path on download; error on extraction or rollback failure.
// Constraints: Free plugins use plugins.nself.org R2 worker with GitHub Releases fallback
//              (S67-T03). Tar extraction enforces path safety (no symlinks, no traversal,
//              no setuid/setgid). Rollback is best-effort; errors are logged, not returned.
// SPORT: download/extract pipeline; callers: installLocked in installer.go

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/license"
)

// downloadPlugin fetches the plugin tarball to a temporary file.
// For paid plugins, it sends the X-License-Key header required by ping.nself.org.
// For free plugins, it tries the R2-backed worker URL first and falls back to
// GitHub Releases on 5xx responses (S67-T03).
func downloadPlugin(ctx context.Context, name, version, repository string) (string, error) {
	return downloadPluginPackage(ctx, name, version, repository, "")
}

// downloadPluginPackage fetches a plugin package, preferring a build for the
// running platform when the plugin ships a command binary.
//
// binaryName is empty for a service plugin, whose package is source and works
// everywhere. For a CLI plugin it is the command binary's name, and the package
// has to contain a build for THIS platform — a Linux binary is of no use to
// someone on macOS. Those live as per-platform release assets, so they are
// tried first, with the generic package as the fallback for a plugin whose
// release predates per-platform assets.
func downloadPluginPackage(ctx context.Context, name, version, repository, binaryName string) (string, error) {
	// Platform-specific package first, for a plugin that provides a command.
	if binaryName != "" {
		if platform, err := PlatformArch(); err == nil {
			repo := repository
			if repo == "" {
				repo = "https://github.com/nself-org/plugins"
			}
			platformURL := binaryPluginDownloadURL(strings.TrimSuffix(repo, ".git"), name, version, platform)
			if tmp, err := downloadFromURL(ctx, platformURL, nil); err == nil {
				return tmp, nil
			}
			// Fall through: no per-platform asset for this release.
		}
	}

	primaryURL := buildDownloadURL(name, version, repository)

	var extraHeaders map[string]string
	if isPaidPlugin(name) {
		// ping.nself.org/plugins/:name/download requires X-License-Key.
		// Use the first available key from env vars or stored key file.
		keys := license.CollectLicenseKeys()
		if len(keys) > 0 {
			extraHeaders = map[string]string{"X-License-Key": keys[0]}
		}
	}

	tmp, err := downloadFromURL(ctx, primaryURL, extraHeaders)
	if err == nil {
		return tmp, nil
	}

	// If not a paid plugin, attempt GitHub Releases fallback on primary failure.
	if !isPaidPlugin(name) {
		fallbackURL := buildFallbackDownloadURL(name, version, repository)
		if fallbackURL != primaryURL {
			tmp2, fallbackErr := downloadFromURL(ctx, fallbackURL, nil)
			if fallbackErr == nil {
				return tmp2, nil
			}
			return "", fmt.Errorf("download failed: primary %s: %w; fallback %s: %v", primaryURL, err, fallbackURL, fallbackErr)
		}
	}

	return "", err
}

// downloadFromURL fetches a single URL to a temp file and returns the file path.
// extraHeaders are added to the request (e.g. X-License-Key for paid plugins).
func downloadFromURL(ctx context.Context, url string, extraHeaders map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "nself-cli")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP GET %s: status %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "nself-plugin-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = tmp.Close() }()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("writing download to temp file: %w", err)
	}

	return tmp.Name(), nil
}

// buildDownloadURL constructs the tarball URL for a plugin. Pro plugins use
// the ping API download endpoint; free plugins use the plugins.nself.org worker
// which 302-redirects to R2 (primary) with GitHub Releases as fallback.
func buildDownloadURL(name, version, repository string) string {
	if isPaidPlugin(name) {
		base := pingAPIURL()
		return fmt.Sprintf("%s/plugins/%s/download", base, name)
	}
	// Free plugins: S67-T03 — use plugins.nself.org worker tarball endpoint.
	// The worker 302-redirects to R2 (primary CDN, free egress) or falls back
	// to GitHub Releases on R2 5xx. Override via NSELF_PLUGIN_REGISTRY env var.
	base := "https://plugins.nself.org"
	if envURL := os.Getenv("NSELF_PLUGIN_REGISTRY"); envURL != "" {
		base = strings.TrimRight(envURL, "/")
	}
	return fmt.Sprintf("%s/plugins/%s/tarball", base, name)
}

// buildFallbackDownloadURL constructs the GitHub Releases fallback URL for a
// free plugin. Used when the primary R2/worker download fails.
func buildFallbackDownloadURL(name, version, repository string) string {
	if repository != "" {
		repo := strings.TrimSuffix(repository, ".git")
		return fmt.Sprintf("%s/releases/download/v%s/%s-v%s.tar.gz", repo, version, name, version)
	}
	return fmt.Sprintf("https://github.com/nself-org/plugins/releases/download/v%s/%s-v%s.tar.gz", version, name, version)
}

// extractTarGz extracts a gzipped tarball into destDir.
func extractTarGz(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// Tar slip prevention: reject absolute paths and directory traversal.
		if filepath.IsAbs(hdr.Name) {
			return fmt.Errorf("unsafe tar entry: absolute path %q", hdr.Name)
		}
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("unsafe tar entry: directory traversal %q", hdr.Name)
		}
		target := filepath.Join(destDir, cleanName)
		// Final safety check: verify resolved target stays within destDir.
		if !strings.HasPrefix(target, destDir+string(filepath.Separator)) && target != destDir {
			return fmt.Errorf("tar slip detected: %q escapes destination directory", hdr.Name)
		}

		// S-014: Strip setuid, setgid, and sticky bits from tar entries
		// to prevent privilege escalation from malicious plugin archives.
		mode := os.FileMode(hdr.Mode) &^ (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)

		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("unsafe tar entry: symlinks not allowed (%q → %q)", hdr.Name, hdr.Linkname)
		case tar.TypeDir:
			if err := os.MkdirAll(target, mode); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", target, err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			_ = outFile.Close()
		}
	}

	return nil
}

// rollbackInstall cleans up a partially installed plugin by removing the
// extracted directory and dropping the database schema. Errors during
// rollback are logged but not returned.
func rollbackInstall(ctx context.Context, cfg *config.Config, name string, destDir string) {
	if err := os.RemoveAll(destDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: rollback cleanup failed for %s: %v\n", destDir, err)
	}
	if err := dropPluginSchema(ctx, cfg, name); err != nil {
		fmt.Fprintf(os.Stderr, "warning: rollback schema drop failed for %s: %v\n", name, err)
	}
}
