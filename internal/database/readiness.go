package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// postgresReadinessStreak is the number of consecutive successful probes
// required before postgres is considered ready.
//
// A single success is not enough: the official postgres image's entrypoint
// boots a TEMPORARY server bound to the local socket on first run so it can
// run initdb and any /docker-entrypoint-initdb.d scripts, then shuts that
// server down and starts the real one. The temporary server answers probes
// (including plain pg_isready) successfully, so a first-success check can
// return during that window — and the very next statement then lands in the
// shutdown, exactly as run 32948887818 showed ("the database system is
// shutting down"). Requiring a streak that RESETS on any failure means the
// shutdown between the temporary and real server always breaks the count
// before we report ready, no matter how the timing lines up.
const postgresReadinessStreak = 3

// errPostgresNotReady is returned internally by waitForPostgresReady when the
// readiness streak was never achieved before the timeout. It is translated
// to errs.ErrDatabaseNotRunning by waitForPostgres so callers keep seeing the
// same sentinel as before.
var errPostgresNotReady = errors.New("postgres readiness streak not achieved within timeout")

// postgresProbeFunc performs a single readiness check against postgres.
// Extracted as a function value so the streak/reset logic in
// waitForPostgresReady can be unit tested without Docker or a real server.
type postgresProbeFunc func(ctx context.Context) error

// waitForPostgresReady polls probe until postgresReadinessStreak consecutive
// calls succeed, or maxWait elapses. Any single failure resets the streak to
// zero. See postgresReadinessStreak for why a streak (not first-success) is
// required.
func waitForPostgresReady(ctx context.Context, probe postgresProbeFunc, maxWait, interval time.Duration) error {
	deadline := time.Now().Add(maxWait)
	streak := 0

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := probe(checkCtx)
		cancel()

		if err == nil {
			streak++
			if streak >= postgresReadinessStreak {
				return nil
			}
		} else {
			streak = 0
		}

		time.Sleep(interval)
	}

	return errPostgresNotReady
}

// waitForPostgres waits for the postgres container's REAL server (not the
// temporary init-time server, see postgresReadinessStreak) to accept
// connections, up to a 60s timeout. It probes with an actual query
// (`SELECT 1`) rather than pg_isready, since pg_isready only checks that the
// socket accepts connections — which the temporary server also does — and
// the streak requirement needs a probe that reflects the server actually
// serving queries. Returns errs.ErrDatabaseNotRunning if postgres does not
// become ready in time.
func waitForPostgres(ctx context.Context, cfg *config.Config) error {
	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	probe := func(probeCtx context.Context) error {
		var stderr bytes.Buffer
		cmd := exec.CommandContext(probeCtx, "docker", "exec", container,
			"psql", "-U", user, "-d", "postgres", "-tAc", "SELECT 1",
		)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return nil
	}

	const (
		maxWait  = 60 * time.Second
		interval = 1 * time.Second
	)

	if err := waitForPostgresReady(ctx, probe, maxWait, interval); err != nil {
		if errors.Is(err, errPostgresNotReady) {
			return fmt.Errorf("PostgreSQL failed to start within 60s. Run 'nself logs postgres' for details: %w", errs.ErrDatabaseNotRunning)
		}
		// Context cancellation/deadline — propagate as-is.
		return err
	}

	slog.Info("postgres is ready", "container", container)
	return nil
}
