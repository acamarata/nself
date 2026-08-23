package commands

// Purpose: the "nself ai pool remove" and "nself ai pool rotate" subcommands
// and their RunE. Inputs are the cobra command/args; outputs are a removed/
// rotated pool key, or an error.
// Constraints: split out of ai_pool.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	poolRemoveAccount string
	poolRemoveKeyID   string
)

var aiPoolRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a key from the pool (soft-revoke + optional GCP delete)",
	RunE:  runPoolRemove,
}

func runPoolRemove(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	keyID := poolRemoveKeyID
	if keyID == "" && poolRemoveAccount != "" {
		// Look up key by account
		body, _, err := aiPluginRequest(ctx, "GET", "/ai/pool/status", nil)
		if err != nil {
			return err
		}
		var ps struct {
			Keys []struct {
				KeyIndex      int    `json:"key_index"`
				GoogleAccount string `json:"google_account"`
			} `json:"keys"`
		}
		json.Unmarshal(body, &ps)
		for _, k := range ps.Keys {
			if k.GoogleAccount == poolRemoveAccount {
				keyID = fmt.Sprintf("%d", k.KeyIndex)
				break
			}
		}
		if keyID == "" {
			return fmt.Errorf("no key found for account %s", poolRemoveAccount)
		}
	}
	if keyID == "" {
		return fmt.Errorf("--key-id or --account required")
	}

	path := fmt.Sprintf("/ai/pool/keys/%s?hard=true", keyID)
	body, status, err := aiPluginRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}
	fmt.Printf("Removed key %s\n", keyID)
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool rotate`
// -----------------------------------------------------------------------------

var poolRotateKeyID string

var aiPoolRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate a key (create new GCP key, revoke old)",
	RunE:  runPoolRotate,
}

func runPoolRotate(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if poolRotateKeyID == "" {
		return fmt.Errorf("--key-id required")
	}

	payload, _ := json.Marshal(map[string]any{
		"key_id": atoi(poolRotateKeyID),
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/pool/rotate", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}

	var result struct {
		OldKeyID       int    `json:"old_key_id"`
		NewKeyID       string `json:"new_key_id"`
		NewFingerprint string `json:"new_fingerprint"`
	}
	json.Unmarshal(body, &result)
	fmt.Printf("Rotated key %d -> %s (fingerprint: %s)\n", result.OldKeyID, result.NewKeyID, result.NewFingerprint)
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool test`
// -----------------------------------------------------------------------------
