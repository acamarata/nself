package commands

// Purpose: the "nself project" parent command plus "create" and "delete".
// Inputs are the cobra command/args; outputs are a created/deleted tenant
// project or an error.
// Constraints: split out of controller.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

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

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Creating project %q at %s...\n",
			projectCreateFlags.Slug, projectCreateFlags.Domain)

		body, code, err := doControllerRequest("POST", "/projects/create", CreateProjectRequest{
			Slug:   projectCreateFlags.Slug,
			Domain: projectCreateFlags.Domain,
		})
		if err != nil {
			return fmt.Errorf("controller error: %w", err)
		}
		if code != http.StatusCreated {
			var e struct {
				Error string `json:"error"`
			}
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project ready: https://%s (id: %s)\n", proj.Domain, proj.ID)
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
			var e struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(body, &e)
			return fmt.Errorf("delete failed (%d): %s", code, e.Error)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project %q deleted.\n", projectDeleteFlags.Slug)
		return nil
	},
}
