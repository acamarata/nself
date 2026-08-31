package commands

// Purpose: the remaining "nself project" subcommands: list, status, migrate,
// shell, and rotate-credentials. Inputs are the cobra command/args; outputs
// are printed project state/results or an error.
// Constraints: split out of controller.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

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
		_, _ = fmt.Fprintf(w, "SLUG\tDOMAIN\tSTATUS\tCREATED\n")
		for _, p := range projects {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Slug, p.Domain, p.Status, p.CreatedAt)
		}
		_ = w.Flush()
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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(body))
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Migration for project %q is handled by 'nself deploy' targeting the project schema.\n", projectMigrateFlags.Slug)
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Use: NSELF_PROJECT_SLUG="+projectMigrateFlags.Slug+" nself deploy")
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

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Opening psql as role %s (schema: %s)...\n", roleName, schemaName)
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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Credential rotation is dispatched to the controller daemon.")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "This feature is available in the controller HTTP API: POST /projects/"+projectRotateCredsFlags.Slug+"/rotate-credentials")
		return nil
	},
}

// ---- registration ----
