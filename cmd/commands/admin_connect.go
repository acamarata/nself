package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/nself-org/cli/internal/admin"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// AdminConnection represents a saved remote admin connection.
type AdminConnection struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	User       string `json:"user"`
	Port       int    `json:"port"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
}

// AdminProject represents a project in the multi-project config.
type AdminProject struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Host    string `json:"host"`
	SSHUser string `json:"ssh_user"`
	URL     string `json:"url"`
}

var adminConnectCmd = &cobra.Command{
	Use:   "connect [host]",
	Short: "Connect to a remote nSelf Admin via SSH tunnel",
	Long: `Establish an SSH tunnel to a remote nSelf Admin instance.

Steps performed:
  1. Verify SSH key-based auth to HOST
  2. Start nself-admin on the server if not running
  3. Open local forward: -L localPort:127.0.0.1:remotePort
  4. Generate per-session token
  5. Open browser to http://localhost:<localPort>?token=<token>
  6. Tear down on Ctrl+C or UI disconnect`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		allFlag, _ := cmd.Flags().GetBool("all")
		asUser, _ := cmd.Flags().GetString("as")
		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetInt("port")
		localPort, _ := cmd.Flags().GetInt("local-port")
		remotePort, _ := cmd.Flags().GetInt("remote-port")

		if len(args) == 0 && !allFlag {
			return fmt.Errorf("specify a host or use --all for multi-project mode")
		}

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		// Handle Ctrl+C for clean teardown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			ui.Bullet("Disconnecting...")
			cancel()
		}()

		host := ""
		if len(args) > 0 {
			host = args[0]
		}

		if allFlag {
			return connectAllProjects(ctx, user, port, localPort, remotePort, asUser)
		}

		return connectSingle(ctx, host, user, port, localPort, remotePort, asUser)
	},
}

func connectSingle(ctx context.Context, host, user string, port, localPort, remotePort int, asUser string) error {
	// Step 1: Verify SSH key auth
	ui.Bullet(fmt.Sprintf("Verifying SSH key auth for %s@%s:%d...", user, host, port))
	if err := admin.VerifySSHKey(ctx, user, host, port); err != nil {
		return err
	}
	ui.Success("SSH key auth verified")

	// Step 2: Start nself-admin on remote if needed
	ui.Bullet("Ensuring nself-admin is running on remote host...")
	if err := admin.EnsureRemoteAdmin(ctx, user, host, port); err != nil {
		ui.Warn(fmt.Sprintf("Could not start remote admin: %v", err))
	}

	// Step 3: Generate session token
	token, err := admin.NewSessionToken()
	if err != nil {
		return err
	}

	// Save connection config
	conn := AdminConnection{
		Name:       host,
		Host:       host,
		User:       user,
		Port:       port,
		LocalPort:  localPort,
		RemotePort: remotePort,
	}
	if saveErr := saveAdminConnection(conn); saveErr != nil {
		ui.Warn(fmt.Sprintf("Failed to save connection: %v", saveErr))
	}

	// Step 4: Open SSH tunnel
	opts := admin.ConnectOpts{
		Host:       host,
		User:       user,
		SSHPort:    port,
		LocalPort:  localPort,
		RemotePort: remotePort,
		AsUser:     asUser,
	}

	ui.Bullet(fmt.Sprintf("Opening SSH tunnel: localhost:%d -> %s:127.0.0.1:%d", localPort, host, remotePort))
	tunnelCmd, err := admin.OpenTunnel(ctx, opts)
	if err != nil {
		return err
	}

	// Step 5: Open browser
	adminURL := admin.AdminURL(localPort, token, "")
	ui.Success(fmt.Sprintf("Connected. Admin UI: %s", adminURL))
	if openErr := admin.OpenBrowser(adminURL); openErr != nil {
		ui.Warn(fmt.Sprintf("Could not open browser: %v (open manually: %s)", openErr, adminURL))
	}

	ui.Bullet("Press Ctrl+C to disconnect")

	// Step 6: Wait for tunnel to end (Ctrl+C or disconnect)
	return tunnelCmd.Wait()
}

func connectAllProjects(ctx context.Context, defaultUser string, sshPort, localPort, remotePort int, asUser string) error {
	projects, err := loadAdminProjects()
	if err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects configured; use 'nself admin projects add' first")
	}

	// Connect to the first project by default
	p := projects[0]
	host := p.Host
	if host == "" {
		return fmt.Errorf("project %q has no host configured", p.Name)
	}
	sshUser := p.SSHUser
	if sshUser == "" {
		sshUser = defaultUser
	}

	ui.CommandHeader("Multi-Project Admin", fmt.Sprintf("%d projects registered", len(projects)))
	for _, proj := range projects {
		fmt.Printf("  %-20s  %s\n", proj.ID, proj.Host)
	}

	return connectSingle(ctx, host, sshUser, sshPort, localPort, remotePort, asUser)
}

var adminProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage multi-project admin configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var adminProjectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured admin projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")

		projects, err := loadAdminProjects()
		if err != nil {
			return fmt.Errorf("load projects: %w", err)
		}

		if jsonOut {
			data, _ := json.MarshalIndent(projects, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(projects) == 0 {
			ui.Bullet("No projects configured. Use 'nself admin projects add' to add one.")
			return nil
		}

		ui.CommandHeader("Admin Projects", fmt.Sprintf("%d projects", len(projects)))
		for _, p := range projects {
			fmt.Printf("  %-20s  %-20s  %-30s  %s\n", p.ID, p.Host, p.Name, p.URL)
		}
		return nil
	},
}

var adminProjectsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a project to the admin multi-project config",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		host, _ := cmd.Flags().GetString("host")
		sshUser, _ := cmd.Flags().GetString("ssh-user")
		url, _ := cmd.Flags().GetString("url")

		if name == "" || host == "" {
			return fmt.Errorf("--name and --host are required")
		}

		project := AdminProject{
			ID:      name,
			Name:    name,
			Host:    host,
			SSHUser: sshUser,
			URL:     url,
		}

		projects, _ := loadAdminProjects()
		for _, p := range projects {
			if p.ID == project.ID {
				return fmt.Errorf("project %q already exists", project.ID)
			}
		}

		projects = append(projects, project)
		if err := saveAdminProjects(projects); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Added project %q (host: %s)", project.Name, project.Host))
		return nil
	},
}

var adminProjectsRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a project from admin config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := loadAdminProjects()
		if err != nil {
			return err
		}

		var filtered []AdminProject
		found := false
		for _, p := range projects {
			if p.ID == args[0] {
				found = true
				continue
			}
			filtered = append(filtered, p)
		}

		if !found {
			return fmt.Errorf("project %q not found", args[0])
		}

		if err := saveAdminProjects(filtered); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Removed project %q", args[0]))
		return nil
	},
}

func adminConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nself", "admin")
}

func saveAdminConnection(conn AdminConnection) error {
	dir := adminConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(dir, "connections.json")
	var conns []AdminConnection

	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &conns)
	}

	// Replace or append
	found := false
	for i, c := range conns {
		if c.Host == conn.Host {
			conns[i] = conn
			found = true
			break
		}
	}
	if !found {
		conns = append(conns, conn)
	}

	out, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func loadAdminProjects() ([]AdminProject, error) {
	path := filepath.Join(adminConfigDir(), "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var projects []AdminProject
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func saveAdminProjects(projects []AdminProject) error {
	dir := adminConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "projects.json"), data, 0o644)
}

func init() {
	adminConnectCmd.Flags().String("user", "nself", "SSH user")
	adminConnectCmd.Flags().Int("port", 22, "SSH port")
	adminConnectCmd.Flags().Int("local-port", 3021, "Local port for tunnel")
	adminConnectCmd.Flags().Int("remote-port", 3021, "Remote admin port")
	adminConnectCmd.Flags().Bool("all", false, "Connect with all registered projects")
	adminConnectCmd.Flags().String("as", "", "Override authenticated identity (for ACL testing)")

	adminProjectsListCmd.Flags().Bool("json", false, "JSON output")
	adminProjectsAddCmd.Flags().String("name", "", "Project name/ID")
	adminProjectsAddCmd.Flags().String("host", "", "Remote host")
	adminProjectsAddCmd.Flags().String("ssh-user", "ops", "SSH user for project")
	adminProjectsAddCmd.Flags().String("url", "", "Project URL")

	adminProjectsCmd.AddCommand(adminProjectsListCmd, adminProjectsAddCmd, adminProjectsRemoveCmd)

	adminCmd.AddCommand(adminConnectCmd, adminProjectsCmd)
}
