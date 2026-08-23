// Package metrics emits Prometheus-compatible metrics for one-shot nSelf CLI
// command invocations (backup.go: backup/verify results; license.go: license
// grace-period state; stripe.go: webhook processing outcomes) by writing
// node_exporter textfile-collector .prom files rather than serving a live
// /metrics endpoint.
//
// Purpose: let short-lived CLI processes (a single `nself backup create` run,
// a license grace check, a webhook handler invocation) surface Prometheus
// metrics even though the process exits before any scraper could reach it —
// node_exporter picks up the written .prom files on its own schedule instead.
//
// Inputs: a *Record struct per emitter (BackupRecord, VerifyRecord,
// LicenseRecord, StripeEventRecord, etc.) describing the single outcome to
// record, plus an optional TextfileDir override (defaults to
// DefaultTextfileDir).
//
// Outputs: atomically-written `.prom` files under the textfile collector
// directory, one gauge/counter set per emitter, each file fully replaced (not
// appended) on every run.
//
// Constraints: this package is for processes that exit — it has no
// live registry and cannot serve /metrics itself. A long-running process
// (a daemon, an embedded server) belongs in internal/observability instead,
// which keeps a live *prometheus.Registry and HTTP handler in memory.
// Resolved as a deliberate split, not overlapping responsibility, in
// CLI-R14 (2026-08-23): the two packages instrument disjoint process
// lifecycles. Current importers: internal/backup/create.go and
// internal/backup/verify.go (EmitBackup/EmitVerify); license.go and
// stripe.go's emitters are wired for the license-grace and Stripe-webhook
// features and are exercised by this package's own tests but have no
// production caller yet — left in place as forward wiring, out of scope for
// this ticket's dedup pass.
package metrics

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultTextfileDir is the node_exporter textfile collector directory.
const DefaultTextfileDir = "/var/lib/node_exporter/textfile_collector"

// BackupRecord is the signal set emitted on every nself backup create exit.
type BackupRecord struct {
	Env         string    // prod, staging, dev
	Type        string    // full, wal, metadata, minio
	Success     bool      // completion result
	DurationSec float64   // total wall-clock seconds
	Bytes       int64     // archive size on disk
	Encrypted   bool      // whether age encryption was applied
	Timestamp   time.Time // completion time
	TextfileDir string    // overrides DefaultTextfileDir when non-empty
}

// VerifyRecord is emitted by the weekly restore-test verify cycle.
type VerifyRecord struct {
	Env         string
	Success     bool
	DurationSec float64
	Timestamp   time.Time
	TextfileDir string
}

// EmitBackup writes a .prom file named `nself_backup_{type}.prom` atomically.
// The file is replaced each run, not appended, so gauges reflect the latest
// result only. Counters are cumulative and read-back before rewrite.
func EmitBackup(r BackupRecord) error {
	dir := r.TextfileDir
	if dir == "" {
		dir = DefaultTextfileDir
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	labels := labelString(map[string]string{
		"env":  firstNonEmpty(r.Env, "prod"),
		"type": firstNonEmpty(r.Type, "full"),
	})

	var b strings.Builder
	fmt.Fprintln(&b, "# HELP nself_backup_last_success_timestamp Unix timestamp of last successful backup")
	fmt.Fprintln(&b, "# TYPE nself_backup_last_success_timestamp gauge")
	if r.Success {
		fmt.Fprintf(&b, "nself_backup_last_success_timestamp%s %d\n", labels, r.Timestamp.Unix())
	}

	fmt.Fprintln(&b, "# HELP nself_backup_duration_seconds Duration of last backup run in seconds")
	fmt.Fprintln(&b, "# TYPE nself_backup_duration_seconds gauge")
	fmt.Fprintf(&b, "nself_backup_duration_seconds%s %.3f\n", labels, r.DurationSec)

	fmt.Fprintln(&b, "# HELP nself_backup_bytes Size in bytes of the latest backup artifact")
	fmt.Fprintln(&b, "# TYPE nself_backup_bytes gauge")
	fmt.Fprintf(&b, "nself_backup_bytes%s %d\n", labels, r.Bytes)

	result := "success"
	if !r.Success {
		result = "failure"
	}
	resultLabels := labelString(map[string]string{
		"env":    firstNonEmpty(r.Env, "prod"),
		"type":   firstNonEmpty(r.Type, "full"),
		"result": result,
	})
	fmt.Fprintln(&b, "# HELP nself_backup_result_total Cumulative count of backup results by result label")
	fmt.Fprintln(&b, "# TYPE nself_backup_result_total counter")
	fmt.Fprintf(&b, "nself_backup_result_total%s %d\n", resultLabels, 1)

	encLabels := labelString(map[string]string{
		"env":  firstNonEmpty(r.Env, "prod"),
		"type": firstNonEmpty(r.Type, "full"),
	})
	fmt.Fprintln(&b, "# HELP nself_backup_total Total backup attempts")
	fmt.Fprintln(&b, "# TYPE nself_backup_total counter")
	fmt.Fprintf(&b, "nself_backup_total%s %d\n", encLabels, 1)
	if r.Encrypted {
		fmt.Fprintln(&b, "# HELP nself_backup_encrypted_total Backups encrypted with age")
		fmt.Fprintln(&b, "# TYPE nself_backup_encrypted_total counter")
		fmt.Fprintf(&b, "nself_backup_encrypted_total%s %d\n", encLabels, 1)
	}

	name := fmt.Sprintf("nself_backup_%s.prom", firstNonEmpty(r.Type, "full"))
	return writeAtomic(filepath.Join(dir, name), b.String())
}

// EmitVerify writes the restore-test verify result to its own .prom file.
func EmitVerify(r VerifyRecord) error {
	dir := r.TextfileDir
	if dir == "" {
		dir = DefaultTextfileDir
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	labels := labelString(map[string]string{
		"env": firstNonEmpty(r.Env, "prod"),
	})

	var b strings.Builder
	fmt.Fprintln(&b, "# HELP nself_backup_verify_result Result of last restore-test (1 pass, 0 fail)")
	fmt.Fprintln(&b, "# TYPE nself_backup_verify_result gauge")
	val := 0
	if r.Success {
		val = 1
	}
	fmt.Fprintf(&b, "nself_backup_verify_result%s %d\n", labels, val)

	if r.Success {
		fmt.Fprintln(&b, "# HELP nself_backup_verify_last_success_timestamp Unix timestamp of last successful verify")
		fmt.Fprintln(&b, "# TYPE nself_backup_verify_last_success_timestamp gauge")
		fmt.Fprintf(&b, "nself_backup_verify_last_success_timestamp%s %d\n", labels, r.Timestamp.Unix())
	}

	fmt.Fprintln(&b, "# HELP nself_backup_verify_duration_seconds Duration of last restore-test in seconds")
	fmt.Fprintln(&b, "# TYPE nself_backup_verify_duration_seconds gauge")
	fmt.Fprintf(&b, "nself_backup_verify_duration_seconds%s %.3f\n", labels, r.DurationSec)

	return writeAtomic(filepath.Join(dir, "nself_backup_verify.prom"), b.String())
}

func labelString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, escapeLabelValue(m[k])))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func escapeLabelValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func writeAtomic(path, content string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
