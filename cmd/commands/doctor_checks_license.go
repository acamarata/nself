package commands

// Purpose: doctor checks for config validators and license state: cache
// freshness and migration rate. Inputs are the project dir and a verbose flag;
// outputs are doctorCheckResult values.
// Constraints: split out of doctor.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/plugin"
)

// checkConfigValidators runs config.Validate against the loaded configuration
// and reports the result. T04 will wire Validate() to call RunAll() internally,
// so this will automatically cover all registered validators after T04 lands.
func checkConfigValidators(projectDir string, verbose bool) doctorCheckResult {
	name := "Config validators"
	cfg, err := config.Load(projectDir)
	if err != nil {
		msg := fmt.Sprintf("cannot load config: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	if err := config.Validate(cfg); err != nil {
		msg := fmt.Sprintf("config validation failed: %v", err)
		printCheck("fail", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "fail", Message: msg}
	}

	msg := "config validators passed"
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// licensestaleDays is the number of days after which the license cache is
// considered stale.
const licensestaleDays = 7

// checkLicenseCache inspects the license entitlements cache. It reports the
// cache age and tier when present, and warns if the cache is older than 7 days.
func checkLicenseCache(verbose bool) doctorCheckResult {
	name := "License cache"

	// Use the public LicenseCacheDir helper so the path stays consistent
	// with the plugin manager.
	cacheDir := plugin.LicenseCacheDir()
	entitlementsPath := filepath.Join(cacheDir, "entitlements.json")

	data, err := os.ReadFile(entitlementsPath)
	if os.IsNotExist(err) {
		// No cache file — not an error, just informational.
		msg := "no license cache found (run 'nself license validate' to populate)"
		printCheck("pass", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}
	if err != nil {
		msg := fmt.Sprintf("cannot read license cache: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	// Parse the entitlements JSON (tier + cached_at).
	var cache struct {
		Tier     string `json:"tier"`
		CachedAt string `json:"cached_at"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		msg := fmt.Sprintf("cannot parse license cache: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	cachedAt, err := time.Parse(time.RFC3339, cache.CachedAt)
	if err != nil {
		msg := fmt.Sprintf("cannot parse cache timestamp: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	age := time.Since(cachedAt)
	ageDays := int(age.Hours() / 24)
	tier := cache.Tier
	if tier == "" {
		tier = "unknown"
	}

	if ageDays >= licensestaleDays {
		msg := fmt.Sprintf("tier=%s, cache age=%dd — license cache is stale — run 'nself license refresh'", tier, ageDays)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	msg := fmt.Sprintf("tier=%s, cache age=%dd", tier, ageDays)
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// checkLicenseMigrationRate implements the LIC-MIGRATION-01 doctor check (SP-04.O11 T11).
//
// After migration has been running for 60+ days, warns if more than 10% of
// daily license validations are still using unmigrated (legacy) keys.
// Data is read from the ping_api telemetry endpoint if NSELF_PING_API_URL is set,
// otherwise the check is skipped (non-fatal — only prod infra exposes telemetry).
func checkLicenseMigrationRate(verbose bool) doctorCheckResult {
	name := "LIC-MIGRATION-01: License migration rate"

	pingURL := os.Getenv("NSELF_PING_API_URL")
	if pingURL == "" {
		pingURL = defaultPingURL
	}

	// Query the migration telemetry summary endpoint (admin-only, only available
	// when DATABASE_URL is configured on the server — not reachable from end-user CLI).
	// We attempt a HEAD to confirm the endpoint exists; if unreachable we skip.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(pingURL + "/admin/migration/status")
	if err != nil {
		// Unreachable — not an error for end-user CLI, skip silently.
		msg := "migration telemetry endpoint unreachable — skipped"
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		// Endpoint exists but requires admin auth — skip (not an error for end-user).
		msg := "migration telemetry requires admin access — skipped"
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("migration telemetry returned HTTP %d — skipped", resp.StatusCode)
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	var telemetry struct {
		TotalHits          int     `json:"total_hits"`
		PendingHits        int     `json:"pending_hits"`
		MigratedHits       int     `json:"migrated_hits"`
		MigrationStartDate string  `json:"migration_start_date"`
		PendingRatioPct    float64 `json:"pending_ratio_pct"`
		DaysSinceMigration int     `json:"days_since_migration"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&telemetry); err != nil {
		msg := fmt.Sprintf("cannot parse migration telemetry: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	// Only alert after 60 days since migration start (spec: acceptance criterion 7).
	if telemetry.DaysSinceMigration < 60 {
		msg := fmt.Sprintf("migration running for %d days — alert threshold not yet reached (60 days)", telemetry.DaysSinceMigration)
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	if telemetry.TotalHits == 0 {
		msg := "no validation hits recorded today"
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	ratio := telemetry.PendingRatioPct
	if ratio > 10.0 {
		msg := fmt.Sprintf("%.1f%% of daily license validations are still using unmigrated keys (threshold: 10%%) — run: nself license migrate --account-id <uuid>", ratio)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	msg := fmt.Sprintf("%.1f%% pending ratio — within threshold", ratio)
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}
