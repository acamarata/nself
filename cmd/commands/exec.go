package commands

import (
	"fmt"

	"nself/internal/docker"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [FLAGS] <SERVICE> [COMMAND...]",
	Short: "Execute a command inside a service container",
	Long: `Execute a command inside a running service container.

If no COMMAND is given, a default is chosen based on the service:
  postgres  → psql -U postgres
  redis     → redis-cli
  (other)   → /bin/sh

Examples:
  nself exec postgres                     # opens psql
  nself exec redis                        # opens redis-cli
  nself exec postgres -- pg_dump mydb     # run pg_dump
  nself exec -u root nginx /bin/bash      # shell as root
  cat file.sql | nself exec -T postgres psql  # pipe SQL via stdin`,
	Args: cobra.MinimumNArgs(1),
	RunE: runExec,
}

func init() {
	execCmd.Flags().BoolP("no-tty", "T", false, "Disable pseudo-TTY allocation")
	execCmd.Flags().StringP("user", "u", "", "Run command as this user")
	execCmd.Flags().StringP("workdir", "w", "", "Working directory inside the container")
	execCmd.Flags().StringSliceP("env", "e", nil, "Set environment variables (KEY=VALUE)")

	RootCmd.AddCommand(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	noTTY, _ := cmd.Flags().GetBool("no-tty")
	user, _ := cmd.Flags().GetString("user")
	workdir, _ := cmd.Flags().GetString("workdir")
	envVars, _ := cmd.Flags().GetStringSlice("env")

	service := args[0]
	command := args[1:]

	// If no command specified, use the service-specific default.
	if len(command) == 0 {
		command = docker.DefaultCommand(service)
	}

	opts := docker.ExecOptions{
		Interactive: true,
		TTY:         !noTTY,
		User:        user,
		WorkDir:     workdir,
		Env:         envVars,
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Executing in %s: %v\n", service, command)

	if err := docker.Exec(cmd.Context(), service, command, opts); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}
