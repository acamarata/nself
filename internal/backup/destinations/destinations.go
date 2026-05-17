// Package destinations implements remote backup destination drivers for
// nSelf's PITR (Point-in-Time Recovery) system.
//
// Remote keys follow rclone's URI scheme:
//
//	s3://bucket/path/to/object
//	r2://bucket/path/to/object
//	minio://bucket/path/to/object
//	b2://bucket/path/to/object
//	gcs://bucket/path/to/object
//	az://container/path/to/object
//
// The minio:// scheme is an nSelf convention: it is rewritten to a
// "minio" rclone remote automatically (rclone remote name "minio").
// All other schemes are passed through to rclone verbatim (scheme
// becomes the rclone remote name).
//
// rclone must be installed and configured (rclone.conf or environment
// variables) before FetchRemote is called.
package destinations

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRemoteKey reports whether key is a supported remote destination URI
// (i.e. not a local filesystem path).
func IsRemoteKey(key string) bool {
	for _, scheme := range []string{"s3://", "r2://", "minio://", "b2://", "gcs://", "az://"} {
		if strings.HasPrefix(key, scheme) {
			return true
		}
	}
	return false
}

// FetchRemote downloads the object at remoteKey to a local file in destDir.
// It returns the local file path on success.
//
// remoteKey must be a URI understood by rclone (or a minio:// URI — see
// package-level documentation for the rewrite rules).
//
// If remoteKey ends in ".age" the caller is responsible for decrypting
// the returned file; FetchRemote copies the raw (still encrypted) bytes.
func FetchRemote(ctx context.Context, remoteKey, destDir string) (string, error) {
	if remoteKey == "" {
		return "", fmt.Errorf("remote key is empty")
	}
	if destDir == "" {
		return "", fmt.Errorf("destination directory is required")
	}

	// Verify rclone is available before attempting the download.
	if _, err := exec.LookPath("rclone"); err != nil {
		return "", fmt.Errorf("rclone not found on PATH: install rclone to use remote backup destinations")
	}

	// Translate URI scheme to an rclone remote path.
	rclonePath, err := toRclonePath(remoteKey)
	if err != nil {
		return "", err
	}

	// Derive a local destination filename from the last path component.
	objectName := filepath.Base(strings.TrimSuffix(remoteKey, "/"))
	if objectName == "" || objectName == "." {
		objectName = "base.tar.gz"
	}
	localPath := filepath.Join(destDir, objectName)

	slog.Info("pitr_fetch_remote",
		"remote_key", remoteKey,
		"rclone_path", rclonePath,
		"local_path", localPath,
	)

	// Use "rclone copyto <remote> <local>" for a single-file download.
	// "copyto" preserves the exact object without rclone adding directories.
	cmd := exec.CommandContext(ctx, "rclone", "copyto", rclonePath, localPath)

	// Let rclone inherit the environment so that rclone.conf, AWS_* variables,
	// and provider-specific env vars (RCLONE_CONFIG_*) are all visible.
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return "", fmt.Errorf("rclone copyto %s: %s: %w", rclonePath, msg, err)
		}
		return "", fmt.Errorf("rclone copyto %s: %w", rclonePath, err)
	}

	// Verify the file actually arrived.
	if _, statErr := os.Stat(localPath); statErr != nil {
		return "", fmt.Errorf("rclone copyto succeeded but %s not found: %w", localPath, statErr)
	}

	slog.Info("pitr_fetch_remote_done", "local_path", localPath)
	return localPath, nil
}

// toRclonePath converts an nSelf remote key URI to an rclone path.
//
// rclone's path format is "<remote>:<bucket>/<key>" or "<remote>:<path>".
// The URI schemes map as follows:
//
//	s3://bucket/key    → "s3:bucket/key"   (standard AWS S3 / compatible)
//	r2://bucket/key    → "r2:bucket/key"   (Cloudflare R2)
//	minio://bucket/key → "minio:bucket/key" (rclone remote named "minio")
//	b2://bucket/key    → "b2:bucket/key"   (Backblaze B2)
//	gcs://bucket/key   → "gcs:bucket/key"  (Google Cloud Storage)
//	az://container/key → "az:container/key" (Azure Blob Storage)
func toRclonePath(uri string) (string, error) {
	schemes := map[string]string{
		"s3://":    "s3:",
		"r2://":    "r2:",
		"minio://": "minio:",
		"b2://":    "b2:",
		"gcs://":   "gcs:",
		"az://":    "az:",
	}

	for scheme, rclonePrefix := range schemes {
		if strings.HasPrefix(uri, scheme) {
			rest := strings.TrimPrefix(uri, scheme)
			if rest == "" {
				return "", fmt.Errorf("remote key %q has no bucket/path after scheme", uri)
			}
			return rclonePrefix + rest, nil
		}
	}

	return "", fmt.Errorf("unsupported remote key scheme in %q: supported schemes are s3://, r2://, minio://, b2://, gcs://, az://", uri)
}
