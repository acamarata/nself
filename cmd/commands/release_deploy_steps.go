package commands

// Purpose: The forward-release cascade step implementations invoked by
// runRelease via runStep — pre-release gate, git tag + GitHub release for
// cli and plugins-pro, admin Docker build/push, Homebrew formula PR, ping
// API canary deploy, artifact verification, web package.json version
// bumps, Vercel deploy, README badge sync, SPORT regen, soak-protocol
// start, and post-release PCIs. Split out of release.go (CLI-R12) to
// separate the many small step functions from the cascade orchestration
// (runRelease) and cobra wiring that remain in release.go.
// Inputs: a context.Context, the release tag/version string, and (for
// runReleaseCheckStep) the current executable's own path.
// Outputs: errors surfaced back through runStep to runRelease's
// releaseResult; each step calls external tools (git, gh, docker, brew,
// vercel, pci-send) as subprocesses.
// Constraints: pure move — no behavior changes. These functions must keep
// the exact step order and step numbers wired in runRelease's runStep(...)
// calls in release.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runReleaseCheckStep(ctx context.Context, ver string) error {
	bin, _ := os.Executable()
	if bin == "" {
		bin, _ = exec.LookPath("nself")
	}
	if bin == "" {
		return fmt.Errorf("nself binary not found for release-check")
	}
	cmd := exec.CommandContext(ctx, bin, "release-check", ver, "--skip-ci")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func execReleaseCLITag(ctx context.Context, tag string) error {
	if err := gitTagAndPush(ctx, tag); err != nil {
		return err
	}
	return createGitHubRelease(ctx, "nself-org/cli", tag)
}

func execReleasePluginsProTag(ctx context.Context, tag string) error {
	// plugins-pro is a private repo — same cascade
	if err := gitTagAndPushInRepo(ctx, tag, "../plugins-pro"); err != nil {
		return fmt.Errorf("plugins-pro tag: %w", err)
	}
	return createGitHubRelease(ctx, "nself-org/plugins-pro", tag)
}

func execAdminDockerBuild(ctx context.Context, ver string) error {
	image := fmt.Sprintf("nself/nself-admin:%s", ver)
	buildArgs := []string{"build", "-t", image, "-t", "nself/nself-admin:latest", "."}
	c := exec.CommandContext(ctx, "docker", buildArgs...)
	c.Dir = "../admin"
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	pushVer := exec.CommandContext(ctx, "docker", "push", image)
	pushVer.Stdout = os.Stdout
	pushVer.Stderr = os.Stderr
	if err := pushVer.Run(); err != nil {
		return fmt.Errorf("docker push %s: %w", image, err)
	}
	pushLatest := exec.CommandContext(ctx, "docker", "push", "nself/nself-admin:latest")
	pushLatest.Stdout = os.Stdout
	pushLatest.Stderr = os.Stderr
	return pushLatest.Run()
}

func execHomebrewPR(ctx context.Context, ver string) error {
	// Trigger the dispatch workflow or run the Ruby formula update script
	c := exec.CommandContext(ctx, "gh", "workflow", "run", "update-formula.yml",
		"--repo", "nself-org/homebrew-nself",
		"-f", "version="+ver,
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("homebrew workflow dispatch: %w", err)
	}
	return nil
}

func execPingAPIDeploy(ctx context.Context, ver string) error {
	// Deploy via vercel CLI (canary → promote)
	deploy := exec.CommandContext(ctx, "vercel", "deploy",
		"--env", "NSELF_CLI_VERSION="+ver,
	)
	deploy.Dir = "../web/backend/services/ping_api"
	deploy.Stdout = os.Stdout
	deploy.Stderr = os.Stderr
	if err := deploy.Run(); err != nil {
		return fmt.Errorf("ping_api canary deploy: %w", err)
	}
	promote := exec.CommandContext(ctx, "vercel", "promote", "--yes")
	promote.Dir = "../web/backend/services/ping_api"
	promote.Stdout = os.Stdout
	promote.Stderr = os.Stderr
	return promote.Run()
}

func execArtifactVerification(ctx context.Context, ver string) error {
	// Curl ping.nself.org/version and verify latestCliVersion
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ping.nself.org/version", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ping.nself.org/version unreachable: %w", err)
	}
	defer resp.Body.Close()
	var v struct {
		LatestCLIVersion string `json:"latestCliVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return fmt.Errorf("parse ping_api response: %w", err)
	}
	reported := strings.TrimPrefix(v.LatestCLIVersion, "v")
	if reported != ver {
		return fmt.Errorf("ping_api reports %q but expected %q — deploy may not be live yet", reported, ver)
	}
	return nil
}

func execWebVersionBumps(ctx context.Context, ver string) error {
	// Semver validation gate — rejects non-semver strings before any exec call.
	// This prevents shell injection via ver (e.g. "1.0.0;echo INJECTED").
	if !releaseVerRe.MatchString(ver) {
		return fmt.Errorf("invalid version %q: must match semver (e.g. 1.2.3 or 1.2.3-rc.1)", ver)
	}

	// Update package.json version across all web subapps.
	// ver is passed via NSELF_VERSION env var — never interpolated into the script string itself.
	const nodeScript = `
const fs = require('fs');
const path = require('path');
const ver = process.env.NSELF_VERSION;
const subapps = ['org','docs','nchat','nclaw','cloud','install','base','ntask','ntv','nfamily','clawde'];
subapps.forEach(app => {
  const pkg = path.join('../web', app, 'package.json');
  if (!fs.existsSync(pkg)) return;
  const j = JSON.parse(fs.readFileSync(pkg, 'utf8'));
  j.version = ver;
  fs.writeFileSync(pkg, JSON.stringify(j, null, 2)+'\n');
  console.log('bumped', app, 'to', j.version);
});
`
	c := exec.CommandContext(ctx, "node", "--eval", nodeScript)
	c.Env = append(os.Environ(), "NSELF_VERSION="+ver)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func execVercelDeploy(ctx context.Context, ver string) error {
	// Trigger Vercel production deploy by pushing to main (auto-deploy) or explicit CLI
	c := exec.CommandContext(ctx, "vercel", "deploy", "--prod", "--yes")
	c.Dir = "../web"
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("vercel deploy: %w", err)
	}
	_ = ver
	return nil
}

func execReadmeBadges(ctx context.Context, ver string) error {
	// Semver validation gate — rejects non-semver strings before any file operation.
	if !releaseVerRe.MatchString(ver) {
		return fmt.Errorf("invalid version %q: must match semver (e.g. 1.2.3 or 1.2.3-rc.1)", ver)
	}
	_ = ctx // no shell invoked; ctx used for future cancellation support

	// Go-native badge replace — no shell, no fmt.Sprintf interpolation into a shell command.
	replacement := "version-" + ver + "-"
	repos := []string{"cli", "admin", "nchat", "nclaw", "ntask", "ntv", "nfamily", "clawde", "plugins", "web"}
	for _, r := range repos {
		readmePath := fmt.Sprintf("../%s/README.md", r)
		if _, err := os.Stat(readmePath); err != nil {
			continue
		}
		f, err := os.OpenFile(readmePath, os.O_RDWR, 0o644)
		if err != nil {
			return fmt.Errorf("badge update open %s/README.md: %w", r, err)
		}
		data, err := io.ReadAll(f)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("badge update read %s/README.md: %w", r, err)
		}
		updated := badgePat.ReplaceAll(data, []byte(replacement))
		if err := f.Truncate(0); err != nil {
			_ = f.Close()
			return fmt.Errorf("badge update truncate %s/README.md: %w", r, err)
		}
		if _, err := f.WriteAt(updated, 0); err != nil {
			_ = f.Close()
			return fmt.Errorf("badge update write %s/README.md: %w", r, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("badge update close %s/README.md: %w", r, err)
		}
		fmt.Printf("badge updated %s/README.md → %s\n", r, ver)
	}
	return nil
}

func execSPORTRegen(ctx context.Context, ver string) error {
	// Run SPORT regeneration if script exists
	sportScript := "../.claude/docs/sport/regen.sh"
	if _, err := os.Stat(sportScript); err == nil {
		c := exec.CommandContext(ctx, "bash", sportScript)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("SPORT regen: %w", err)
		}
	}
	_ = ver
	return nil
}

func execSoakStart(ctx context.Context, ver string) error {
	// Write soak-start marker to .claude/docs/operations/release-soak-protocol.md
	// and emit a PCI to Downloads agent
	soakPath := "../.claude/docs/operations/release-soak-protocol.md"
	if _, err := os.Stat(soakPath); err == nil {
		entry := fmt.Sprintf("\n## Soak started for v%s\n\n- Started: %s\n- Target: 48h with no CRITICAL incidents\n- GA gate: 2 × green synthetic probe cycles\n",
			ver, time.Now().UTC().Format(time.RFC3339))
		f, err := os.OpenFile(soakPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.WriteString(entry)
			f.Close()
		}
	}

	// PCI to Downloads inbox to start camclaw soak
	msg := fmt.Sprintf(`## Context
Release v%s has shipped. 48h soak protocol starts now.

## Request
Monitor camclaw instance for CRITICAL issues. Report back after 48h or immediately on any CRITICAL incident.

## Soak criteria
- Zero CRITICAL issues in 48h window
- All synthetic probes green
- User-reported issues: none CRITICAL
`, ver)
	c := exec.CommandContext(ctx,
		os.ExpandEnv("$HOME/bin/pci-send"),
		"nclaw", fmt.Sprintf("soak-start-v%s", ver), "medium", "info",
		fmt.Sprintf("48h soak started for v%s", ver),
	)
	c.Stdin = strings.NewReader(msg)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run() // non-fatal — soak can be started manually
	return nil
}

func execPostReleasePCIs(_ context.Context, ver string) error {
	fmt.Printf("    ↳ Post-release PCIs: send 'v%s released' to camclaw / ummat / acamarata inboxes via pci-send\n", ver)
	return nil
}
