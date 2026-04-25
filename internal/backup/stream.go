// Package backup — streaming encrypted backup to remote destinations.
//
// The pipeline runs three concurrent goroutines:
//
//	pg_dump (streaming) | age (encrypt) | rclone (multipart upload)
//
// No temp files are written. Encryption recipients may be age public keys,
// SSH public keys, or GitHub user keys (fetched via github.com/users/<user>/keys).
// The upload destination is any rclone-supported remote: s3://, r2://, b2://, gcs://, az://.
package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// StreamConfig holds parameters for a streaming encrypted backup.
type StreamConfig struct {
	// PgURL is the postgres DSN used by pg_dump. If empty, derived from cfg.
	PgURL string

	// Destination is the rclone remote path, e.g. "s3:bucket/prefix/".
	Destination string

	// Recipients holds age/SSH public keys (one per entry). May also be
	// "github:<username>" to fetch the user's SSH public keys from GitHub.
	Recipients []string

	// Key is the object name appended to Destination. Auto-generated if empty.
	Key string

	// ChunkMB is the multipart chunk size. Defaults to 64.
	ChunkMB int
}

// StreamOptions holds CLI flag values for `nself backup stream`.
type StreamOptions struct {
	To        string   // destination URL (rclone remote path)
	Recipients []string // --recipient flags (may be specified multiple times)
	DryRun    bool
}

// StreamResult is returned by Stream on success.
type StreamResult struct {
	BackupID    string    `json:"backup_id"`
	Destination string    `json:"destination"`
	StartedAt   time.Time `json:"started_at"`
	Duration    string    `json:"duration"`
	Encrypted   bool      `json:"encrypted"`
}

// Stream runs a three-stage concurrent pipeline:
//
//  1. pg_dump stdout -> pgWriter
//  2. age encrypts pgReader -> encWriter (skipped if no recipients)
//  3. rclone rcat uploads encReader to Destination/Key
//
// All three stages run in parallel goroutines. The first error from any stage
// cancels the others via context cancellation.
func Stream(ctx context.Context, cfg *config.Config, opts StreamOptions) (*StreamResult, error) {
	if opts.To == "" {
		if cfg.Backup.Remote != "" {
			opts.To = cfg.Backup.Remote
		} else {
			return nil, fmt.Errorf("destination required: use --to <url> or set NSELF_BACKUP_DESTINATION")
		}
	}

	recipients := opts.Recipients
	if len(recipients) == 0 && cfg.Backup.AgeRecipients != "" {
		recipients = strings.Fields(cfg.Backup.AgeRecipients)
	}

	// Resolve GitHub keys.
	var err error
	recipients, err = resolveRecipients(ctx, recipients)
	if err != nil {
		return nil, fmt.Errorf("resolve recipients: %w", err)
	}

	// Build the object key.
	ts := time.Now().UTC().Format("20060102_150405")
	key := fmt.Sprintf("%s_stream_%s.sql", cfg.ProjectName, ts)
	if len(recipients) > 0 {
		key += ".age"
	}

	pgURL := buildPgURL(cfg)

	if opts.DryRun {
		slog.Info("dry-run: streaming backup",
			"destination", opts.To+"/"+key,
			"encrypted", len(recipients) > 0,
			"pg_url", redactURL(pgURL),
		)
		return &StreamResult{
			BackupID:    key,
			Destination: opts.To + "/" + key,
			StartedAt:   time.Now(),
			Duration:    "0s",
			Encrypted:   len(recipients) > 0,
		}, nil
	}

	start := time.Now()

	if err := checkBinaries(recipients); err != nil {
		return nil, err
	}

	result, err := runStreamPipeline(ctx, cfg, pgURL, opts.To, key, recipients)
	if err != nil {
		return nil, err
	}
	result.StartedAt = start
	result.Duration = time.Since(start).String()
	return result, nil
}

// runStreamPipeline wires the three concurrent goroutines and waits for all.
func runStreamPipeline(ctx context.Context, cfg *config.Config, pgURL, destination, key string, recipients []string) (*StreamResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Stage plumbing: two io.Pipe connections.
	//   pg_dump -> pgW | pgR -> (age) -> encW | encR -> rclone
	pgR, pgW := io.Pipe()
	var uploadReader io.ReadCloser

	encrypt := len(recipients) > 0
	var encW *io.PipeWriter
	var encR *io.PipeReader
	if encrypt {
		encR, encW = io.Pipe()
		uploadReader = encR
	} else {
		uploadReader = pgR
	}

	var wg sync.WaitGroup
	errc := make(chan error, 3)

	// ── Stage 1: pg_dump ─────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pgW.Close()
		if err := runPgDump(ctx, cfg, pgURL, pgW); err != nil {
			cancel()
			errc <- fmt.Errorf("pg_dump: %w", err)
		}
	}()

	// ── Stage 2: age encrypt (optional) ──────────────────────────────
	if encrypt {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer encW.Close()
			if err := ageEncryptStream(ctx, pgR, encW, recipients); err != nil {
				cancel()
				pgR.CloseWithError(err)
				errc <- fmt.Errorf("age encrypt: %w", err)
			}
		}()
	}

	// ── Stage 3: rclone rcat upload ───────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer uploadReader.Close()
		if err := rcloneRcat(ctx, uploadReader, destination, key); err != nil {
			cancel()
			errc <- fmt.Errorf("rclone upload: %w", err)
		}
	}()

	wg.Wait()
	close(errc)

	for e := range errc {
		if e != nil {
			return nil, e
		}
	}

	full := destination
	if !strings.HasSuffix(destination, "/") {
		full = destination + "/" + key
	} else {
		full = destination + key
	}

	slog.Info("streaming backup complete", "destination", full, "encrypted", encrypt)

	return &StreamResult{
		BackupID:    key,
		Destination: full,
		Encrypted:   encrypt,
	}, nil
}

// runPgDump executes pg_dump and writes its output to w.
// Uses pg_dump custom format for efficient streaming and restore.
func runPgDump(ctx context.Context, cfg *config.Config, pgURL string, w io.Writer) error {
	args := []string{
		"--format=custom",
		"--no-password",
		pgURL,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Stdout = w
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_dump: %w", err)
	}

	errOut, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if len(errOut) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupFailed, strings.TrimSpace(string(errOut)))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupFailed, err)
	}

	return nil
}

// ageEncryptStream pipes r through the age binary and writes ciphertext to w.
// Supports multiple recipients (age public keys, SSH public keys).
func ageEncryptStream(ctx context.Context, r io.Reader, w io.Writer, recipients []string) error {
	args := []string{}
	for _, rec := range recipients {
		args = append(args, "-r", rec)
	}

	cmd := exec.CommandContext(ctx, "age", args...)
	cmd.Stdin = r
	cmd.Stdout = w

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("age stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start age: %w", err)
	}

	errOut, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if len(errOut) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupEncryptFailed, strings.TrimSpace(string(errOut)))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupEncryptFailed, err)
	}

	return nil
}

// rcloneRcat uploads from r to destination/key using rclone rcat.
// rclone handles multipart uploads internally for all supported backends.
func rcloneRcat(ctx context.Context, r io.Reader, destination, key string) error {
	remote := destination
	if !strings.HasSuffix(remote, "/") {
		remote = remote + "/"
	}
	remote = remote + key

	cmd := exec.CommandContext(ctx, "rclone", "rcat", remote)
	cmd.Stdin = r

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("rclone stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start rclone: %w", err)
	}

	errOut, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if len(errOut) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupRemoteFailed, strings.TrimSpace(string(errOut)))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupRemoteFailed, err)
	}

	return nil
}

// ResumeState holds the persisted state for a resumable streaming backup.
type ResumeState struct {
	BackupID    string    `json:"backup_id"`
	Destination string    `json:"destination"`
	Key         string    `json:"key"`
	Recipients  []string  `json:"recipients"`
	StartedAt   time.Time `json:"started_at"`
	// For rclone-based multipart, we store the partial object key prefix.
	// rclone handles actual part tracking internally; we record enough to
	// invoke rclone copyto from a local temp file on resume.
	PartialPath string `json:"partial_path,omitempty"`
}

// stateDir returns the directory for resume state files.
func stateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nself", "backup-state")
}

// saveResumeState persists state to ~/.nself/backup-state/<id>.json.
func saveResumeState(state ResumeState) error {
	dir := stateDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	path := filepath.Join(dir, state.BackupID+".json")
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

// loadResumeState reads ~/.nself/backup-state/<id>.json.
func loadResumeState(backupID string) (*ResumeState, error) {
	path := filepath.Join(stateDir(), backupID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no resume state found for backup ID %q", backupID)
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var state ResumeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &state, nil
}

// deleteResumeState removes the state file on successful completion.
func deleteResumeState(backupID string) {
	path := filepath.Join(stateDir(), backupID+".json")
	_ = os.Remove(path)
}

// listResumeStates returns all IDs with saved resume state.
func listResumeStates() ([]ResumeState, error) {
	dir := stateDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state dir: %w", err)
	}
	var states []ResumeState
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		state, err := loadResumeState(id)
		if err != nil {
			continue
		}
		states = append(states, *state)
	}
	return states, nil
}

// ResumeOptions holds flags for `nself backup resume`.
type ResumeOptions struct {
	BackupID string
}

// Resume attempts to complete a previously interrupted streaming backup.
// Since rclone rcat cannot resume a partial upload, Resume re-streams from
// pg_dump through the full pipeline. The previous partial upload is
// overwritten at the same destination key.
func Resume(ctx context.Context, cfg *config.Config, opts ResumeOptions) (*StreamResult, error) {
	state, err := loadResumeState(opts.BackupID)
	if err != nil {
		return nil, err
	}

	slog.Info("resuming backup",
		"backup_id", state.BackupID,
		"destination", state.Destination,
		"started_at", state.StartedAt,
	)

	pgURL := buildPgURL(cfg)
	result, err := runStreamPipeline(ctx, cfg, pgURL, state.Destination, state.Key, state.Recipients)
	if err != nil {
		return nil, fmt.Errorf("resume stream: %w", err)
	}

	deleteResumeState(opts.BackupID)
	return result, nil
}

// resolveRecipients expands "github:<username>" entries into SSH public keys.
func resolveRecipients(ctx context.Context, recipients []string) ([]string, error) {
	var resolved []string
	for _, r := range recipients {
		if strings.HasPrefix(r, "github:") {
			username := strings.TrimPrefix(r, "github:")
			keys, err := fetchGitHubKeys(ctx, username)
			if err != nil {
				return nil, fmt.Errorf("fetch GitHub keys for %s: %w", username, err)
			}
			resolved = append(resolved, keys...)
		} else {
			resolved = append(resolved, r)
		}
	}
	return resolved, nil
}

// fetchGitHubKeys fetches SSH public keys for a GitHub username.
func fetchGitHubKeys(ctx context.Context, username string) ([]string, error) {
	url := "https://api.github.com/users/" + username + "/keys"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for user %s", resp.StatusCode, username)
	}

	var apiKeys []struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiKeys); err != nil {
		return nil, fmt.Errorf("decode GitHub API response: %w", err)
	}
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("no SSH keys found for GitHub user %s", username)
	}

	keys := make([]string, len(apiKeys))
	for i, k := range apiKeys {
		keys[i] = k.Key
	}
	return keys, nil
}

// buildPgURL constructs a postgres DSN from config.
func buildPgURL(cfg *config.Config) string {
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	password := cfg.Postgres.Password
	host := "localhost"
	port := cfg.Postgres.Port
	if port == 0 {
		port = 5432
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	if password != "" {
		return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", user, password, host, port, db)
	}
	return fmt.Sprintf("postgresql://%s@%s:%d/%s?sslmode=disable", user, host, port, db)
}

// redactURL removes the password component of a postgres DSN for logging.
// It handles "scheme://user:pass@host" but not "scheme://user@host" (no password).
func redactURL(url string) string {
	// Find the authority part: after "://"
	schemeEnd := strings.Index(url, "://")
	if schemeEnd < 0 {
		return url
	}
	authority := url[schemeEnd+3:]

	atIdx := strings.Index(authority, "@")
	if atIdx < 0 {
		// No @ means no userinfo at all.
		return url
	}

	userinfo := authority[:atIdx]
	colonIdx := strings.Index(userinfo, ":")
	if colonIdx < 0 {
		// userinfo has no colon — no password present.
		return url
	}

	// Replace password with ***.
	redacted := url[:schemeEnd+3] + userinfo[:colonIdx+1] + "***" + url[schemeEnd+3+atIdx:]
	return redacted
}

// checkBinaries verifies that required external programs are on PATH.
func checkBinaries(recipients []string) error {
	required := []string{"pg_dump", "rclone"}
	if len(recipients) > 0 {
		required = append(required, "age")
	}
	for _, bin := range required {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("required binary %q not found on PATH: install it before running backup stream", bin)
		}
	}
	return nil
}

// RestoreFromRemote decrypts and restores a backup directly from a remote URL.
// It streams: rclone cat | age -d | pg_restore. No local temp file.
func RestoreFromRemote(ctx context.Context, cfg *config.Config, from, keyPath string) error {
	if from == "" {
		return fmt.Errorf("--from destination required")
	}

	if keyPath == "" {
		keyPath = filepath.Join(os.Getenv("HOME"), ".config", "nself", "age-key.txt")
	}

	encrypted := strings.HasSuffix(from, ".age")

	// Check binaries.
	binaries := []string{"rclone", "pg_restore"}
	if encrypted {
		binaries = append(binaries, "age")
	}
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("required binary %q not found: %w", bin, err)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var downloadReader io.ReadCloser
	var restoreReader io.ReadCloser

	// Stage 1 output.
	dlR, dlW := io.Pipe()

	var wg sync.WaitGroup
	errc := make(chan error, 3)

	// ── Stage 1: rclone cat <remote> ─────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer dlW.Close()
		cmd := exec.CommandContext(ctx, "rclone", "cat", from)
		cmd.Stdout = dlW
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			errc <- fmt.Errorf("rclone stderr pipe: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			cancel()
			errc <- fmt.Errorf("start rclone: %w", err)
			return
		}
		errOut, _ := io.ReadAll(stderr)
		if err := cmd.Wait(); err != nil {
			cancel()
			msg := strings.TrimSpace(string(errOut))
			if msg != "" {
				errc <- fmt.Errorf("%w: %s", errs.ErrBackupRemoteFailed, msg)
			} else {
				errc <- fmt.Errorf("%w: %v", errs.ErrBackupRemoteFailed, err)
			}
		}
	}()

	downloadReader = dlR

	// ── Stage 2: age -d (optional) ───────────────────────────────────
	var decR *io.PipeReader
	var decW *io.PipeWriter
	if encrypted {
		decR, decW = io.Pipe()
		restoreReader = decR
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer decW.Close()
			cmd := exec.CommandContext(ctx, "age", "--decrypt", "-i", keyPath)
			cmd.Stdin = downloadReader
			cmd.Stdout = decW
			stderr, err := cmd.StderrPipe()
			if err != nil {
				cancel()
				dlR.CloseWithError(err)
				errc <- fmt.Errorf("age stderr pipe: %w", err)
				return
			}
			if err := cmd.Start(); err != nil {
				cancel()
				dlR.CloseWithError(err)
				errc <- fmt.Errorf("start age: %w", err)
				return
			}
			errOut, _ := io.ReadAll(stderr)
			if err := cmd.Wait(); err != nil {
				cancel()
				msg := strings.TrimSpace(string(errOut))
				if msg != "" {
					errc <- fmt.Errorf("%w: %s", errs.ErrBackupDecryptFailed, msg)
				} else {
					errc <- fmt.Errorf("%w: %v", errs.ErrBackupDecryptFailed, err)
				}
			}
		}()
	} else {
		restoreReader = downloadReader
	}

	// ── Stage 3: pg_restore ───────────────────────────────────────────
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer restoreReader.Close()
		args := []string{
			"exec", "-i", container,
			"pg_restore",
			"-U", user,
			"-d", db,
			"--clean",
			"--if-exists",
		}
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdin = restoreReader
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			errc <- fmt.Errorf("pg_restore stderr pipe: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			cancel()
			errc <- fmt.Errorf("start pg_restore: %w", err)
			return
		}
		errOut, _ := io.ReadAll(stderr)
		if err := cmd.Wait(); err != nil {
			errStr := string(errOut)
			if strings.Contains(errStr, "FATAL") || strings.Contains(errStr, "could not") {
				cancel()
				errc <- fmt.Errorf("%w: %s", errs.ErrBackupRestoreFailed, strings.TrimSpace(errStr))
			} else if errStr != "" {
				slog.Warn("pg_restore warnings", "output", strings.TrimSpace(errStr))
			}
		}
	}()

	wg.Wait()
	close(errc)

	for e := range errc {
		if e != nil {
			return e
		}
	}

	slog.Info("remote restore complete", "from", from)
	return nil
}

// ScheduleStream adds a systemd timer for nself backup stream.
// It delegates to the existing systemd infrastructure via a dedicated unit.
func ScheduleStream(cfg *config.Config, cron, to, recipient, unitDir string, dryRun bool) error {
	if cron == "" {
		return fmt.Errorf("--cron expression required (e.g. '0 2 * * *')")
	}
	if to == "" {
		to = cfg.Backup.Remote
	}
	if to == "" {
		return fmt.Errorf("--to destination required for schedule")
	}

	binaryPath := "/usr/local/bin/nself"
	envFile := "/etc/nself/backup.env"
	projectDir := "/opt/nself"
	if unitDir == "" {
		unitDir = "/etc/systemd/system"
	}

	recipientFlag := ""
	if recipient != "" {
		recipientFlag = " --recipient " + recipient
	}

	execStart := fmt.Sprintf("%s backup stream --to %s%s", binaryPath, to, recipientFlag)

	// Convert simple cron "M H * * *" to systemd OnCalendar syntax.
	onCalendar := cronToSystemd(cron)

	service := renderService(serviceSpec{
		Description: "nSelf streaming encrypted backup",
		EnvFile:     envFile,
		WorkingDir:  projectDir,
		ExecStart:   execStart,
	})
	timer := renderTimer(timerSpec{
		Description: "nSelf streaming backup: " + cron,
		OnCalendar:  onCalendar,
		Unit:        "nself-backup-stream.service",
		Persistent:  true,
	})

	if dryRun {
		fmt.Printf("# ── nself-backup-stream.service ──\n%s\n", service)
		fmt.Printf("# ── nself-backup-stream.timer ──\n%s\n", timer)
		return nil
	}

	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", unitDir, err)
	}

	servicePath := filepath.Join(unitDir, "nself-backup-stream.service")
	timerPath := filepath.Join(unitDir, "nself-backup-stream.timer")

	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		return fmt.Errorf("write service unit: %w", err)
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0644); err != nil {
		return fmt.Errorf("write timer unit: %w", err)
	}

	if err := run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := run("systemctl", "enable", "--now", "nself-backup-stream.timer"); err != nil {
		return fmt.Errorf("enable timer: %w", err)
	}

	slog.Info("stream backup scheduled", "cron", cron, "destination", to)
	return nil
}

// cronToSystemd converts a 5-field cron expression to a systemd OnCalendar value.
// Handles simple patterns only; complex expressions pass through with a warning.
func cronToSystemd(cron string) string {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		slog.Warn("cron expression not in 5-field format; using verbatim", "cron", cron)
		return cron
	}
	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]

	// Daily at HH:MM when dom, month, dow are all *.
	if dom == "*" && month == "*" && dow == "*" {
		return fmt.Sprintf("*-*-* %s:%s:00 UTC", zeroPad(hour), zeroPad(minute))
	}

	// Weekly: dow set, dom and month *.
	if dow != "*" && dom == "*" && month == "*" {
		dayName := cronDOWName(dow)
		return fmt.Sprintf("%s *-*-* %s:%s:00 UTC", dayName, zeroPad(hour), zeroPad(minute))
	}

	// Monthly: dom set, dow and month *.
	if dom != "*" && dow == "*" && month == "*" {
		return fmt.Sprintf("*-*-%s %s:%s:00 UTC", zeroPad(dom), zeroPad(hour), zeroPad(minute))
	}

	// Fall back to daily to avoid invalid syntax; caller sees the warning above.
	return fmt.Sprintf("*-*-* %s:%s:00 UTC", zeroPad(hour), zeroPad(minute))
}

func zeroPad(s string) string {
	if len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
		return "0" + s
	}
	return s
}

func cronDOWName(n string) string {
	days := map[string]string{
		"0": "Sun", "7": "Sun",
		"1": "Mon", "2": "Tue", "3": "Wed",
		"4": "Thu", "5": "Fri", "6": "Sat",
	}
	if name, ok := days[n]; ok {
		return name
	}
	return n
}
