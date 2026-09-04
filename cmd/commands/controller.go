// Package commands: controller + project subcommands for B46 multi-tenant master controller.
//
// CLI surface:
//
//	nself controller start | stop | status | init
//	nself project create --slug <slug> --domain <domain>
//	nself project delete --slug <slug> [--force]
//	nself project list
//	nself project status --slug <slug>
//	nself project migrate --slug <slug>
//	nself project shell --slug <slug>
//	nself project rotate-credentials --slug <slug>
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
	"regexp"
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
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
			"503 Multi-tenant controller not enabled.\n"+
				"Set NSELF_FLAG_MULTI_TENANT_CONTROLLER=true to enable.")
		return false
	}
	return true
}

// CreateProjectRequest is the typed body for POST /projects/create, replacing
// the untyped map[string]string literal previously built inline at the call
// site (P4 deferred-backlog row 2 — no generic [T any] request-builder
// pattern exists elsewhere in this CLI to mirror, so each doControllerRequest
// call site gets its own typed struct instead).
type CreateProjectRequest struct {
	Slug   string
	Domain string
}

// doControllerRequest performs an authenticated HTTP request to the controller daemon.
// body is intentionally interface{}: it is marshalled straight to JSON, and
// every call site now passes either nil or a typed *Request struct (see
// CreateProjectRequest) rather than an ad hoc map literal.
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
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// ---- controller commands ----

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
