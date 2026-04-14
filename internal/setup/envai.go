// Package setup — .env.ai generator for P88 Zero-Config AI (Sprint 01, T-01-09).
//
// .env.ai is the canonical home for AI-tier configuration and the master secret
// used to encrypt OAuth refresh tokens and pooled GCP API keys. It is written
// once by `nself init` and must never be overwritten by subsequent runs — the
// master secret is irrecoverable and all encrypted material would become
// unreadable. Spec: p88-block-a-zero-config-ai-spec.md §8.1 / §8.3 / §8.5.
package setup

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// envAITemplate is the exact content written to .env.ai on first init.
// Values match spec §8.1 verbatim. NSELF_MASTER_SECRET is replaced with a
// fresh 32-byte random string at generation time.
const envAITemplate = `# .env.ai — P88 Zero-Config AI configuration
#
# This file is written once by "nself init" and MUST NOT be edited by hand
# unless you understand the implications. NSELF_MASTER_SECRET is the KEK
# that protects OAuth refresh tokens and pooled GCP API keys — losing or
# changing it after keys have been stored will make them unreadable.
#
# Load order: appended to the env cascade after .env.computed so AI_* vars
# always land in the plugin-ai process environment at startup.

# ----- AI profile -----
AI_PROFILE=auto
AI_AUTO_INSTALL=true

# ----- Local (Ollama) models -----
AI_DEFAULT_MODEL=gemma2:2b
AI_EMBEDDING_MODEL=nomic-embed-text

# ----- Pool (zero-config GCP) -----
AI_POOL_AUTO_PROVISION=true
AI_BACKGROUND_LOCAL_ONLY=true

# ----- Budgets (0 = unlimited) -----
AI_DAILY_BUDGET_USD=0
AI_MONTHLY_BUDGET_USD=0

# ----- Per-tier timeouts (ms, 0 = no cap) -----
AI_TIMEOUT_LOCAL_MS=0
AI_TIMEOUT_OAUTH_MS=0
AI_TIMEOUT_POOL_MS=0
AI_TIMEOUT_PAID_MS=0

# ----- Optional: bring-your-own OAuth client (empty = use nself-hosted app) -----
AI_POOL_OAUTH_CLIENT_ID=
AI_POOL_OAUTH_CLIENT_SECRET=

# ----- Master secret (KEK) — DO NOT REGENERATE -----
NSELF_MASTER_SECRET=%s
`

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

// writeEnvAI creates projectDir/.env.ai with perm 0600. It uses O_EXCL so a
// pre-existing .env.ai is preserved (returning ok=false, err=nil). The master
// secret is freshly generated per invocation from system entropy — verified
// unique across runs by sprint acceptance criteria.
//
// Returns:
//
//	ok=true  — file created
//	ok=false — file already existed (not an error; caller should continue)
//	err      — any other I/O or entropy failure
func writeEnvAI(projectDir string) (ok bool, err error) {
	path := filepath.Join(projectDir, ".env.ai")

	secret, err := generateMasterSecret()
	if err != nil {
		return false, fmt.Errorf("generate master secret: %w", err)
	}
	content := fmt.Sprintf(envAITemplate, secret)

	// O_EXCL: fail if the file already exists. This is the core anti-clobber
	// guarantee — once NSELF_MASTER_SECRET is on disk, init must never touch
	// it again. Mode 0600: owner read/write only (spec §8.3).
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			// Already present — preserve and signal caller.
			return false, nil
		}
		return false, fmt.Errorf("open .env.ai: %w", err)
	}
	defer f.Close()

	if _, werr := f.WriteString(content); werr != nil {
		// Best-effort cleanup of a partially-written file.
		_ = os.Remove(path)
		return false, fmt.Errorf("write .env.ai: %w", werr)
	}

	// Double-check perm (umask can narrow but never widen O_CREATE mode).
	if err := os.Chmod(path, 0600); err != nil {
		// Non-fatal on platforms where Chmod is a no-op, but surface the error.
		return true, fmt.Errorf("chmod .env.ai 0600: %w", err)
	}
	return true, nil
}

// (.env.ai is added to .gitignore via the shared gitignoreEntries list in
// setup.go — no separate helper needed.)
