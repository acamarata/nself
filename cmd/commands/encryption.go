package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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

// ──────────────────────────────────────────────────────────────────────────
// nself encryption rotate
// ──────────────────────────────────────────────────────────────────────────

var encryptionRotateDryRunFlag bool

var encryptionRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate data encryption keys",
	Long: `Re-encrypts all stored data with fresh DEKs wrapped by your current CMK.
Use this after rotating the CMK in your KMS provider.

Use --dry-run to estimate record counts without performing encryption.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if encryptionRotateDryRunFlag {
			fmt.Println("Dry-run: key rotation would re-encrypt all records for this tenant. Pass without --dry-run to execute.")
			return nil
		}
		baseURL, err := resolveByokBaseURL()
		if err != nil {
			return err
		}
		status, resp, err := doByokRequest(http.MethodPost, baseURL+"/api/v1/encryption/rotate", nil)
		if err != nil {
			return fmt.Errorf("trigger rotation: %w", err)
		}
		if status != http.StatusAccepted {
			return fmt.Errorf("rotation failed (HTTP %d): %s", status, resp)
		}
		fmt.Printf("Key rotation started: %s\n", resp)
		fmt.Println("Track progress with: nself encryption status")
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────────────
// nself encryption status
// ──────────────────────────────────────────────────────────────────────────

var encryptionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show BYOK configuration and last verification",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, err := resolveByokBaseURL()
		if err != nil {
			return err
		}
		status, resp, err := doByokRequest(http.MethodGet, baseURL+"/api/v1/encryption/kms", nil)
		if err != nil {
			return fmt.Errorf("get status: %w", err)
		}
		if status == http.StatusNotFound {
			fmt.Println("BYOK: not configured. Run: nself encryption configure --provider <aws|gcp|vault> ...")
			return nil
		}
		if status != http.StatusOK {
			return fmt.Errorf("status failed (HTTP %d): %s", status, resp)
		}
		fmt.Println(resp)
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────────────
// nself encryption key-events
// ──────────────────────────────────────────────────────────────────────────

var encryptionKeyEventsCmd = &cobra.Command{
	Use:   "key-events",
	Short: "List the key event audit trail",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, err := resolveByokBaseURL()
		if err != nil {
			return err
		}
		status, resp, err := doByokRequest(http.MethodGet, baseURL+"/api/v1/encryption/key-events", nil)
		if err != nil {
			return fmt.Errorf("list key events: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("key-events failed (HTTP %d): %s", status, resp)
		}
		fmt.Println(resp)
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────

// resolveByokBaseURL returns the BYOK plugin base URL from env.
// Gate: NSELF_BYOK must be "true" (Enterprise tier required).
// Falls back to the nSelf API base if BYOK_PLUGIN_URL is not set.
func resolveByokBaseURL() (string, error) {
	// BYOK/Enterprise gate — Security-Always-Free Doctrine applies to security
	// scans, not to BYOK which is genuinely Enterprise-only.
	if os.Getenv("NSELF_BYOK") != "true" {
		return "", fmt.Errorf(
			"BYOK encryption requires an Enterprise license (set NSELF_BYOK=true).\n" +
				"Contact support@nself.org to upgrade, or run 'nself license status'.",
		)
	}
	if u := os.Getenv("BYOK_PLUGIN_URL"); u != "" {
		return u, nil
	}
	if u := os.Getenv("NSELF_API_URL"); u != "" {
		return u, nil
	}
	return "http://localhost:3741", nil
}

// resolveKeyRef returns (provider, keyRef) from CLI flags.
func resolveKeyRef() (string, string, error) {
	provider := encProviderFlag
	if provider == "" {
		return "", "", fmt.Errorf("--provider is required (aws, gcp, or vault)")
	}
	var keyRef string
	switch provider {
	case "aws":
		keyRef = encKeyIDFlag
		if keyRef == "" {
			return "", "", fmt.Errorf("--key-id is required for AWS KMS")
		}
	case "gcp":
		keyRef = encKeyNameFlag
		if keyRef == "" {
			return "", "", fmt.Errorf("--key-name is required for GCP Cloud KMS")
		}
	case "vault":
		keyRef = encKeyPathFlag
		if keyRef == "" {
			return "", "", fmt.Errorf("--key-path is required for HashiCorp Vault")
		}
	default:
		return "", "", fmt.Errorf("unsupported provider %q (must be aws, gcp, or vault)", provider)
	}
	return provider, keyRef, nil
}

// doByokRequest performs a JSON HTTP request to the BYOK plugin API.
// Returns (statusCode, responseBody, error).
func doByokRequest(method, url string, body any) (int, string, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("marshal request: %w", err)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tid := os.Getenv("NSELF_TENANT_ID"); tid != "" {
		req.Header.Set("X-Tenant-ID", tid)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("HTTP %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), nil
}

// ──────────────────────────────────────────────────────────────────────────
// init
// ──────────────────────────────────────────────────────────────────────────

func init() {
	// configure flags
	encryptionConfigureCmd.Flags().StringVar(&encProviderFlag, "provider", "", "KMS provider: aws, gcp, or vault (required)")
	encryptionConfigureCmd.Flags().StringVar(&encKeyIDFlag, "key-id", "", "AWS KMS key ARN or alias")
	encryptionConfigureCmd.Flags().StringVar(&encKeyNameFlag, "key-name", "", "GCP Cloud KMS key resource path")
	encryptionConfigureCmd.Flags().StringVar(&encKeyPathFlag, "key-path", "", "HashiCorp Vault Transit key path")
	encryptionConfigureCmd.Flags().StringVar(&encEndpointFlag, "endpoint", "", "Vault endpoint URL")
	encryptionConfigureCmd.Flags().StringVar(&encRegionFlag, "region", "", "AWS/GCP region")
	encryptionConfigureCmd.Flags().StringVar(&encCredRefFlag, "credentials-ref", "", "np_secrets key name for KMS credentials")

	// rotate flags
	encryptionRotateCmd.Flags().BoolVar(&encryptionRotateDryRunFlag, "dry-run", false, "Estimate record counts without re-encrypting")

	// assemble subcommands
	encryptionCmd.AddCommand(encryptionConfigureCmd)
	encryptionCmd.AddCommand(encryptionVerifyCmd)
	encryptionCmd.AddCommand(encryptionRotateCmd)
	encryptionCmd.AddCommand(encryptionStatusCmd)
	encryptionCmd.AddCommand(encryptionKeyEventsCmd)

	RootCmd.AddCommand(encryptionCmd)
}
