package commands

// deploy_run_pipeline.go — T05 control-plane pipeline deploy path for
// `nself deploy`.
//
// Purpose: active when .nself/control-plane.yaml exists OR --server is
//          specified; runs the topology-aware multi-server pipeline instead
//          of the legacy single-host path. Split out of deploy_run.go
//          (T-P6-E2-W1-S1-T3) for 300-line compliance.
// Inputs:  the relevant runDeploy flag values + cmd/workdir/target/strategy.
// Outputs: handled=false when neither control-plane.yaml nor --server is
//          present — caller runs the legacy single-host path unchanged
//          (back-compat byte-identical guarantee). handled=true means this
//          function fully handled the deploy and its err is runDeploy's result.
// Constraints: pure move, same checks/output/errors, no behavior change.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/controlplane"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// runDeployControlPlanePipeline handles the T05 pipeline deploy path. See
// file header for the handled/err contract.
func runDeployControlPlanePipeline(cmd *cobra.Command, workdir, target, strategy, serverFilter string, dryRun, jsonOut bool) (handled bool, err error) {
	// T05: Control-plane pipeline path.
	// Active when .nself/control-plane.yaml exists OR when --server is specified.
	// When neither condition is true the legacy single-host path runs unchanged
	// (back-compat byte-identical guarantee per sprint spec).
	cpYamlPath := filepath.Join(workdir, ".nself", "control-plane.yaml")
	_, cpYamlErr := os.Stat(cpYamlPath)
	usePipeline := cpYamlErr == nil || serverFilter != ""
	if !usePipeline {
		return false, nil
	}

	inv, loadErr := controlplane.Load(workdir)
	if loadErr != nil {
		return true, fmt.Errorf("deploy: load inventory: %w", loadErr)
	}

	// Apply server filter: remove all servers that do not match the requested name.
	if serverFilter != "" {
		inv = filterInventoryByServer(inv, serverFilter)
		if totalServers(inv) == 0 {
			return true, fmt.Errorf("deploy: --server %q not found in inventory", serverFilter)
		}
	}

	// --dry-run: print topology plan and exit without executing.
	if dryRun {
		prober := controlplane.NewSSHProber(workdir, false)
		statuses := controlplane.Resolve(inv, prober)
		if !jsonOut {
			fmt.Printf("  [dry-run] Topology plan for target %q:\n", target)
			for _, ts := range statuses {
				if ts.Capability == controlplane.CapHidden {
					continue
				}
				fmt.Printf("    %s/%s role=%-15s capability=%s", ts.Env, ts.Server, "", string(ts.Capability))
				if ts.Reason != "" {
					fmt.Printf(" reason=%q", ts.Reason)
				}
				fmt.Println()
			}
		} else {
			type dryRow struct {
				Env        string `json:"env"`
				Server     string `json:"server"`
				Capability string `json:"capability"`
				Reason     string `json:"reason,omitempty"`
			}
			var rows []dryRow
			for _, ts := range statuses {
				if ts.Capability == controlplane.CapHidden {
					continue
				}
				rows = append(rows, dryRow{Env: ts.Env, Server: ts.Server, Capability: string(ts.Capability), Reason: ts.Reason})
			}
			b, _ := json.MarshalIndent(map[string]interface{}{"dry_run": true, "topology": rows}, "", "  ")
			fmt.Println(string(b))
		}
		return true, nil
	}

	// Execute via topology-aware pipeline.
	prober := controlplane.NewSSHProber(workdir, false)
	composePath := filepath.Join(workdir, "docker-compose.yml")

	if !jsonOut {
		ui.CommandHeader(fmt.Sprintf("nself deploy %s (pipeline)", target), fmt.Sprintf("strategy=%s server=%s", strategy, serverFilter))
	}

	result, pipeErr := controlplane.Run(cmd.Context(), inv, prober, composePath)
	if pipeErr != nil {
		return true, fmt.Errorf("deploy pipeline: %w", pipeErr)
	}

	// Primary-skip gate: non-zero exit when primary was skipped.
	if result.PrimarySkipped {
		if !jsonOut {
			ui.Warn("Primary server was skipped (read-only capability) — deploy incomplete")
		} else {
			b, _ := json.MarshalIndent(result.Servers, "", "  ")
			fmt.Println(string(b))
		}
		return true, fmt.Errorf("deploy: primary server skipped (read-only capability); re-run once SSH access is restored")
	}

	if jsonOut {
		b, _ := json.MarshalIndent(result.Servers, "", "  ")
		fmt.Println(string(b))
	} else {
		for _, sr := range result.Servers {
			switch sr.Status {
			case "ok":
				ui.Success(fmt.Sprintf("  [ok] %s/%s", sr.Env, sr.Server))
			case "skipped":
				ui.Warn(fmt.Sprintf("  [skipped] %s/%s (read-only)", sr.Env, sr.Server))
			case "failed":
				ui.Error(fmt.Sprintf("  [failed] %s/%s: %v", sr.Env, sr.Server, sr.Err))
			}
		}
		ui.Success(fmt.Sprintf("Deploy %s (pipeline) complete", target))
	}
	return true, nil
}
