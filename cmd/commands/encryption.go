package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var encryptionCmd = &cobra.Command{
	Use:   "encryption",
	Short: "Manage BYOK per-tenant encryption (Enterprise)",
	Long: `Bring Your Own Key (BYOK) encryption for nSelf Cloud.

Each tenant can supply their own Customer Managed Key (CMK) hosted in
AWS KMS, GCP Cloud KMS, or HashiCorp Vault Transit. nSelf uses envelope
encryption: data is encrypted with a Data Encryption Key (DEK), and the
DEK is wrapped by the tenant's CMK.

Requires an Enterprise license (NSELF_BYOK=true).

Subcommands:
  configure   Configure a KMS provider for the current tenant
  verify      Test KMS connectivity (wrap+unwrap round-trip)
  rotate      Rotate data encryption keys (re-wrap existing DEKs)
  status      Show current BYOK configuration and last verification
  key-events  List the key event audit trail`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ──────────────────────────────────────────────────────────────────────────
// nself encryption configure
// ──────────────────────────────────────────────────────────────────────────

var (
	encProviderFlag string
	encKeyIDFlag    string
	encKeyNameFlag  string
	encKeyPathFlag  string
	encEndpointFlag string
	encRegionFlag   string
	encCredRefFlag  string
)

var encryptionConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a KMS provider for BYOK encryption",
	Long: `Configure your KMS provider so nSelf can wrap data encryption keys
using your Customer Managed Key (CMK).

Examples:
  # AWS KMS
  nself encryption configure --provider aws --key-id arn:aws:kms:us-east-1:123456:key/abc123

  # GCP Cloud KMS
  nself encryption configure --provider gcp \
    --key-name projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key

  # HashiCorp Vault Transit
  nself encryption configure --provider vault \
    --key-path transit/keys/tenant-abc \
    --endpoint https://vault.example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, err := resolveByokBaseURL()
		if err != nil {
			return err
		}

		provider, keyRef, err := resolveKeyRef()
		if err != nil {
			return err
		}

		body := map[string]any{
			"provider": provider,
			"key_ref":  keyRef,
		}
		if encRegionFlag != "" {
			body["region"] = encRegionFlag
		}
		if encEndpointFlag != "" {
			body["endpoint_url"] = encEndpointFlag
		}
		if encCredRefFlag != "" {
			body["credentials_ref"] = encCredRefFlag
		}

		status, resp, err := doByokRequest(http.MethodPost, baseURL+"/api/v1/encryption/kms", body)
		if err != nil {
			return fmt.Errorf("configure KMS: %w", err)
		}
		if status != http.StatusCreated {
			return fmt.Errorf("configure KMS failed (HTTP %d): %s", status, resp)
		}
		fmt.Printf("KMS configured successfully (%s: %s)\n", provider, keyRef)
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────────────
// nself encryption verify
// ──────────────────────────────────────────────────────────────────────────

var encryptionVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Test KMS connectivity (wrap+unwrap round-trip)",
	Long: `Performs a wrap+unwrap round-trip against your configured KMS to confirm
nSelf has encrypt and decrypt permissions on your CMK.

Returns exit code 0 on success, 1 on failure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, err := resolveByokBaseURL()
		if err != nil {
			return err
		}
		status, resp, err := doByokRequest(http.MethodPost, baseURL+"/api/v1/encryption/kms/verify", nil)
		if err != nil {
			return fmt.Errorf("verify KMS: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("KMS verification failed (HTTP %d): %s", status, resp)
		}
		fmt.Println("KMS verification succeeded: wrap+unwrap round-trip OK")
		return nil
	},
}
