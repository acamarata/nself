package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nself-org/cli/internal/license"

	"github.com/spf13/cobra"
)

var licenseRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force-refresh license validation against ping.nself.org",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys := license.CollectLicenseKeys()
		if len(keys) == 0 {
			return fmt.Errorf("no license key configured — run 'nself license add <key>' first")
		}

		for _, key := range keys {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			result, err := license.RefreshCache(ctx, key)
			cancel()

			masked := license.MaskKey(key)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: refresh failed: %v\n", masked, err)
				continue
			}
			if !result.Valid {
				fmt.Fprintf(os.Stderr, "%s: invalid — %s\n", masked, result.Message)
				continue
			}
			fmt.Printf("%s: refreshed (tier: %s, expires: %s)\n",
				masked, result.Tier, result.ExpiresAt.Format("2006-01-02"))
		}
		return nil
	},
}

var licenseExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export signed license cache for air-gap transfer",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := license.ExportCache()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

var licenseImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a previously exported license cache file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading cache file: %w", err)
		}
		if err := license.ImportCache(data); err != nil {
			return err
		}
		fmt.Println("License cache imported successfully.")
		return nil
	},
}

func init() {
	licenseCmd.AddCommand(licenseRefreshCmd)
	licenseCmd.AddCommand(licenseExportCmd)
	licenseCmd.AddCommand(licenseImportCmd)
}
