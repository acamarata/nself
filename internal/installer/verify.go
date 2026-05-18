package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nself-org/cli/internal/httptimeout"
)

// DownloadAndVerify downloads the resource at url into a private temporary
// directory (0700 owner-only permissions), computes the SHA-256 of the
// downloaded content, and compares it against expectedSHA.
//
// On success it returns the path to the verified script file inside the
// temporary directory. The caller is responsible for removing the entire
// directory with os.RemoveAll when done — the returned path's parent is the
// directory to remove.
//
// Using an owner-only 0700 parent directory (instead of the shared system
// $TMPDIR) closes the TOCTOU window between file close and script execution:
// a same-uid process cannot replace the file at the returned path because
// the containing directory is not world-writable.
//
// Verification is unconditional: expectedSHA must be a non-empty hex string.
// Pass the value from ExpectedOllamaInstallChecksum().
//
// Body reads are capped at 2 MiB via io.LimitReader. A body that exceeds
// this cap produces a checksum mismatch (the truncated bytes do not match
// the full-file hash) and DownloadAndVerify returns an error. See
// Supply-Chain.md for documentation of this behaviour.
func DownloadAndVerify(ctx context.Context, url, expectedSHA string) (string, error) {
	if expectedSHA == "" {
		return "", errf(ErrOllamaInstallFailed, "expectedSHA must not be empty: checksum pinning is required", nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errf(ErrOllamaInstallFailed, "build download request", err)
	}

	resp, err := httptimeout.Installer.Do(req)
	if err != nil {
		return "", errf(ErrOllamaInstallFailed, fmt.Sprintf("download %s", url), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errf(ErrOllamaInstallFailed,
			fmt.Sprintf("download %s: HTTP %d", url, resp.StatusCode), nil)
	}

	// Create a private 0700 directory so the script path is not inside the
	// shared system temp directory. This prevents a same-uid attacker from
	// replacing the verified file between Close and exec.
	dir, err := os.MkdirTemp("", "nself-installer-*")
	if err != nil {
		return "", errf(ErrOllamaInstallFailed, "create temp dir", err)
	}
	if err = os.Chmod(dir, 0o700); err != nil {
		os.RemoveAll(dir)
		return "", errf(ErrOllamaInstallFailed, "chmod temp dir", err)
	}

	tmpPath := dir + "/install.sh"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		os.RemoveAll(dir)
		return "", errf(ErrOllamaInstallFailed, "create temp file", err)
	}

	h := sha256.New()
	w := io.MultiWriter(tmp, h)

	if _, err = io.Copy(w, io.LimitReader(resp.Body, 2*1024*1024)); err != nil {
		tmp.Close()
		os.RemoveAll(dir)
		return "", errf(ErrOllamaInstallFailed, "write temp file", err)
	}
	if err = tmp.Close(); err != nil {
		os.RemoveAll(dir)
		return "", errf(ErrOllamaInstallFailed, "close temp file", err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedSHA {
		os.RemoveAll(dir)
		return "", errf(ErrOllamaInstallFailed,
			fmt.Sprintf("checksum mismatch: got %s want %s", got, expectedSHA), nil)
	}

	// Return the script path. The caller must os.RemoveAll(filepath.Dir(tmpPath))
	// when done — the parent directory is the cleanup target.
	return tmpPath, nil
}
