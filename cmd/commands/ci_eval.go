package commands

import (
	"github.com/spf13/cobra"
)

var ciEvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate quality gates for CI/CD",
	Long:  "Evaluate quality gates for CI/CD pipelines in ɳSelf",
}

func init() {
	ciEvalCmd.AddCommand(ciEvalGateCmd)
}
