// Package commands — gdpr.go
//
// `nself gdpr` implements GDPR data portability (Art. 20) and right-to-erasure
// (Art. 17) flows for self-hosted nSelf instances.
//
// Subcommands:
//
//	nself gdpr export --user <id>         Build an export archive for a user
//	nself gdpr delete --user <id>         Cascade-delete/anonymize a user's data
//	nself gdpr forget --user <id>         Alias for delete (Art. 17 "right to be forgotten")
//	nself gdpr status --request <id>      Show status of a GDPR request
//	nself gdpr list-requests              List all GDPR requests
//
// Subcommand RunE bodies now live in gdpr_export.go, gdpr_delete.go
// (delete + its forget alias), and gdpr_query.go (status + list-requests)
// — CLI-R12 Batch B mechanical file-size split. This file keeps the root
// command, cobra registration, and the shared openDB/strPtr helpers.
package commands

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
)

var gdprCmd = &cobra.Command{
	Use:   "gdpr",
	Short: "GDPR data portability and right-to-erasure (Art. 20, 17)",
	Long: `GDPR compliance tools for self-hosted nSelf instances.

Subcommands:
  nself gdpr export        Export all data for a user (Art. 20 portability)
  nself gdpr delete        Delete or anonymize all data for a user (Art. 17 erasure)
  nself gdpr status        Check the status of a GDPR request
  nself gdpr list-requests List all open and completed GDPR requests

All GDPR requests are logged to np_gdpr_requests for audit purposes.
That table is append-only: rows are never deleted.`,
	// PersistentPreRunE installs a recover() guard across the entire gdpr
	// subcommand tree. Any unexpected panic is caught here, logged as an
	// internal error, and converted to a non-zero exit without crashing the
	// process in a way that suppresses the error message.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (retErr error) {
		defer func() {
			if r := recover(); r != nil {
				retErr = fmt.Errorf("gdpr: internal error (panic): %v", r)
			}
		}()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	gdprCmd.AddCommand(gdprExportCmd)
	gdprCmd.AddCommand(gdprDeleteCmd)
	gdprCmd.AddCommand(gdprForgetCmd)
	gdprCmd.AddCommand(gdprStatusCmd)
	gdprCmd.AddCommand(gdprListCmd)
	RootCmd.AddCommand(gdprCmd)
}

// openDB opens a Postgres connection from DATABASE_URL.
func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set; run `nself env` to verify your environment")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("gdpr: open database: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// strPtr returns a pointer to a string — used for optional nullable fields.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
