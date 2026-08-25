package commands

// Purpose: RunE implementations for "nself config show" and "nself config
// get". Inputs are the cobra command/args; outputs are printed config values
// or an error.
// Constraints: split out of config.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func runConfigShow(cmd *cobra.Command, args []string) error {
	reveal, _ := cmd.Flags().GetBool("reveal")
	envFlag, _ := cmd.Flags().GetString("env")
	format, _ := cmd.Flags().GetString("format")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)
	pairs, err := godotenv.Read(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env file not found: %s", envFile)
		}
		return fmt.Errorf("reading %s: %w", envFile, err)
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	switch format {
	case "yaml":
		for _, k := range keys {
			v := maskValue(k, pairs[k], reveal)
			if strings.ContainsAny(v, ": \t#\"'\\") || v == "" {
				fmt.Printf("%s: %q\n", k, v)
			} else {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
	case "json":
		m := make(map[string]string, len(pairs))
		for _, k := range keys {
			m[k] = maskValue(k, pairs[k], reveal)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	default: // table
		for _, k := range keys {
			fmt.Printf("%s=%s\n", k, maskValue(k, pairs[k], reveal))
		}
	}
	return nil
}

// --- S4-T02: config get ---

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	reveal, _ := cmd.Flags().GetBool("reveal")
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)
	pairs, err := godotenv.Read(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env file not found: %s", envFile)
		}
		return fmt.Errorf("reading %s: %w", envFile, err)
	}

	val, ok := pairs[key]
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	fmt.Println(maskValue(key, val, reveal))
	return nil
}

// --- S4-T03: config set ---
