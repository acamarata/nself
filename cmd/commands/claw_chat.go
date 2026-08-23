package commands

import (
	"github.com/spf13/cobra"
)

var (
	clawChatTopic   string
	clawChatModel   string
	clawChatResume  bool
	clawChatSession string
)

var clawChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session with nClaw",
	Long: `Open an interactive terminal chat session with your nClaw AI assistant.

Type messages at the ɳ> prompt. AI responses stream with markdown rendering.

REPL commands:
  /exit, /quit   Exit the session
  /topic <name>  Switch topic context
  /model <name>  Switch model
  /memory        Show recent memories
  /clear         Clear screen
  /help          Show available commands

Ctrl+C cancels the current generation.
Ctrl+D exits the session.`,
	RunE: runClawChat,
}

func init() {
	clawChatCmd.Flags().StringVar(&clawChatTopic, "topic", "", "Start in specific topic")
	clawChatCmd.Flags().StringVar(&clawChatModel, "model", "", "Use specific model")
	clawChatCmd.Flags().BoolVar(&clawChatResume, "resume", false, "Resume last conversation")
	clawChatCmd.Flags().StringVar(&clawChatSession, "session", "", "Resume specific session ID")
}
