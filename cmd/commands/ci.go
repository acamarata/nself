package commands

import (
	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "CI/CD operations and gates",
	Long:  "CI/CD operations and gates for ɳSelf",
}

func init() {
	RootCmd.AddCommand(ciCmd)
	ciCmd.AddCommand(ciEvalCmd)
}
