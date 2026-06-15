package aistudio

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// cloudflaredVersion is the minimum binary version we download.
const cloudflaredVersion = "2024.11.0"

// cloudflaredBinPath returns the path where cloudflared is cached locally.
func cloudflaredBinPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	bin := filepath.Join(home, ".nself", "bin", "cloudflared")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	return bin, nil
}

// ensureCloudflared downloads cloudflared if not present, returns binary path.
func ensureCloudflared(ctx context.Context) (string, error) {
	binPath, err := cloudflaredBinPath()
	if err != nil {
		return "", err
	}

	// Already present?
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	// Create parent directory.
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		return "", fmt.Errorf("mkdir ~/.nself/bin: %w", err)
	}

	url := cloudflaredDownloadURL()
	slog.Info("downloading cloudflared", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("download cloudflared: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download cloudflared: HTTP %d", resp.StatusCode)
	}

	tmp := binPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("open tmp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("write cloudflared: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, binPath); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("install cloudflared: %w", err)
	}

	slog.Info("cloudflared installed", "path", binPath)
	return binPath, nil
}

// cloudflaredDownloadURL returns the platform-specific download URL.
func cloudflaredDownloadURL() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var osStr, archStr string
	switch goos {
	case "linux":
		osStr = "linux"
	case "darwin":
		osStr = "darwin"
	case "windows":
		osStr = "windows"
	default:
		osStr = goos
	}

	switch goarch {
	case "amd64":
		archStr = "amd64"
	case "arm64":
		archStr = "arm64"
	default:
		archStr = goarch
	}

	base := fmt.Sprintf(
		"https://github.com/cloudflare/cloudflared/releases/download/%s/cloudflared-%s-%s",
		cloudflaredVersion, osStr, archStr,
	)
	if goos == "windows" {
		return base + ".exe"
	}
	return base
}

// startTunnel starts cloudflared tunnel --url <localURL> and returns the
// public trycloudflare.com URL. The tunnel process is terminated when ctx
// is cancelled.
func startTunnel(ctx context.Context, binPath, localURL string) (string, *exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, binPath, "tunnel", "--url", localURL, "--no-autoupdate")
	cmd.Env = os.Environ()

	// cloudflared prints the tunnel URL to stderr.
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", nil, fmt.Errorf("pipe: %w", err)
	}
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return "", nil, fmt.Errorf("start cloudflared: %w", err)
	}
	pw.Close()

	// Read stderr until we see the tunnel URL (or timeout).
	urlCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		accumulated := ""
		for {
			n, readErr := pr.Read(buf)
			if n > 0 {
				accumulated += string(buf[:n])
				if idx := strings.Index(accumulated, "trycloudflare.com"); idx != -1 {
					// Find the start of the URL on the same line.
					start := strings.LastIndex(accumulated[:idx], "https://")
					if start == -1 {
						start = strings.LastIndex(accumulated[:idx], "http://")
					}
					if start != -1 {
						end := strings.IndexAny(accumulated[start:], " \t\n\r")
						if end == -1 {
							urlCh <- accumulated[start:]
						} else {
							urlCh <- accumulated[start : start+end]
						}
						pr.Close()
						return
					}
				}
			}
			if readErr != nil {
				break
			}
		}
		pr.Close()
	}()

	select {
	case u := <-urlCh:
		return strings.TrimSpace(u), cmd, nil
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return "", nil, fmt.Errorf("cloudflared tunnel URL not received within 30s")
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}
}
