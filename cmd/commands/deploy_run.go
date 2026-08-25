package commands

// Purpose: runDeploy, the RunE for the top-level "nself deploy" command. Inputs
// are the cobra command/args (target, strategy, flags); outputs are deploy step
// results printed as text or JSON, or a non-nil error on failure.
// Constraints: split out of deploy.go (CLI-R12) as a pure move, no behavior change.
// This file is a CLI-R12 justified exception: runDeploy is a single ~374-line
// function with no internal phase boundaries, so it cannot be brought under the
// 300-line cap by moving code, doing so would require extracting phases from a
// production deploy path, which is a refactor, not a move.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/controlplane"
	"github.com/nself-org/cli/internal/deploy/bluegreen"
	"github.com/nself-org/cli/internal/maintenance"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

func runDeploy(cmd *cobra.Command, args []string) error {
	// Resolve target: --env flag takes priority over the positional argument.
	envFlag, _ := cmd.Flags().GetString("env")
	var rawTarget string
	switch {
	case envFlag != "":
		rawTarget = envFlag
	case len(args) == 1:
		rawTarget = args[0]
	default:
		return fmt.Errorf("target environment required: pass it as an argument (nself deploy staging) or via --env (nself deploy --env staging)")
	}
	target, err := resolveTarget(rawTarget)
	if err != nil {
		return err
	}

	// Load the env file cascade for the resolved target into the current process
	// so that downstream helpers (build, health checks, SSH env) pick up the
	// correct environment variables. NSELF_DEPLOY_ENV is also set for subprocesses.
	// workdir may not be known yet; use a best-effort lookup here.
	if wd, wdErr := projectRoot(); wdErr == nil {
		loadDeployEnvCascade(wd, target)
	}

	strategy, _ := cmd.Flags().GetString("strategy")
	if rolling, _ := cmd.Flags().GetBool("rolling"); rolling {
		strategy = "rolling"
	}
	if !deployStrategies[strategy] {
		return fmt.Errorf("invalid strategy %q (allowed: rolling, blue-green, canary, preview)", strategy)
	}

	// Strategies other than rolling are not yet implemented. Fall back to
	// rolling with an explicit warning so users know the flag was accepted but
	// has no effect yet.
	if notYetImplementedStrategies[strategy] {
		if !func() bool { v, _ := cmd.Flags().GetBool("json"); return v }() {
			ui.Warn(fmt.Sprintf("Strategy %q is not yet implemented in v1.0.9. Tracked for v1.1.0; falling back to rolling. See .claude/docs/operations/deploy-strategies.md", strategy))
		}
		strategy = "rolling"
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		force = true
	}
	follow, _ := cmd.Flags().GetBool("follow")
	skipHealth, _ := cmd.Flags().GetBool("skip-health")
	jsonOut, _ := cmd.Flags().GetBool("json")
	includeFrontends, _ := cmd.Flags().GetBool("include-frontends")
	excludeFrontends, _ := cmd.Flags().GetBool("exclude-frontends")
	canaryPct, _ := cmd.Flags().GetInt("canary")
	skipCanary, _ := cmd.Flags().GetBool("skip-canary")
	forceMigration, _ := cmd.Flags().GetBool("force-migration")
	serverFilter, _ := cmd.Flags().GetString("server")

	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	// Blue/green canary path (Y17 — blue_green_deploy feature flag).
	// When --canary N is passed and the flag is ON, route to the bluegreen package.
	// The feature flag check is intentionally lightweight: the env var
	// NSELF_FEATURE_BLUE_GREEN_DEPLOY=true mirrors what the feature-flags plugin
	// would return (nself flags list | grep blue_green_deploy). In production the
	// flag plugin is the authoritative source; the env var is the fallback for
	// environments without the flags plugin running.
	bgEnabled := os.Getenv("NSELF_FEATURE_BLUE_GREEN_DEPLOY") == "true"
	if (canaryPct > 0 || skipCanary) && bgEnabled {
		if !jsonOut {
			label := fmt.Sprintf("canary=%d%% skip-canary=%v force-migration=%v dry-run=%v", canaryPct, skipCanary, forceMigration, dryRun)
			ui.CommandHeader(fmt.Sprintf("nself deploy %s (blue/green)", target), label)
		}
		if target == "prod" && !dryRun && !force {
			return fmt.Errorf("production blue/green deploy requires --force. Re-run with --force once ready")
		}
		cfg := bluegreen.DeployConfig{
			ProjectRoot:    workdir,
			CanaryPercent:  canaryPct,
			SkipCanary:     skipCanary,
			ForceMigration: forceMigration,
			DryRun:         dryRun,
		}
		result := bluegreen.Deploy(cmd.Context(), cfg)
		if jsonOut {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
		} else if result.RolledBack {
			ui.Error("Canary auto-rolled back: " + result.Error)
		} else if !result.Success {
			ui.Error("Blue/green deploy failed: " + result.Error)
		} else {
			ui.Success(fmt.Sprintf("Blue/green deploy complete in %s", result.Duration.Round(time.Millisecond)))
		}
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}
		return nil
	}

	if !jsonOut {
		ui.CommandHeader(fmt.Sprintf("nself deploy %s", target), fmt.Sprintf("strategy=%s dry-run=%v include-frontends=%v exclude-frontends=%v", strategy, dryRun, includeFrontends, excludeFrontends))
	}

	// Production safety gate: require --force (or a "prod" confirm) when not in dry-run.
	if target == "prod" && !dryRun && !force {
		return fmt.Errorf("production deploy requires --force (or --dry-run). Re-run with --force once ready")
	}

	// T05: Control-plane pipeline path.
	// Active when .nself/control-plane.yaml exists OR when --server is specified.
	// When neither condition is true the legacy single-host path runs unchanged
	// (back-compat byte-identical guarantee per sprint spec).
	cpYamlPath := filepath.Join(workdir, ".nself", "control-plane.yaml")
	_, cpYamlErr := os.Stat(cpYamlPath)
	usePipeline := cpYamlErr == nil || serverFilter != ""

	if usePipeline {
		inv, loadErr := controlplane.Load(workdir)
		if loadErr != nil {
			return fmt.Errorf("deploy: load inventory: %w", loadErr)
		}

		// Apply server filter: remove all servers that do not match the requested name.
		if serverFilter != "" {
			inv = filterInventoryByServer(inv, serverFilter)
			if totalServers(inv) == 0 {
				return fmt.Errorf("deploy: --server %q not found in inventory", serverFilter)
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
			return nil
		}

		// Execute via topology-aware pipeline.
		prober := controlplane.NewSSHProber(workdir, false)
		composePath := filepath.Join(workdir, "docker-compose.yml")

		if !jsonOut {
			ui.CommandHeader(fmt.Sprintf("nself deploy %s (pipeline)", target), fmt.Sprintf("strategy=%s server=%s", strategy, serverFilter))
		}

		result, pipeErr := controlplane.Run(cmd.Context(), inv, prober, composePath)
		if pipeErr != nil {
			return fmt.Errorf("deploy pipeline: %w", pipeErr)
		}

		// Primary-skip gate: non-zero exit when primary was skipped.
		if result.PrimarySkipped {
			if !jsonOut {
				ui.Warn("Primary server was skipped (read-only capability) — deploy incomplete")
			} else {
				b, _ := json.MarshalIndent(result.Servers, "", "  ")
				fmt.Println(string(b))
			}
			return fmt.Errorf("deploy: primary server skipped (read-only capability); re-run once SSH access is restored")
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
		return nil
	}

	steps := []deployStep{}
	start := time.Now()

	// Build
	if !dryRun {
		if !jsonOut {
			fmt.Println("  [running] Build images")
		}
		if err := runCLISelf(cmd.Context(), workdir, "build"); err != nil {
			return finalize(jsonOut, target, strategy, start, append(steps, deployStep{Name: "Build images", Status: "failed"}), err)
		}
	}
	steps = append(steps, deployStep{Name: "Build images", Status: stepStatus(dryRun, "done")})
	if !jsonOut && !dryRun {
		fmt.Println("  [done] Build images")
	}

	// Target-specific action
	switch target {
	case "local":
		if dryRun {
			if !jsonOut {
				fmt.Printf("  [dry-run] Would: docker compose up -d (rolling: %v, frontends: include=%v exclude=%v)\n", strategy, includeFrontends, excludeFrontends)
			}
			steps = append(steps, deployStep{Name: "Start local stack", Status: "pending"})
		} else {
			if !jsonOut {
				fmt.Println("  [running] Start local stack (rolling sequenced restart)")
			}
			restartSteps, restartErr := runRollingRestart(cmd.Context(), workdir, jsonOut)
			steps = append(steps, restartSteps...)
			if restartErr != nil {
				return finalize(jsonOut, target, strategy, start, steps, restartErr)
			}
		}

	case "staging", "prod":
		host := os.Getenv("NSELF_DEPLOY_HOST_" + strings.ToUpper(target))
		if host == "" {
			// Fall back to common env-var name patterns
			host = os.Getenv(strings.ToUpper(target) + "_DEPLOY_HOST")
		}

		if dryRun {
			if host != "" {
				if !jsonOut {
					fmt.Printf("  [dry-run] Would: ssh+rsync to %s then docker compose pull + rolling restart\n", host)
					fmt.Printf("  [dry-run] SSH key: %s\n", sshKeyPath())
					fmt.Printf("  [dry-run] Rolling restart order: %s\n", strings.Join(deployServiceOrder, " → "))
					fmt.Printf("  [dry-run] Frontends: include=%v exclude=%v\n", includeFrontends, excludeFrontends)
				}
				steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "pending"})
				steps = append(steps, deployStep{Name: "Rolling restart (sequenced)", Status: "pending"})
			} else {
				if !jsonOut {
					fmt.Printf("  [dry-run] No NSELF_DEPLOY_HOST_%s set; would run locally\n", strings.ToUpper(target))
					fmt.Printf("  [dry-run] Set NSELF_DEPLOY_HOST_%s=user@host:/path to enable remote push\n", strings.ToUpper(target))
				}
				steps = append(steps, deployStep{Name: "Start stack (local host)", Status: "pending"})
			}
		} else if host != "" {
			// Remote push: rsync compose file + env + migrations, then pull images
			// and run the rolling restart on the remote host via ssh.
			if !jsonOut {
				fmt.Printf("  [running] Remote push to %s\n", host)
			}
			pushErr := remoteDeployPush(cmd.Context(), workdir, host, target, jsonOut)
			if pushErr != nil {
				steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "failed"})
				return finalize(jsonOut, target, strategy, start, steps, pushErr)
			}
			steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "done"})
		} else {
			// No host configured: run locally (matches v0.9.x behaviour when
			// deploy is triggered from a session on the target machine itself).
			if !jsonOut {
				fmt.Println("  [running] Start stack (rolling sequenced restart, local host)")
			}
			restartSteps, restartErr := runRollingRestart(cmd.Context(), workdir, jsonOut)
			steps = append(steps, restartSteps...)
			if restartErr != nil {
				return finalize(jsonOut, target, strategy, start, steps, restartErr)
			}
		}

		// Health gate (post-restart).
		if skipHealth {
			if !jsonOut {
				ui.Warn("Skipping health checks (--skip-health). Stack state unverified.")
			}
			steps = append(steps, deployStep{Name: "Health checks", Status: "skipped"})
		} else if !dryRun {
			healthStep, healthErr := runDeployHealthCheck(cmd.Context(), workdir, jsonOut)
			steps = append(steps, healthStep)
			if healthErr != nil {
				return finalize(jsonOut, target, strategy, start, steps, healthErr)
			}
		} else {
			steps = append(steps, deployStep{Name: "Health checks", Status: "pending"})
		}
	}

	// Auto-install daily disk-cleanup timer after successful staging/prod deploy (P98 T10.T07).
	// Non-fatal: a failure here warns but does not roll back the deploy.
	if !dryRun && (target == "staging" || target == "prod") {
		if timerErr := maintenance.InstallDailyTimer(); timerErr != nil {
			ui.Warn(fmt.Sprintf("daily maintenance timer install failed (non-fatal): %v", timerErr))
			ui.Warn("Run `nself maintenance schedule --daily` manually to enable disk-cleanup cron")
		}
	}

	if err := finalize(jsonOut, target, strategy, start, steps, nil); err != nil {
		return err
	}

	// --follow: stream container logs until Ctrl-C (staging/prod only).
	// Not supported for dry-run or local (local already has foreground start).
	if follow && !dryRun && (target == "staging" || target == "prod") {
		if !jsonOut {
			ui.Info("Following container logs (Ctrl-C to stop)...")
		}
		host := os.Getenv("NSELF_DEPLOY_HOST_" + strings.ToUpper(target))
		if host == "" {
			host = os.Getenv(strings.ToUpper(target) + "_DEPLOY_HOST")
		}
		if host != "" {
			// Remote follow: tail logs on the remote host via SSH.
			colonIdx := strings.LastIndex(host, ":")
			sshTarget := host
			remotePath := ""
			if colonIdx >= 0 {
				sshTarget = host[:colonIdx]
				remotePath = host[colonIdx+1:]
			}
			sshKey := sshKeyPath()
			logsCmd := "docker compose logs -f --tail=50"
			if remotePath != "" {
				logsCmd = fmt.Sprintf("cd %s && docker compose logs -f --tail=50", remotePath)
			}
			sc := exec.CommandContext(cmd.Context(), "ssh",
				"-i", sshKey,
				"-o", "StrictHostKeyChecking=accept-new",
				"-o", "ForwardAgent=no",
				sshTarget, logsCmd)
			sc.Stdout = os.Stdout
			sc.Stderr = os.Stderr
			_ = sc.Run() // Ctrl-C exits; non-zero exit is not an error from user perspective.
		} else {
			// Local host: stream logs directly.
			lc := exec.CommandContext(cmd.Context(), "docker", "compose", "logs", "-f", "--tail=50")
			lc.Dir = workdir
			lc.Stdout = os.Stdout
			lc.Stderr = os.Stderr
			_ = lc.Run()
		}
	}

	return nil
}
