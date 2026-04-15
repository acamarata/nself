package dr

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// Rollback demotes the promoted standby and resyncs from the original primary.
func Rollback(ctx context.Context, cfg *config.Config) error {
	standbyHost := cfg.DR.StandbyHost
	if standbyHost == "" {
		return fmt.Errorf("%w: DR_STANDBY_HOST not configured", errs.ErrDRRollbackFailed)
	}

	slog.Info("rolling back: demoting standby, resyncing from original primary", "standby", standbyHost)

	// Step 1: Stop services on the promoted standby.
	slog.Info("step 1: stopping services on standby")
	stopArgs := []string{standbyHost, "nself", "stop"}
	cmd := exec.CommandContext(ctx, "ssh", stopArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("stop on standby failed", "output", string(output), "error", err)
	}

	// Step 2: Reconfigure DNS back to original primary.
	slog.Info("step 2: restoring DNS to original primary")
	slog.Warn("automatic DNS restoration is not yet available — Cloudflare API integration is pending. Manually update DNS A records to point back to the original primary via the Cloudflare dashboard")

	// Step 3: Resync standby from original primary using pg_basebackup.
	slog.Info("step 3: resyncing standby from primary")
	return fmt.Errorf("%w: pg_basebackup resync is not yet automated — manually run pg_basebackup on %s to restore replication from the original primary", errs.ErrDRRollbackFailed, standbyHost)
}

// Fence sets a read-only flag in Redis for split-brain prevention.
func Fence(ctx context.Context, cfg *config.Config) error {
	if !cfg.Redis.Enabled {
		return fmt.Errorf("%w: Redis not enabled — cannot set fence", errs.ErrDRFenceFailed)
	}

	container := cfg.ProjectName + "_redis"
	slog.Info("setting split-brain fence in Redis", "container", container)

	args := []string{
		"exec", container,
		"redis-cli", "SET", "nself:dr:fence", "1", "EX", "3600",
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", errs.ErrDRFenceFailed, string(output))
	}

	slog.Info("split-brain fence set (TTL: 3600s)")
	return nil
}
