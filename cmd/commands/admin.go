package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
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

var adminStartFlags struct {
	exposeNetwork bool
}

var adminStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Admin service",
	Long: `Start the nSelf Admin service.

By default the Admin UI is bound to 127.0.0.1:3021 and is only reachable from
the local machine. Use --expose-network to bind to 0.0.0.0:3021 and make it
reachable from other hosts on your network.

WARNING: --expose-network must only be used behind a TLS-terminating reverse
proxy with authentication enabled. Never expose the Admin UI to the public
internet without additional protection.`,
	RunE: runAdminStart,
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
	adminStartCmd.Flags().BoolVar(&adminStartFlags.exposeNetwork, "expose-network", false, "Bind to 0.0.0.0 instead of 127.0.0.1 (WARNING: only use behind a TLS reverse proxy)")
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

	// Skip the browser launch in test / CI / headless contexts. Launching
	// Safari or xdg-open here during `go test` caused orphan GUI processes
	// that pushed the macos-14 CI job past its 10-minute timeout.
	if !shouldOpenBrowser() {
		fmt.Printf("Admin UI: %s\n", adminURL)
		return nil
	}

	openCmd := openBrowserCmd(cmd.Context(), adminURL)
	if openCmd == nil {
		fmt.Printf("Admin UI: %s\n", adminURL)
		return nil
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

	// --expose-network warning banner. Admin is a local-only tool; binding to
	// 0.0.0.0 exposes credentials and sensitive operations to the network.
	if adminStartFlags.exposeNetwork {
		fmt.Println("WARNING: --expose-network binds Admin to 0.0.0.0:" + port)
		fmt.Println("         Only use this behind a TLS reverse proxy with authentication.")
		fmt.Println("         Never expose Admin directly to the public internet.")
		fmt.Println()
	}

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

	// Set network binding. Default is 127.0.0.1 (loopback only). When
	// --expose-network is passed, set to 0.0.0.0 so the admin is reachable
	// from other hosts on the network (use with caution, see warning above).
	adminBindHost := "127.0.0.1"
	if adminStartFlags.exposeNetwork {
		adminBindHost = "0.0.0.0"
	}
	if err2 := setEnvKeyInFile(envFile, "NSELF_ADMIN_BIND_HOST", adminBindHost); err2 != nil {
		return fmt.Errorf("setting admin bind host: %w", err2)
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

	listenAddr := adminBindHost + ":" + port
	fmt.Printf("Admin started. Listening on %s\n", listenAddr)
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
	resp, err := httptimeout.Default.Do(req)
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
