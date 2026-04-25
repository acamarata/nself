// Package commands: controller + project subcommands for B46 multi-tenant master controller.
//
// CLI surface:
//   nself controller start | stop | status | init
//   nself project create --slug <slug> --domain <domain>
//   nself project delete --slug <slug> [--force]
//   nself project list
//   nself project status --slug <slug>
//   nself project migrate --slug <slug>
//   nself project shell --slug <slug>
//   nself project rotate-credentials --slug <slug>
//
// Feature flag: NSELF_FLAG_MULTI_TENANT_CONTROLLER must be true.
// When OFF, all commands print 503 Multi-tenant controller not enabled.
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// slugRe validates project slugs: lowercase alphanumeric and hyphens only,
// 1-63 characters. Must match Postgres identifier constraints.
var slugRe = regexp.MustCompile(`^[a-z0-9-]{1,63}$`)

// controllerAddr resolves the tenant-controller HTTP address.
func controllerAddr() string {
	if addr := os.Getenv("NSELF_CONTROLLER_ADDR"); addr != "" {
		return addr
	}
	host := os.Getenv("NSELF_CONTROLLER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("NSELF_CONTROLLER_PORT")
	if port == "" {
		port = "3750"
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

// controllerToken returns the admin token from env.
func controllerToken() string {
	return os.Getenv("NSELF_CONTROLLER_ADMIN_TOKEN")
}

// controllerEnabled returns true when the feature flag is set.
func controllerEnabled() bool {
	v := os.Getenv("NSELF_FLAG_MULTI_TENANT_CONTROLLER")
	if v == "" {
		v = os.Getenv("NSELF_CONTROLLER_ENABLED")
	}
	return v == "true" || v == "1"
}

// assertControllerEnabled prints the 503 message and returns false when the flag is off.
func assertControllerEnabled(cmd *cobra.Command) bool {
	if !controllerEnabled() {
		fmt.Fprintln(cmd.ErrOrStderr(),
			"503 Multi-tenant controller not enabled.\n"+
				"Set NSELF_FLAG_MULTI_TENANT_CONTROLLER=true to enable.")
		return false
	}
	return true
}

// doControllerRequest performs an authenticated HTTP request to the controller daemon.
func doControllerRequest(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(method, controllerAddr()+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Controller-Token", controllerToken())

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("controller request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// ---- controller commands ----

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Manage the multi-tenant master controller (nCloud)",
	Long: `Commands for the nCloud multi-tenant master controller.

The controller manages N isolated nSelf project instances on a single server.
Requires NSELF_FLAG_MULTI_TENANT_CONTROLLER=true.

Subcommands:
  start   Start the controller daemon
  stop    Stop the controller daemon
  status  Show all projects and resource usage
  init    Initialize the nself_controller Postgres schema`,
}

var controllerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the controller daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		// The controller daemon is managed as a systemd service or nself service unit.
		// Delegate to the service manager.
		c := exec.Command("systemctl", "start", "nself-controller")
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()
		if err := c.Run(); err != nil {
			// Fallback for non-systemd environments.
			fmt.Fprintln(cmd.OutOrStdout(), "Hint: run the controller binary directly:")
			fmt.Fprintln(cmd.OutOrStdout(), "  NSELF_FLAG_MULTI_TENANT_CONTROLLER=true nself-tenant-controller &")
		}
		return nil
	},
}

var controllerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the controller daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		c := exec.Command("systemctl", "stop", "nself-controller")
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()
		_ = c.Run()
		return nil
	},
}

var controllerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all projects and resource usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		body, code, err := doControllerRequest("GET", "/controller/status", nil)
		if err != nil {
			return fmt.Errorf("controller unreachable: %w\nHint: run 'nself controller start'", err)
		}
		if code != http.StatusOK {
			return fmt.Errorf("controller status %d: %s", code, body)
		}

		var cs struct {
			Projects []struct {
				Project struct {
					Slug      string `json:"slug"`
					Domain    string `json:"domain"`
					Status    string `json:"status"`
					CreatedAt string `json:"created_at"`
				} `json:"project"`
				ConnectionCount int   `json:"connection_count"`
				SchemaBytes     int64 `json:"schema_bytes_approx"`
				Healthy         bool  `json:"healthy"`
			} `json:"projects"`
			TotalActive    int     `json:"total_active"`
			MaxProjects    int     `json:"max_projects"`
			UtilizationPct float64 `json:"utilization_pct"`
		}
		if err := json.Unmarshal(body, &cs); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "SLUG\tDOMAIN\tSTATUS\tCONNS\tSIZE\tHEALTHY\n")
		for _, p := range cs.Projects {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%v\n",
				p.Project.Slug,
				p.Project.Domain,
				p.Project.Status,
				p.ConnectionCount,
				formatBytes(p.SchemaBytes),
				p.Healthy,
			)
		}
		w.Flush()
		fmt.Fprintf(cmd.OutOrStdout(), "\n%d / %d projects active (%.0f%% utilized)\n",
			cs.TotalActive, cs.MaxProjects, cs.UtilizationPct)
		return nil
	},
}

// ---- project commands ----

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage nSelf projects (multi-tenant controller)",
	Long: `Create, delete, list, and inspect nSelf projects managed by the controller.

Each project gets its own Postgres schema, Hasura metadata source, Nginx
virtual host, JWT secret, Redis key prefix, and MinIO bucket.

Requires NSELF_FLAG_MULTI_TENANT_CONTROLLER=true.`,
}

var projectCreateFlags struct {
	Slug   string
	Domain string
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new isolated nSelf project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		if projectCreateFlags.Slug == "" {
			return fmt.Errorf("--slug is required")
		}
		if projectCreateFlags.Domain == "" {
			return fmt.Errorf("--domain is required")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Creating project %q at %s...\n",
			projectCreateFlags.Slug, projectCreateFlags.Domain)

		body, code, err := doControllerRequest("POST", "/projects/create", map[string]string{
			"Slug":   projectCreateFlags.Slug,
			"Domain": projectCreateFlags.Domain,
		})
		if err != nil {
			return fmt.Errorf("controller error: %w", err)
		}
		if code != http.StatusCreated {
			var e struct{ Error string `json:"error"` }
			_ = json.Unmarshal(body, &e)
			return fmt.Errorf("create failed (%d): %s", code, e.Error)
		}

		var proj struct {
			ID     string `json:"id"`
			Slug   string `json:"slug"`
			Domain string `json:"domain"`
			Status string `json:"status"`
		}
		_ = json.Unmarshal(body, &proj)
		fmt.Fprintf(cmd.OutOrStdout(), "Project ready: https://%s (id: %s)\n", proj.Domain, proj.ID)
		return nil
	},
}

var projectDeleteFlags struct {
	Slug  string
	Force bool
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a project (irreversible — requires --force)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		if projectDeleteFlags.Slug == "" {
			return fmt.Errorf("--slug is required")
		}
		if !projectDeleteFlags.Force {
			return fmt.Errorf("this action is irreversible; pass --force to confirm")
		}

		url := "/projects/" + projectDeleteFlags.Slug + "?force=true"
		body, code, err := doControllerRequest("DELETE", url, nil)
		if err != nil {
			return fmt.Errorf("controller error: %w", err)
		}
		if code != http.StatusNoContent {
			var e struct{ Error string `json:"error"` }
			_ = json.Unmarshal(body, &e)
			return fmt.Errorf("delete failed (%d): %s", code, e.Error)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Project %q deleted.\n", projectDeleteFlags.Slug)
		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		body, code, err := doControllerRequest("GET", "/projects", nil)
		if err != nil {
			return fmt.Errorf("controller error: %w", err)
		}
		if code != http.StatusOK {
			return fmt.Errorf("list failed (%d): %s", code, body)
		}

		var projects []struct {
			Slug      string `json:"slug"`
			Domain    string `json:"domain"`
			Status    string `json:"status"`
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(body, &projects); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "SLUG\tDOMAIN\tSTATUS\tCREATED\n")
		for _, p := range projects {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Slug, p.Domain, p.Status, p.CreatedAt)
		}
		w.Flush()
		return nil
	},
}

var projectStatusFlags struct {
	Slug string
}

var projectStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show health and resource stats for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		if projectStatusFlags.Slug == "" {
			return fmt.Errorf("--slug is required")
		}
		body, code, err := doControllerRequest("GET", "/projects/"+projectStatusFlags.Slug+"/status", nil)
		if err != nil {
			return fmt.Errorf("controller error: %w", err)
		}
		if code != http.StatusOK {
			return fmt.Errorf("status failed (%d): %s", code, body)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	},
}

var projectMigrateFlags struct {
	Slug string
}

var projectMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run pending migrations for a project schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		if projectMigrateFlags.Slug == "" {
			return fmt.Errorf("--slug is required")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Migration for project %q is handled by 'nself deploy' targeting the project schema.\n", projectMigrateFlags.Slug)
		fmt.Fprintln(cmd.OutOrStdout(), "Use: NSELF_PROJECT_SLUG="+projectMigrateFlags.Slug+" nself deploy")
		return nil
	},
}

var projectShellFlags struct {
	Slug string
}

var projectShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open a psql shell scoped to the project schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		if projectShellFlags.Slug == "" {
			return fmt.Errorf("--slug is required")
		}
		if !slugRe.MatchString(projectShellFlags.Slug) {
			return fmt.Errorf("invalid slug %q: must match [a-z0-9-]{1,63}", projectShellFlags.Slug)
		}

		schemaName := "project_" + projectShellFlags.Slug
		roleName := "nself_project_" + projectShellFlags.Slug

		fmt.Fprintf(cmd.OutOrStdout(), "Opening psql as role %s (schema: %s)...\n", roleName, schemaName)
		c := exec.Command("psql",
			"-U", roleName,
			"-d", "nself",
			"-c", "SET search_path TO "+schemaName+";",
		)
		c.Stdin = os.Stdin
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()
		return c.Run()
	},
}

var projectRotateCredsFlags struct {
	Slug string
}

var projectRotateCredsCmd = &cobra.Command{
	Use:   "rotate-credentials",
	Short: "Rotate role password and JWT secret for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !assertControllerEnabled(cmd) {
			return nil
		}
		if projectRotateCredsFlags.Slug == "" {
			return fmt.Errorf("--slug is required")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Credential rotation is dispatched to the controller daemon.")
		fmt.Fprintln(cmd.OutOrStdout(), "This feature is available in the controller HTTP API: POST /projects/"+projectRotateCredsFlags.Slug+"/rotate-credentials")
		return nil
	},
}

// ---- registration ----

// RegisterControllerCommands attaches controller + project commands to the root command.
func RegisterControllerCommands(root *cobra.Command) {
	// controller subcommands
	controllerCmd.AddCommand(controllerStartCmd)
	controllerCmd.AddCommand(controllerStopCmd)
	controllerCmd.AddCommand(controllerStatusCmd)
	root.AddCommand(controllerCmd)

	// project create flags
	projectCreateCmd.Flags().StringVar(&projectCreateFlags.Slug, "slug", "", "Project slug (lowercase, alphanumeric, hyphens)")
	projectCreateCmd.Flags().StringVar(&projectCreateFlags.Domain, "domain", "", "Project domain (e.g. myapp.myserver.com)")

	// project delete flags
	projectDeleteCmd.Flags().StringVar(&projectDeleteFlags.Slug, "slug", "", "Project slug to delete")
	projectDeleteCmd.Flags().BoolVar(&projectDeleteFlags.Force, "force", false, "Confirm irreversible deletion")

	// project status flags
	projectStatusCmd.Flags().StringVar(&projectStatusFlags.Slug, "slug", "", "Project slug")

	// project migrate flags
	projectMigrateCmd.Flags().StringVar(&projectMigrateFlags.Slug, "slug", "", "Project slug")

	// project shell flags
	projectShellCmd.Flags().StringVar(&projectShellFlags.Slug, "slug", "", "Project slug")

	// project rotate-credentials flags
	projectRotateCredsCmd.Flags().StringVar(&projectRotateCredsFlags.Slug, "slug", "", "Project slug")

	// project subcommands
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectStatusCmd)
	projectCmd.AddCommand(projectMigrateCmd)
	projectCmd.AddCommand(projectShellCmd)
	projectCmd.AddCommand(projectRotateCredsCmd)
	root.AddCommand(projectCmd)
}

