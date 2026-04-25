package commands

import (
	"github.com/nself-org/cli/cmd/aistudio"
)

// init registers the `nself ai-studio` command subtree with the root CLI.
func init() {
	RootCmd.AddCommand(aistudio.AiStudioCmd)
}
