package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

// adminPort returns the admin UI port from env or falls back to 3021.
func adminPort() string {
	if p := os.Getenv("NSELF_ADMIN_PORT"); p != "" {
		return p
	}
	if p := os.Getenv("ADMIN_PORT"); p != "" {
		return p
	}
	return "3021"
}

// adminContainerID returns the docker container name for the admin service.
func adminContainerID() string {
	cfg, _, err := loadHealthConfig()
	if err != nil || cfg.ProjectName == "" {
		return "nself_admin"
	}
	return fmt.Sprintf("%s_admin", cfg.ProjectName)
}

var adminCmd = &cobra.Command{
	Use:   "admin [subcommand]",
	Short: "Manage the nSelf Admin dashboard",
	Long: `Open, start, stop, inspect logs for, or health-check the nSelf Admin UI.

With no subcommand, opens the Admin dashboard (http://localhost:3021) in your
default browser.

Subcommands:
  start   Start the Admin service (enables + builds + boots if not running)
  stop    Stop the Admin container gracefully
  logs    Tail Admin container logs
  health  Check Admin liveness (HTTP probe on /health)`,
	RunE: runAdmin,
}

var adminStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Admin service",
	RunE:  runAdminStart,
}

var adminStopFlags struct {
	force bool
}

var adminStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Admin container",
	RunE:  runAdminStop,
}

var adminLogsFlags struct {
	follow bool
	tail   int
}

var adminLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail Admin container logs",
	RunE:  runAdminLogs,
}

var adminHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check Admin service liveness",
	RunE:  runAdminHealth,
}

func init() {
	adminStopCmd.Flags().BoolVar(&adminStopFlags.force, "force", false, "Skip graceful drain and force-stop")
	adminLogsCmd.Flags().BoolVar(&adminLogsFlags.follow, "follow", false, "Stream logs continuously")
	adminLogsCmd.Flags().IntVar(&adminLogsFlags.tail, "tail", 100, "Number of recent lines to show")

	adminCmd.AddCommand(adminStartCmd)
	adminCmd.AddCommand(adminStopCmd)
	adminCmd.AddCommand(adminLogsCmd)
	adminCmd.AddCommand(adminHealthCmd)

	RootCmd.AddCommand(adminCmd)
}

// runAdmin opens the Admin UI in the default browser (default action, no subcommand).
func runAdmin(cmd *cobra.Command, args []string) error {
	port := adminPort()
	adminURL := "http://localhost:" + port

	ctx := cmd.Context()

	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.CommandContext(ctx, "open", adminURL)
	case "windows":
		openCmd = exec.CommandContext(ctx, "cmd", "/c", "start", adminURL)
	default:
		openCmd = exec.CommandContext(ctx, "xdg-open", adminURL)
	}

	if err := openCmd.Start(); err != nil {
		// Graceful fallback — headless server or command not found.
		fmt.Printf("Admin UI: %s\n", adminURL)
		return nil
	}

	fmt.Printf("Opening %s in your browser...\n", adminURL)
	return nil
}

// runAdminStart enables the admin service, rebuilds, and starts it.
// Idempotent: prints "already running" if the container is healthy.
func runAdminStart(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	port := adminPort()
	adminURL := "http://localhost:" + port + "/health"

	// Check if already running.
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	resp, err := http.Get(adminURL) //nolint:noctx // intentional quick probe
	cancel()
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Println("Admin is already running at http://localhost:" + port)
			return nil
		}
	}
	_ = probeCtx

	// Enable admin in .env.
	envFile, err2 := resolveEnvFile("")
	if err2 != nil {
		return err2
	}
	if err2 := setEnvKeyInFile(envFile, "NSELF_ADMIN_ENABLED", "true"); err2 != nil {
		return fmt.Errorf("enabling admin: %w", err2)
	}

	// Rebuild.
	fmt.Println("Building admin service...")
	buildCmd := exec.CommandContext(ctx, "nself", "build")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err2 := buildCmd.Run(); err2 != nil {
		return fmt.Errorf("nself build: %w", err2)
	}

	// Start the admin container.
	fmt.Println("Starting admin container...")
	startCmd := exec.CommandContext(ctx, "docker", "start", adminContainerID())
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err2 := startCmd.Run(); err2 != nil {
		// Fall back to nself start if docker start fails (first run).
		nstart := exec.CommandContext(ctx, "nself", "start", "admin")
		nstart.Stdout = os.Stdout
		nstart.Stderr = os.Stderr
		if err3 := nstart.Run(); err3 != nil {
			return fmt.Errorf("starting admin: %w", err3)
		}
	}

	fmt.Println("Admin started at http://localhost:" + port)
	return nil
}

// runAdminStop stops the admin container gracefully (or forcefully with --force).
func runAdminStop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cid := adminContainerID()

	dockerArgs := []string{"stop"}
	if adminStopFlags.force {
		dockerArgs = append(dockerArgs, "-t", "0")
	}
	dockerArgs = append(dockerArgs, cid)

	stopCmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	stopCmd.Stdout = os.Stdout
	stopCmd.Stderr = os.Stderr
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("stopping admin container: %w", err)
	}

	fmt.Println("Admin stopped.")
	return nil
}

// runAdminLogs tails or streams admin container logs.
func runAdminLogs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cid := adminContainerID()

	dockerArgs := []string{"logs"}
	if adminLogsFlags.follow {
		dockerArgs = append(dockerArgs, "--follow")
	}
	dockerArgs = append(dockerArgs, fmt.Sprintf("--tail=%d", adminLogsFlags.tail))
	dockerArgs = append(dockerArgs, cid)

	logsCmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	logsCmd.Stdout = os.Stdout
	logsCmd.Stderr = os.Stderr
	return logsCmd.Run()
}

// runAdminHealth probes GET /health on the admin service.
func runAdminHealth(cmd *cobra.Command, args []string) error {
	port := adminPort()
	url := "http://localhost:" + port + "/health"

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("Admin health: unhealthy (%v)\n", err)
		return fmt.Errorf("admin unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Admin health: healthy (HTTP %d, %s)\n", resp.StatusCode, elapsed.Truncate(time.Millisecond))
		return nil
	}

	fmt.Printf("Admin health: unhealthy (HTTP %d, %s)\n", resp.StatusCode, elapsed.Truncate(time.Millisecond))
	return fmt.Errorf("admin returned HTTP %d", resp.StatusCode)
}
