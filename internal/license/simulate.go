// Package license — simulate.go provides offline simulation mode for
// testing grace period behavior without actually blocking network access.
//
// Simulation writes an override to the license cache that makes the system
// believe it has been offline for N days. Guarded by LICENSE_ALLOW_SIMULATION
// env var (must be "true" explicitly; defaults to false in production).
package license

import (
	"fmt"
	"os"
	"time"
)

// SimulationGuardEnv is the env var that must be set to "true" to allow
// simulation mode. This prevents accidental use in production.
const SimulationGuardEnv = "LICENSE_ALLOW_SIMULATION"

// ErrSimulationNotAllowed is returned when simulation is attempted without
// the guard env var being set.
var ErrSimulationNotAllowed = fmt.Errorf(
	"simulation mode is disabled — set %s=true to enable (never in production)",
	SimulationGuardEnv,
)

// SimulateOffline modifies the license cache to appear as if the system has
// been offline for the given number of days. This is used for testing grace
// period transitions without iptables manipulation.
//
// The function:
// 1. Checks the LICENSE_ALLOW_SIMULATION guard
// 2. Reads the current cache
// 3. Backdates FetchedAt by the specified number of days
// 4. Writes the modified cache back
func SimulateOffline(days int) (*GraceCheckResult, error) {
	if os.Getenv(SimulationGuardEnv) != "true" {
		return nil, ErrSimulationNotAllowed
	}

	if days < 0 {
		return nil, fmt.Errorf("days must be non-negative, got %d", days)
	}

	entry, err := ReadCache()
	if err != nil {
		return nil, fmt.Errorf("reading license cache: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("no license cache found — run 'nself license validate' first")
	}

	// Backdate the FetchedAt timestamp to simulate being offline for N days.
	entry.FetchedAt = time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()

	if err := WriteCache(entry); err != nil {
		return nil, fmt.Errorf("writing simulated cache: %w", err)
	}

	// Evaluate and return the resulting grace state.
	result := DetermineGraceState(entry)
	return &result, nil
}

// ClearSimulation restores the cache to current time (as if just validated).
// Also guarded by LICENSE_ALLOW_SIMULATION.
func ClearSimulation() error {
	if os.Getenv(SimulationGuardEnv) != "true" {
		return ErrSimulationNotAllowed
	}

	entry, err := ReadCache()
	if err != nil {
		return fmt.Errorf("reading license cache: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("no license cache found")
	}

	entry.FetchedAt = time.Now().Unix()
	return WriteCache(entry)
}
