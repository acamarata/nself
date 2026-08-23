package commands

// Purpose: RunE implementations for "nself db shell/list/reset/drop" plus the
// requireProductionConfirmation guard shared by the destructive ones. Inputs
// are the cobra command/args; outputs are printed results or an error.
// Constraints: split out of db.go (CLI-R12) as a pure move, no behavior change.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/docker"

	"github.com/spf13/cobra"
)

func runDBShell(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	psqlCmd := []string{"psql", "-U", user, "-d", db}
	opts := docker.ExecOptions{
		Interactive: true,
		TTY:         true,
	}

	return docker.Exec(cmd.Context(), container, psqlCmd, opts)
}

// runDBList implements 'nself db list'. It queries the Postgres container for
// all database names and prints them one per line.
func runDBList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	// -t: tuples only, -A: unaligned, -c: inline SQL — produces one db name per line.
	listSQL := "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname"
	args := []string{
		"exec", container,
		"psql", "-U", user, "-d", "postgres", "-t", "-A", "-c", listSQL,
	}
	c := exec.CommandContext(cmd.Context(), "docker", args...)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	if err := c.Run(); err != nil {
		return fmt.Errorf("db list failed: %w", err)
	}
	return nil
}

func runDBReset(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	yesFlag, _ := cmd.Flags().GetBool("yes")
	skipConfirm := dbResetForce || yesFlag

	// Production environments require typing the project name to confirm.
	if cfg.IsProduction() && !skipConfirm {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	} else if !skipConfirm {
		// Non-production: simple yes/no confirmation.
		fmt.Fprintf(os.Stderr, "WARNING: This will drop and recreate database %q. Type \"yes\" to confirm: ", db)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		if strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Fprintln(os.Stderr, "aborted")
			return fmt.Errorf("reset aborted by user")
		}
	}

	// Terminate existing connections and drop the database, then recreate.
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	terminateSQL := fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
		strings.ReplaceAll(db, "'", "''"),
	)
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", db)
	createSQL := fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\"", db, user)

	for _, sql := range []string{terminateSQL, dropSQL, createSQL} {
		args := []string{
			"exec", container,
			"psql", "-U", user, "-d", "postgres", "-c", sql,
		}
		c := exec.CommandContext(cmd.Context(), "docker", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("reset failed on %q: %w", sql, err)
		}
	}

	// Re-initialize schemas and extensions.
	if err := database.InitializeDatabase(cmd.Context(), cfg); err != nil {
		return fmt.Errorf("reinitialize after reset: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Database reset complete.")
	return nil
}

// requireProductionConfirmation prints a production warning and requires the
// user to type the project name exactly to proceed. Returns an error if the
// confirmation fails or the input cannot be read.
func requireProductionConfirmation(projectName string) error {
	fmt.Printf("WARNING: PRODUCTION: This will DESTROY the database %s.\n", projectName)
	fmt.Print("   Type the project name to confirm: ")
	var confirm string
	if _, err := fmt.Scanln(&confirm); err != nil {
		return fmt.Errorf("reading confirmation: %w", err)
	}
	if strings.TrimSpace(confirm) != projectName {
		return fmt.Errorf("confirmation failed: got %q, expected %q", confirm, projectName)
	}
	return nil
}

func runDBDrop(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	yesFlag, _ := cmd.Flags().GetBool("yes")

	if cfg.IsProduction() && !yesFlag {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	} else if !yesFlag {
		fmt.Fprintf(os.Stderr, "WARNING: This will drop database %q. Type \"yes\" to confirm: ", db)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if scanErr := scanner.Err(); scanErr != nil {
			return fmt.Errorf("reading confirmation: %w", scanErr)
		}
		if strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Fprintln(os.Stderr, "aborted")
			return fmt.Errorf("drop aborted by user")
		}
	}

	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	terminateSQL := fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
		strings.ReplaceAll(db, "'", "''"),
	)
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", db)

	for _, sql := range []string{terminateSQL, dropSQL} {
		args := []string{
			"exec", container,
			"psql", "-U", user, "-d", "postgres", "-c", sql,
		}
		c := exec.CommandContext(cmd.Context(), "docker", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("drop failed on %q: %w", sql, err)
		}
	}

	fmt.Fprintln(os.Stderr, "Database dropped.")
	return nil
}
