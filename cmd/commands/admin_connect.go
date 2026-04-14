package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

var adminConnectCmd = &cobra.Command{
	Use:   "connect <host>",
	Short: "Connect to a remote nSelf Admin via SSH tunnel",
	Long: `Establish an SSH tunnel to a remote nSelf Admin instance.

This creates a local port forward from localhost:3022 to the remote host's
admin port (default 3021), allowing the local Admin UI to manage remote servers.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetInt("port")
		localPort, _ := cmd.Flags().GetInt("local-port")
		remotePort, _ := cmd.Flags().GetInt("remote-port")

		conn := AdminConnection{
			Name:       host,
			Host:       host,
			User:       user,
			Port:       port,
			LocalPort:  localPort,
			RemotePort: remotePort,
		}

		// Save connection config
		if err := saveAdminConnection(conn); err != nil {
			ui.Warn(fmt.Sprintf("Failed to save connection: %v", err))
		}

		// Establish SSH tunnel
		sshArgs := []string{
			"-N", "-L",
			fmt.Sprintf("%d:127.0.0.1:%d", localPort, remotePort),
			"-p", fmt.Sprintf("%d", port),
			fmt.Sprintf("%s@%s", user, host),
		}

		ui.Bullet(fmt.Sprintf("Establishing SSH tunnel: localhost:%d → %s:%d", localPort, host, remotePort))

		sshCmd := exec.CommandContext(cmd.Context(), "ssh", sshArgs...)
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Start(); err != nil {
			return fmt.Errorf("ssh tunnel failed: %w", err)
		}

		ui.Success(fmt.Sprintf("Connected. Admin UI available at http://localhost:%d", localPort))
		ui.Bullet("Press Ctrl+C to disconnect")

		return sshCmd.Wait()
	},
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
			fmt.Printf("  %-20s  %-30s  %s\n", p.ID, p.Name, p.URL)
		}
		return nil
	},
}

var adminProjectsAddCmd = &cobra.Command{
	Use:   "add <id> <name> <path>",
	Short: "Add a project to the admin multi-project config",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")

		project := AdminProject{
			ID:   args[0],
			Name: args[1],
			Path: args[2],
			URL:  url,
		}

		projects, _ := loadAdminProjects()
		// Check for duplicates
		for _, p := range projects {
			if p.ID == project.ID {
				return fmt.Errorf("project %q already exists", project.ID)
			}
		}

		projects = append(projects, project)
		if err := saveAdminProjects(projects); err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Added project %q", project.Name))
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
	return filepath.Join(home, ".config", "nself")
}

func saveAdminConnection(conn AdminConnection) error {
	dir := adminConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(dir, "admin-connections.json")
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
	path := filepath.Join(adminConfigDir(), "admin-projects.json")
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
	return os.WriteFile(filepath.Join(dir, "admin-projects.json"), data, 0o644)
}

func init() {
	adminConnectCmd.Flags().String("user", "nself", "SSH user")
	adminConnectCmd.Flags().Int("port", 22, "SSH port")
	adminConnectCmd.Flags().Int("local-port", 3022, "Local port for tunnel")
	adminConnectCmd.Flags().Int("remote-port", 3021, "Remote admin port")

	adminProjectsListCmd.Flags().Bool("json", false, "JSON output")
	adminProjectsAddCmd.Flags().String("url", "", "Project URL")

	adminProjectsCmd.AddCommand(adminProjectsListCmd, adminProjectsAddCmd, adminProjectsRemoveCmd)

	adminCmd.AddCommand(adminConnectCmd, adminProjectsCmd)
}
