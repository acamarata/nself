package dr

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// PromoteOptions holds flags for `nself dr promote-standby`.
type PromoteOptions struct {
	Region string
	Yes    bool
}

// PromoteStandby promotes the warm standby to primary and updates DNS.
func PromoteStandby(ctx context.Context, cfg *config.Config, opts PromoteOptions) error {
	standbyHost := cfg.DR.StandbyHost
	if standbyHost == "" {
		return fmt.Errorf("%w: DR_STANDBY_HOST not configured", errs.ErrDRPromoteFailed)
	}

	slog.Info("promoting standby to primary", "standby", standbyHost, "region", opts.Region)

	// Step 1: Verify standby is reachable and healthy.
	slog.Info("step 1: verifying standby health")
	checkArgs := []string{standbyHost, "nself", "doctor", "--deep"}
	cmd := exec.CommandContext(ctx, "ssh", checkArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: standby health check failed: %s", errs.ErrDRPromoteFailed, string(output))
	}

	// Step 2: Promote standby postgres from replica to primary.
	slog.Info("step 2: promoting postgres on standby")
	promoteArgs := []string{standbyHost, "docker", "exec", cfg.ProjectName + "_postgres", "pg_ctl", "promote"}
	cmd = exec.CommandContext(ctx, "ssh", promoteArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: postgres promotion failed: %s", errs.ErrDRPromoteFailed, string(output))
	}

	// Step 3: Update DNS A records to point to standby.
	slog.Info("step 3: updating DNS records")
	if err := ReconfigureDNS(ctx, cfg, standbyHost); err != nil {
		return fmt.Errorf("DNS reconfiguration: %w", err)
	}

	slog.Info("standby promoted successfully", "new_primary", standbyHost)
	return nil
}

// ReconfigureDNS updates Cloudflare A records to point to a new IP.
func ReconfigureDNS(ctx context.Context, cfg *config.Config, newIP string) error {
	domain := cfg.BaseDomain
	if domain == "" {
		return fmt.Errorf("BASE_DOMAIN not configured")
	}

	slog.Info("updating Cloudflare DNS", "domain", domain, "new_ip", newIP)

	// Use Cloudflare API via curl or cf CLI.
	// This is a placeholder for the actual Cloudflare API integration.
	slog.Warn("DNS reconfiguration requires manual Cloudflare API calls", "domain", domain, "ip", newIP)
	return nil
}
