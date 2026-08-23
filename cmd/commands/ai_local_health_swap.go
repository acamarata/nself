package commands

// Purpose: the "nself ai local health" and "nself ai local swap" subcommands
// and their RunE. Inputs are the cobra command/args; outputs are printed
// health/swap results or an error.
// Constraints: split out of ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	aiHealthWatch bool
	aiHealthJSON  bool
)

var aiLocalHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show Ollama + plugin-ai health",
	RunE:  runAILocalHealth,
}

func runAILocalHealth(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	for {
		body, status, err := aiPluginRequest(ctx, "GET", "/ai/local/status", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin-ai unreachable: %v\n", err)
			if !aiHealthWatch {
				return err
			}
		} else {
			if aiHealthJSON {
				fmt.Println(string(body))
			} else {
				fmt.Printf("HTTP %d\n%s\n", status, string(body))
			}
		}
		if !aiHealthWatch {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

var (
	swapTask   string
	swapReason string
)

var aiLocalSwapCmd = &cobra.Command{
	Use:   "swap <model>",
	Short: "Hot-swap the default model for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAILocalSwap,
}

func runAILocalSwap(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	tasks := []string{"chat"}
	if swapTask != "" && swapTask != "all" {
		tasks = strings.Split(swapTask, ",")
	}
	if swapTask == "all" {
		tasks = []string{"chat", "embed", "classify"}
	}
	payload, _ := json.Marshal(map[string]any{
		"model":  args[0],
		"tasks":  tasks,
		"reason": swapReason,
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/local/swap-model", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("swap failed %d: %s", status, string(body))
	}
	fmt.Printf("Swapped to %s for tasks=%s\n", args[0], strings.Join(tasks, ","))
	return nil
}
