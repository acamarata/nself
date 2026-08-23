package setup

// envai.go — AI-tier config block, folded into .env.secrets (CLI-R18).
//
// Purpose: Write the AI-tier config vars (AI_* + NSELF_MASTER_SECRET) that
//          used to live in a dedicated .env.ai cascade layer. CLI-R18 (GATE
//          B, 2026-08-23) eliminated .env.ai entirely: the vars now live
//          inside .env.secrets, the file they always belonged with (never
//          committed, host-local, already outranks bare .env under the new
//          cascade order).
// Inputs:  projectDir — the nSelf working directory.
// Outputs: writeAIConfig creates or extends projectDir/.env.secrets.
// Constraints: Must never regenerate NSELF_MASTER_SECRET once one exists —
//              it is the KEK for OAuth/API-key encryption; regenerating it
//              makes all previously-encrypted material unreadable. Written
//              file must be mode 0600 (P15 regression: leaked secrets at
//              0644). Spec: p88-block-a-zero-config-ai-spec.md §8.1/§8.3/§8.5
//              (original .env.ai design; CLI-R18 supersedes only the file
//              location, not the content or the anti-clobber guarantee).
// SPORT:   cli/internal/setup — CLI-R18 env cascade canon.

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// envAIBlock is the AI-tier config block appended to .env.secrets on first
// init (or on upgrade, via the migration shim in internal/migrate). Values
// match the original .env.ai template verbatim. NSELF_MASTER_SECRET is
// replaced with a fresh 32-byte random string at generation time.
const envAIBlock = `
# ----- AI profile (CLI-R18: folded from the retired .env.ai cascade layer) -----
# Written once by "nself init" and MUST NOT be edited by hand unless you
# understand the implications. NSELF_MASTER_SECRET is the KEK that protects
# OAuth refresh tokens and pooled GCP API keys — losing or changing it after
# keys have been stored will make them unreadable.
AI_PROFILE=auto
AI_AUTO_INSTALL=true
AI_DEFAULT_MODEL=gemma2:2b
AI_EMBEDDING_MODEL=nomic-embed-text
AI_POOL_AUTO_PROVISION=true
AI_BACKGROUND_LOCAL_ONLY=true
AI_DAILY_BUDGET_USD=0
AI_MONTHLY_BUDGET_USD=0
AI_TIMEOUT_LOCAL_MS=0
AI_TIMEOUT_OAUTH_MS=0
AI_TIMEOUT_POOL_MS=0
AI_TIMEOUT_PAID_MS=0
AI_POOL_OAUTH_CLIENT_ID=
AI_POOL_OAUTH_CLIENT_SECRET=
NSELF_MASTER_SECRET=%s
`

// masterSecretMarker is the line prefix used to detect whether the AI config
// block (and, critically, an existing NSELF_MASTER_SECRET) has already been
// written to .env.secrets — the anti-clobber guard that used to be provided
// by .env.ai's O_EXCL create.
const masterSecretMarker = "NSELF_MASTER_SECRET="

// generateMasterSecret returns a base64url-encoded 32-byte value (256 bits of
// entropy). No padding, URL-safe alphabet — safe inside .env without quoting.
// This is the KEK that wraps all OAuth/API-key DEKs in the zero-config pool.
func generateMasterSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// writeAIConfig ensures projectDir/.env.secrets carries the AI-tier config
// block. It creates .env.secrets (mode 0600) if it does not exist yet, or
// appends the block to it if it exists but has no NSELF_MASTER_SECRET yet.
// If NSELF_MASTER_SECRET is already present, the file is left untouched —
// this is the anti-clobber guarantee .env.ai used to provide via O_EXCL.
//
// Returns:
//
//	ok=true  — block appended (file created or existing file extended)
//	ok=false — NSELF_MASTER_SECRET already present; nothing written
//	err      — any other I/O or entropy failure
func writeAIConfig(projectDir string) (ok bool, err error) {
	path := filepath.Join(projectDir, ".env.secrets")

	if existing, readErr := os.ReadFile(path); readErr == nil {
		if bytes.Contains(existing, []byte(masterSecretMarker)) {
			// Already present — preserve and signal caller.
			return false, nil
		}
	}

	secret, err := generateMasterSecret()
	if err != nil {
		return false, fmt.Errorf("generate master secret: %w", err)
	}
	block := fmt.Sprintf(envAIBlock, secret)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return false, fmt.Errorf("open .env.secrets: %w", err)
	}
	defer f.Close()

	if _, werr := f.WriteString(block); werr != nil {
		return false, fmt.Errorf("write AI config to .env.secrets: %w", werr)
	}

	// Double-check perm (umask can narrow but never widen O_CREATE mode).
	if err := os.Chmod(path, 0600); err != nil {
		// Non-fatal on platforms where Chmod is a no-op, but surface the error.
		return true, fmt.Errorf("chmod .env.secrets 0600: %w", err)
	}
	return true, nil
}
