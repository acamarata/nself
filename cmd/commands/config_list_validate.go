package commands

// Purpose: RunE implementations for "nself config list" and "nself config
// validate". Inputs are the cobra command/args; outputs are printed config
// listings/validation results or an error.
// Constraints: split out of config.go (CLI-R12) as a pure move, no behavior change.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func runConfigList(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)

	// Read the env file if it exists. Missing file is not an error for list.
	var pairs map[string]string
	if _, statErr := os.Stat(envFile); statErr == nil {
		pairs, err = godotenv.Read(envFile)
		if err != nil {
			return fmt.Errorf("reading %s: %w", envFile, err)
		}
	} else {
		pairs = make(map[string]string)
	}

	known := config.KnownEnvVars()

	// Print header.
	fmt.Printf("%-45s %-30s %s\n", "KEY", "VALUE", "SOURCE")
	fmt.Println(strings.Repeat("-", 90))

	for _, k := range known {
		val, ok := pairs[k]
		source := filepath.Base(envFile)
		displayVal := val
		if !ok || val == "" {
			displayVal = config.DefaultFor(k)
			if displayVal != "" {
				displayVal = "(default: " + displayVal + ")"
				source = "default"
			} else {
				displayVal = "(unset)"
				source = "unset"
			}
		} else {
			displayVal = maskValue(k, val, false)
		}
		fmt.Printf("%-45s %-30s %s\n", k, displayVal, source)
	}
	return nil
}

// --- S4-T05: config validate ---

func runConfigValidate(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	// For validate, use the cascade loader so all defaults apply.
	// Temporarily honour --env by setting ENV if provided.
	if envFlag != "" {
		os.Setenv("ENV", envFlag)
	}

	cfg, err := config.Load(projectDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Run validators individually to report per-validator pass/fail.
	results := config.RunAllWithResults(cfg)
	errorCount := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %s\n", r.Name, r.Err.Error())
			errorCount++
		} else {
			fmt.Printf("[PASS] %s\n", r.Name)
		}
	}

	if errorCount == 0 {
		fmt.Println("config OK")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n%d error(s) found\n", errorCount)
	return fmt.Errorf("one or more validators failed")
}

// --- S4-T06: config export ---
