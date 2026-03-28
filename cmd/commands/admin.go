package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Open the nSelf Admin dashboard in your browser",
	Long:  `Open the local nSelf Admin UI (http://localhost:3021) in your default browser.`,
	RunE:  runAdmin,
}

func init() {
	RootCmd.AddCommand(adminCmd)
}

func runAdmin(cmd *cobra.Command, args []string) error {
	port := "3021"
	if envPort := os.Getenv("ADMIN_PORT"); envPort != "" {
		port = envPort
	}
	adminURL := "http://localhost:" + port

	ctx := cmd.Context()

	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.CommandContext(ctx, "open", adminURL)
	case "windows":
		openCmd = exec.CommandContext(ctx, "cmd", "/c", "start", adminURL)
	default:
		openCmd = exec.CommandContext(ctx, "xdg-open", adminURL)
	}

	if err := openCmd.Start(); err != nil {
		// Graceful fallback — headless server or command not found.
		fmt.Printf("Admin UI: %s\n", adminURL)
		return nil
	}

	fmt.Printf("Opening %s in your browser...\n", adminURL)
	return nil
}
