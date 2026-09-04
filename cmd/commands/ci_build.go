package commands

// Purpose: nself ci build — local release-artifact build lane.
//   Wraps the nself-ci binary's "build" subcommand (plugins/free/ci) so a
//   private repo (web, packages, plugins-pro) can produce a signed Android
//   release APK on the developer's own machine and optionally attach it to
//   a GitHub release, without a GitHub-hosted runner or a third nSelf
//   server. Split out of ci.go (which shells to the same binary for the
//   default gate command) purely to keep each command's flag set and RunE
//   in its own file — same pattern as ci_bootstrap.go / ci_forgejo.go /
//   ci_serve.go.
// Inputs:  [flags] [android-project-dir]
// Outputs: build log to stdout, signed APK on disk, optional gh release
//   upload, exit 0/1
// Constraints: Android only — macOS/Windows/TV/WearOS artifacts stay on
//   GitHub-hosted runners (P6-E11-W2-S1-T6). No hardcoded keystore secrets;
//   ANDROID_KEYSTORE_BASE64/ANDROID_KEYSTORE_PASSWORD/ANDROID_KEY_ALIAS/
//   ANDROID_KEY_PASSWORD are read from the environment by the gate binary.
// SPORT: CLI-CMD-CI-002

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var ciBuildCmd = &cobra.Command{
	Use:   "build [android-project-dir]",
	Short: "Build a signed release artifact locally (Android only)",
	Long: `Build a signed release artifact on this machine instead of a GitHub-hosted runner.

Only "android" is supported today. Mirrors the exact keystore-signing steps
nchat's own workflows already use (build-react-native.yml,
deploy-mobile-android.yml): base64-decode a keystore from
ANDROID_KEYSTORE_BASE64, run "./gradlew assembleRelease" with the standard
ANDROID_KEYSTORE_PASSWORD / ANDROID_KEY_ALIAS / ANDROID_KEY_PASSWORD env
vars, then locate the produced signed APK.

macOS, Windows, TV, and Wear OS artifacts are explicitly out of scope — they
stay on GitHub-hosted runners; see .claude/docs/doctrines/nself-ci-runner-ceiling.md.

Examples:
  nself ci build --artifact android frontend/platforms/react-native/android
  nself ci build --artifact android --upload --tag v1.2.3 frontend/platforms/react-native/android`,
	RunE: runCIBuild,
}

func initCIBuildCmd() {
	ciBuildCmd.Flags().String("artifact", "android", `Artifact type to build (only "android" is supported)`)
	ciBuildCmd.Flags().Bool("upload", false, "Attach the built artifact to a GitHub release via `gh release upload`")
	ciBuildCmd.Flags().String("tag", "", "Release tag to upload to (required with --upload)")
	ciBuildCmd.Flags().String("owner", "", "GitHub owner for --upload (default: from git remote)")
	ciBuildCmd.Flags().String("repo", "", "GitHub repo for --upload (default: from git remote)")
	ciBuildCmd.Flags().BoolP("verbose", "v", false, "Print each command before running")
	ciBuildCmd.Flags().Int("timeout", 900, "Build step timeout in seconds (release builds are slow)")
}

func runCIBuild(cmd *cobra.Command, args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}
	projectDir = abs

	artifact, _ := cmd.Flags().GetString("artifact")
	upload, _ := cmd.Flags().GetBool("upload")
	tag, _ := cmd.Flags().GetString("tag")
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetInt("timeout")

	if upload && tag == "" {
		return fmt.Errorf("--tag is required with --upload")
	}

	binaryPath, buildErr := ensureCIBinary(verbose)
	if buildErr != nil {
		return fmt.Errorf("cannot build nself-ci gate binary: %w", buildErr)
	}

	buildArgs := []string{"build", "--artifact", artifact, "--timeout", fmt.Sprintf("%d", timeout)}
	if verbose {
		buildArgs = append(buildArgs, "-v")
	}
	if upload {
		buildArgs = append(buildArgs, "--upload", "--tag", tag)
		if owner != "" {
			buildArgs = append(buildArgs, "--owner", owner)
		}
		if repo != "" {
			buildArgs = append(buildArgs, "--repo", repo)
		}
	}
	buildArgs = append(buildArgs, projectDir)

	ui.Section("nself-ci artifact build")
	ui.Info(fmt.Sprintf("artifact: %s, dir: %s", artifact, projectDir))

	c := exec.Command(binaryPath, buildArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return fmt.Errorf("artifact build failed")
		}
		return fmt.Errorf("nself-ci build: %w", err)
	}
	return nil
}
