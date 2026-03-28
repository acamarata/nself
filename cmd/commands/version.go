package commands

import (
	"fmt"
	"runtime"

	"nself/internal/ui"
	"nself/internal/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and system information",
	RunE: func(cmd *cobra.Command, args []string) error {
		short, _ := cmd.Flags().GetBool("short")
		jsonOut, _ := cmd.Flags().GetBool("json")

		ver := version.GetVersion()
		commit := version.GetCommit()
		buildDate := version.GetBuildDate()
		goVer := runtime.Version()
		platform := runtime.GOOS + "/" + runtime.GOARCH

		if short {
			fmt.Println(ver)
			return nil
		}

		if jsonOut {
			return ui.PrintJSON(map[string]string{
				"version":   ver,
				"commit":    commit,
				"buildDate": buildDate,
				"goVersion": goVer,
				"platform":  platform,
			})
		}

		// Default verbose output
		fmt.Printf("%s %s\n", ui.C(ui.Bold, "nSelf CLI"), ui.C(ui.Green, ver))
		fmt.Printf("  Commit:     %s\n", commit)
		fmt.Printf("  Built:      %s\n", buildDate)
		fmt.Printf("  Go version: %s\n", goVer)
		fmt.Printf("  Platform:   %s\n", platform)

		return nil
	},
}

func init() {
	versionCmd.Flags().Bool("short", false, "Print version number only")
	versionCmd.Flags().Bool("json", false, "Print version info as JSON")
	RootCmd.AddCommand(versionCmd)
}
