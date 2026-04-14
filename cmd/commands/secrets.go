package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/secrets"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var secretsEnvFlag string

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encrypted project secrets (age encryption)",
	Long: `Manage encrypted secrets for nSelf projects using age encryption.

Secrets are stored as age-encrypted JSON files per environment in .secrets/.
Each team member has their own age keypair; secrets are encrypted to all
team members' public keys.

Subcommands:
  init             Generate age key and set up .secrets/
  set <KEY>        Set a secret value
  get <KEY>        Get a secret value
  list             List all secrets
  edit             Open decrypted secrets in $EDITOR
  rotate <KEY>     Rotate a secret value
  decrypt-on-deploy Output secrets as KEY=VALUE for CI/CD
  audit            Report secrets needing rotation (>90 days)
  lint             Check for plaintext secrets in source code
  rekey            Re-encrypt removing a team member's key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var secretsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate age key and set up .secrets/ directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		return secrets.Init(cwd)
	},
}

var secretsSetCmd = &cobra.Command{
	Use:   "set <KEY> [VALUE]",
	Short: "Set a secret value (prompts if value not provided)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		key := args[0]
		var value string
		if len(args) > 1 {
			value = args[1]
		} else {
			fmt.Printf("Enter value for %s: ", key)
			var v string
			if _, err := fmt.Scanln(&v); err != nil {
				return fmt.Errorf("reading value: %w", err)
			}
			value = v
		}
		if err := secrets.Set(cwd, secretsEnvFlag, key, value); err != nil {
			return err
		}
		fmt.Printf("Secret %s set in %s environment.\n", key, secretsEnvFlag)
		return nil
	},
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <KEY>",
	Short: "Get a secret value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		value, err := secrets.Get(cwd, secretsEnvFlag, args[0])
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil
	},
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets for an environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		keys, entries, err := secrets.List(cwd, secretsEnvFlag)
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			fmt.Printf("No secrets in %s environment.\n", secretsEnvFlag)
			return nil
		}

		tbl := ui.NewTable("Key", "Updated", "Rotated")
		for _, k := range keys {
			e := entries[k]
			rotated := "-"
			if e.RotatedAt != "" {
				rotated = e.RotatedAt[:10]
			}
			updated := "-"
			if e.UpdatedAt != "" {
				updated = e.UpdatedAt[:10]
			}
			tbl.AddRow(k, updated, rotated)
		}
		tbl.Render()
		fmt.Printf("\n%d secrets in %s environment.\n", len(keys), secretsEnvFlag)
		return nil
	},
}

var secretsEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open decrypted secrets in $EDITOR, re-encrypt on save",
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Decrypt to temp file.
		keys, entries, err := secrets.List(cwd, secretsEnvFlag)
		if err != nil {
			return err
		}
		var lines []string
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s=%s", k, entries[k].Value))
		}

		tmpFile, err := os.CreateTemp("", "nself-secrets-*.env")
		if err != nil {
			return err
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := tmpFile.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
			return err
		}
		tmpFile.Close()

		// Open editor.
		c := exec.Command(editor, tmpPath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		// Re-read and save.
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			if err := secrets.Set(cwd, secretsEnvFlag, parts[0], parts[1]); err != nil {
				return fmt.Errorf("setting %s: %w", parts[0], err)
			}
		}
		fmt.Println("Secrets updated.")
		return nil
	},
}

var secretsRotateCmd = &cobra.Command{
	Use:   "rotate <KEY>",
	Short: "Rotate a secret to a new value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dualWindow, _ := cmd.Flags().GetBool("dual-window")
		if dualWindow {
			if err := secrets.RotateDualWindow(cwd, secretsEnvFlag, args[0]); err != nil {
				return err
			}
			fmt.Printf("Secret %s rotated with dual-key window. Both _CURRENT and _PREVIOUS are active.\n", args[0])
			return nil
		}
		newValue, err := secrets.Rotate(cwd, secretsEnvFlag, args[0])
		if err != nil {
			return err
		}
		if newValue == "" {
			fmt.Println("Secret requires manual rotation through the provider.")
		} else {
			fmt.Printf("Secret %s rotated. New value: %s\n", args[0], newValue)
		}
		return nil
	},
}

var secretsRetireCmd = &cobra.Command{
	Use:   "retire <KEY>",
	Short: "Retire the _PREVIOUS variant of a dual-window rotated secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := secrets.RetireOldKey(cwd, secretsEnvFlag, args[0]); err != nil {
			return err
		}
		fmt.Printf("Retired %s_PREVIOUS. Only the new key remains.\n", args[0])
		return nil
	},
}

var secretsScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Show rotation schedule status for all tracked secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		checks, err := secrets.CheckSchedule(cwd)
		if err != nil {
			return err
		}
		if len(checks) == 0 {
			fmt.Println("No rotation schedules configured.")
			return nil
		}
		tbl := ui.NewTable("Secret", "Cadence", "Window", "Next Due", "Due In", "Status")
		for _, c := range checks {
			dueIn := "-"
			if c.DueInDays >= 0 {
				dueIn = fmt.Sprintf("%dd", c.DueInDays)
			}
			nextDue := c.NextDue
			if len(nextDue) > 10 {
				nextDue = nextDue[:10]
			}
			if nextDue == "" {
				nextDue = "-"
			}
			tbl.AddRow(c.SecretName, fmt.Sprintf("%dd", c.CadenceDays),
				fmt.Sprintf("%dd", c.WindowDays), nextDue, dueIn, c.Status)
		}
		tbl.Render()
		return nil
	},
}

var secretsDecryptOnDeployCmd = &cobra.Command{
	Use:   "decrypt-on-deploy",
	Short: "Output decrypted secrets as KEY=VALUE for CI/CD",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		output, err := secrets.DecryptForDeploy(cwd, secretsEnvFlag)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	},
}

var secretsAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Report secrets that haven't been rotated in >90 days",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		findings, err := secrets.Audit(cwd, secretsEnvFlag)
		if err != nil {
			return err
		}
		if len(findings) == 0 {
			fmt.Println("All secrets are within rotation policy.")
			return nil
		}
		tbl := ui.NewTable("Key", "Issue", "Severity")
		for _, f := range findings {
			tbl.AddRow(f.Key, f.Issue, f.Severity)
		}
		tbl.Render()
		fmt.Printf("\n%d finding(s) in %s environment.\n", len(findings), secretsEnvFlag)
		return nil
	},
}

var secretsLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Check for plaintext secrets in git-tracked files",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		findings, err := secrets.LintSecrets(cwd)
		if err != nil {
			return err
		}
		if len(findings) == 0 {
			fmt.Println("No plaintext secrets detected.")
			return nil
		}
		tbl := ui.NewTable("File", "Line", "Rule", "Description")
		for _, f := range findings {
			tbl.AddRow(f.File, fmt.Sprintf("%d", f.Line), f.Rule, f.Message)
		}
		tbl.Render()
		fmt.Printf("\n%d finding(s).\n", len(findings))
		return nil
	},
}

var secretsRekeyCmd = &cobra.Command{
	Use:   "rekey --remove <pubkey>",
	Short: "Re-encrypt all secrets, removing a team member's key",
	RunE: func(cmd *cobra.Command, args []string) error {
		removePubKey, _ := cmd.Flags().GetString("remove")
		if removePubKey == "" {
			return fmt.Errorf("--remove flag is required (specify the public key to remove)")
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		return secrets.Rekey(cwd, removePubKey)
	},
}

func init() {
	// Global --env flag for all secrets subcommands.
	secretsCmd.PersistentFlags().StringVar(&secretsEnvFlag, "env", "dev", "Environment (dev, staging, prod)")

	secretsRekeyCmd.Flags().String("remove", "", "Public key to remove from recipients")
	secretsRotateCmd.Flags().Bool("dual-window", false, "Keep old key as _PREVIOUS during overlap window")

	secretsCmd.AddCommand(secretsInitCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsEditCmd)
	secretsCmd.AddCommand(secretsRotateCmd)
	secretsCmd.AddCommand(secretsRetireCmd)
	secretsCmd.AddCommand(secretsScheduleCmd)
	secretsCmd.AddCommand(secretsDecryptOnDeployCmd)
	secretsCmd.AddCommand(secretsAuditCmd)
	secretsCmd.AddCommand(secretsLintCmd)
	secretsCmd.AddCommand(secretsRekeyCmd)

	RootCmd.AddCommand(secretsCmd)
}
