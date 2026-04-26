package commands

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/nself-org/cli/internal/ui"
	"github.com/nself-org/cli/internal/version"

	"github.com/spf13/cobra"
)

// githubRelease represents the minimal fields from the GitHub releases API.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

const (
	githubLatestReleaseURL = "https://api.github.com/repos/nself-org/cli/releases/latest"
	githubReleaseByTagURL  = "https://api.github.com/repos/nself-org/cli/releases/tags/%s"
	githubDownloadBaseURL  = "https://github.com/nself-org/cli/releases/download"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the nSelf CLI and admin UI",
	Long: `Check for and install updates to the nSelf CLI binary and the
admin Docker image. Use --check to see if an update is available
without installing it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		cliOnly, _ := cmd.Flags().GetBool("cli")
		adminOnly, _ := cmd.Flags().GetBool("admin")
		force, _ := cmd.Flags().GetBool("force")
		restart, _ := cmd.Flags().GetBool("restart")
		targetVersion, _ := cmd.Flags().GetString("version")

		current := version.GetVersion()

		// Resolve the target version: explicit flag > latest from GitHub.
		var latest, releaseURL string
		if targetVersion != "" {
			tag := targetVersion
			if !strings.HasPrefix(tag, "v") {
				tag = "v" + tag
			}
			var err error
			latest, releaseURL, err = fetchReleaseByTag(tag)
			if err != nil {
				return fmt.Errorf("failed to fetch release %s: %w", targetVersion, err)
			}
		} else {
			var err error
			latest, releaseURL, err = fetchLatestRelease()
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}
		}

		upToDate := normalizeVersion(current) == normalizeVersion(latest)

		if check {
			return printCheckResult(current, latest, releaseURL, upToDate)
		}

		if upToDate && !force {
			ui.Info(fmt.Sprintf("Already on latest version (%s)", current))
			// Return a sentinel so the caller can exit non-zero.
			return &alreadyLatestError{version: current}
		}

		// Determine what to update.
		updateCLI := !adminOnly
		updateAdmin := !cliOnly

		if updateCLI {
			ui.Info(fmt.Sprintf("Updating CLI: %s -> %s", current, latest))
			if err := selfUpdate(latest); err != nil {
				return fmt.Errorf("CLI update failed: %w", err)
			}
			ui.Success(fmt.Sprintf("CLI updated: %s -> %s", current, latest))
		}

		if updateAdmin {
			ui.Info("Pulling latest admin image: nself/nself-admin:latest")
			pullCmd := exec.Command("docker", "pull", "nself/nself-admin:latest")
			pullCmd.Stdout = os.Stdout
			pullCmd.Stderr = os.Stderr
			if err := pullCmd.Run(); err != nil {
				return fmt.Errorf("pulling admin image: %w", err)
			}
			ui.Success("Admin image updated.")
		}

		if restart {
			projectDir, projErr := resolveProjectDir()
			if projErr != nil {
				ui.Warn("Cannot restart services: no nself project found in current directory")
			} else {
				ui.Info("Restarting services...")
				restartExec := exec.Command("docker", "compose", "restart")
				restartExec.Dir = projectDir
				restartExec.Stdout = os.Stdout
				restartExec.Stderr = os.Stderr
				if err := restartExec.Run(); err != nil {
					return fmt.Errorf("restarting services: %w", err)
				}
				ui.Success("Services restarted.")
			}
		}

		return nil
	},
}

// alreadyLatestError is returned (non-nil) when the binary is already on the
// latest version. Cobra will print the message and exit non-zero.
type alreadyLatestError struct{ version string }

func (e *alreadyLatestError) Error() string {
	return fmt.Sprintf("Already on latest version (%s)", e.version)
}

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
func fetchLatestRelease() (tag string, url string, err error) {
	return fetchReleaseFromURL(githubLatestReleaseURL)
}

// fetchReleaseByTag queries the GitHub API for a specific tagged release.
func fetchReleaseByTag(tag string) (string, string, error) {
	return fetchReleaseFromURL(fmt.Sprintf(githubReleaseByTagURL, tag))
}

// fetchReleaseFromURL fetches and parses a GitHub release from the given API URL.
func fetchReleaseFromURL(apiURL string) (tag string, htmlURL string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", "", fmt.Errorf("failed to parse release JSON: %w", err)
	}

	if release.TagName == "" {
		return "", "", fmt.Errorf("no tag_name in release response")
	}

	return release.TagName, release.HTMLURL, nil
}

// normalizeVersion strips a leading "v" for comparison.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// pingVersionResponse is the minimal shape we need from GET ping.nself.org/version.
type pingVersionResponse struct {
	Migration *struct {
		Available bool   `json:"available"`
		From      string `json:"from"`
		To        string `json:"to"`
		GuideURL  string `json:"guide_url"`
	} `json:"migration,omitempty"`
}

// fetchMigrationGuideURL queries ping.nself.org/version with the current CLI
// User-Agent. If the server returns a migration block (v0.9.x UA), the guide
// URL is returned. Returns an empty string on any error — callers treat it as
// "no migration info available" and proceed without blocking.
func fetchMigrationGuideURL(currentVersion string) string {
	pingURL := "https://ping.nself.org/version"
	ua := fmt.Sprintf("nself/%s", strings.TrimPrefix(currentVersion, "v"))
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, pingURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var pv pingVersionResponse
	if err := json.Unmarshal(body, &pv); err != nil {
		return ""
	}
	if pv.Migration != nil && pv.Migration.Available && pv.Migration.GuideURL != "" {
		return pv.Migration.GuideURL
	}
	return ""
}

// printCheckResult displays the version comparison result.
// For v0.9.x CLIs, it also queries ping.nself.org for a migration guide URL.
func printCheckResult(current, latest, releaseURL string, upToDate bool) error {
	fmt.Printf("%s %s\n", ui.C(ui.Bold, "Current version:"), current)
	fmt.Printf("%s %s\n", ui.C(ui.Bold, "Latest release: "), latest)

	// S60-T06: Surface migration guide URL for v0.9.x CLIs.
	normalCurrent := normalizeVersion(current)
	if strings.HasPrefix(normalCurrent, "0.") {
		if guideURL := fetchMigrationGuideURL(current); guideURL != "" {
			fmt.Println()
			ui.Warn("You are running a legacy v0.9 CLI. Migration to v1.0.9 is required.")
			ui.Info(fmt.Sprintf("Upgrade guide: %s", guideURL))
			fmt.Println()
		}
	}

	if upToDate {
		ui.Success("You are running the latest version.")
	} else {
		ui.Warn(fmt.Sprintf("Update available: %s -> %s", current, latest))
		if releaseURL != "" {
			ui.Info(fmt.Sprintf("Release notes: %s", releaseURL))
		}
		fmt.Printf("\nRun %s to update.\n", ui.C(ui.Cyan, "nself update"))
	}
	return nil
}

func init() {
	updateCmd.Flags().Bool("check", false, "Check for updates without installing")
	updateCmd.Flags().Bool("cli", false, "Only update the CLI binary")
	updateCmd.Flags().Bool("admin", false, "Only update the admin UI")
	updateCmd.Flags().Bool("force", false, "Force update even if already up to date")
	updateCmd.Flags().Bool("restart", false, "Restart services after update")
	updateCmd.Flags().String("version", "", "Download a specific version (e.g. v1.2.3)")
	RootCmd.AddCommand(updateCmd)
}
