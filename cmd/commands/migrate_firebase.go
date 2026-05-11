package commands

import (
	"fmt"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// migrateFirebaseCmd implements `nself migrate firebase`.
// This is a stub — full implementation is targeted for v1.2.0.
var migrateFirebaseCmd = &cobra.Command{
	Use:   "firebase",
	Short: "Migrate a Firebase project to nSelf (stub — v1.2.0 target)",
	Long: `Pull Firestore collections, Firebase Auth users, and Firebase Storage
objects from a Firebase project and generate nSelf-compatible migration artifacts.

This command is not yet implemented. Firebase migration is targeted for v1.2.0.

See the roadmap: https://github.com/nself-org/cli/wiki/Roadmap

If you need Firebase migration sooner, open an issue or subscribe to the tracking
issue for vote-based prioritisation.`,
	RunE: runMigrateFirebase,
}

func init() {
	migrateFirebaseCmd.Flags().String("service-account", "", "Path to Firebase service account JSON (required when implemented)")
	migrateCmd.AddCommand(migrateFirebaseCmd)
}

func runMigrateFirebase(_ *cobra.Command, _ []string) error {
	ui.CommandHeader("nself migrate firebase", "Firebase migration — coming in v1.2.0")
	fmt.Println()
	ui.Warn("Firebase migration v1.2.0 target — see roadmap")
	fmt.Println()
	ui.Info("This command is not yet implemented.")
	ui.Info("Planned capabilities:")
	fmt.Printf("  - Pull Firestore collections and documents\n")
	fmt.Printf("  - Pull Firebase Auth users\n")
	fmt.Printf("  - Pull Firebase Storage objects\n")
	fmt.Printf("  - Generate nSelf SQL migrations and Hasura metadata\n")
	fmt.Printf("  - Generate auth user import script\n")
	fmt.Println()
	ui.Info("Track or vote on the issue: https://github.com/nself-org/cli/wiki/Roadmap")
	fmt.Println()
	return fmt.Errorf("firebase migration is not yet implemented (v1.2.0 target)")
}
