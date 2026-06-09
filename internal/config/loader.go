package config

// loader.go — env-cascade loader; entry point for config.Load().
//
// Purpose: Orchestrate the nSelf .env file cascade: read files in precedence
//          order, pass keys through warnUnknownEnvVars, then delegate to
//          parseEnvToConfig (loader_parse_env.go) for struct population and
//          ApplyDefaults (defaults.go) for smart defaults.
// Inputs:  projectDir string — the nSelf working directory that contains .env.*
//          files. ENV os env var controls which .env.{ENV} file is loaded.
// Outputs: *Config — fully populated and defaulted configuration struct.
// Constraints: Only Load() lives here. The env var list is in
//              loader_known_vars.go; field mapping in loader_parse_env.go;
//              collectPassthrough/parseInternalRoutes/parseExtensionList in
//              loader_helpers.go.
// SPORT:   cli/internal/config — decomposed from loader.go (T-E2-06).

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Load reads the .env cascade from projectDir, populates a Config struct from
// os.Getenv, applies smart defaults, and returns the complete configuration.
//
// Cascade order (later overrides earlier):
//
//	.env.dev → .env.{ENV} → .env.secrets → .env.local → .env → .env.ai
//
// .env.ai is loaded last so the AI tier configuration (generated once by
// `nself init`, contains NSELF_MASTER_SECRET) always takes effect at plugin
// startup without requiring a separate loader. Spec: p88 §8.4.
//
// Each file is optional. Missing files are silently skipped.
func Load(projectDir string) (*Config, error) {
	// 1. Detect ENV first (needed to pick the correct .env.{ENV} file).
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}
	env = normalizeEnv(env)

	// 2. Build the file cascade.
	files := []string{
		filepath.Join(projectDir, ".env.dev"),
	}
	switch env {
	case "staging":
		files = append(files, filepath.Join(projectDir, ".env.staging"))
	case "prod":
		files = append(files, filepath.Join(projectDir, ".env.prod"))
	}
	files = append(files,
		filepath.Join(projectDir, ".env.secrets"),
		filepath.Join(projectDir, ".env.local"),
		filepath.Join(projectDir, ".env"),
		// P88 Sprint 01 T-01-10: AI tier config is loaded last so AI_* vars
		// always reach plugin-ai at startup. Contains NSELF_MASTER_SECRET.
		filepath.Join(projectDir, ".env.ai"),
	)

	// 3. Load each file (skip if not exists, later overrides earlier).
	// Simultaneously collect all keys that appear in any .env file so we can
	// warn about unknown vars after loading is complete.
	const maxEnvFileSize = 1 << 20 // 1 MB
	loadedFromFiles := make(map[string]string)
	for _, f := range files {
		info, statErr := os.Stat(f)
		if statErr != nil {
			continue // file doesn't exist, skip
		}
		if info.Size() > maxEnvFileSize {
			return nil, fmt.Errorf("config file too large (max 1MB): %s", f)
		}
		contents, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		if pos := bytes.IndexByte(contents, 0); pos >= 0 {
			return nil, fmt.Errorf("invalid .env file: contains null byte at position %d in %s", pos, f)
		}
		if err := godotenv.Overload(f); err != nil {
			return nil, fmt.Errorf("loading %s: %w", f, err)
		}
		// godotenv.Read parses the file without touching os.Environ, giving
		// us the raw key set to check for unknown vars.
		if m, err := godotenv.Read(f); err == nil {
			for k, v := range m {
				loadedFromFiles[k] = v
			}
		}
	}

	// Warn about env var names that are not in the known schema.
	// Warnings go to stderr via slog.Warn and never cause a non-zero exit.
	warnUnknownEnvVars(loadedFromFiles, knownEnvVars)

	// 4. Parse os.Environ into Config struct.
	cfg := parseEnvToConfig()

	// 5. Parse dynamic collections.
	customServices, err := parseCustomServices()
	if err != nil {
		return nil, err
	}
	cfg.CustomServices = customServices
	frontendApps, err := parseFrontendApps()
	if err != nil {
		return nil, err
	}
	cfg.FrontendApps = frontendApps
	remoteSchemas, err := parseRemoteSchemas()
	if err != nil {
		return nil, err
	}
	cfg.RemoteSchemas = remoteSchemas
	cfg.InternalRoutes = parseInternalRoutes()

	// 6. Collect passthrough vars (AUTH_PROVIDER_*, REMOTE_SCHEMA_*, HASURA_EXTRA_*).
	cfg.Passthrough = collectPassthrough(os.Environ())

	// 7. Apply smart defaults (fills every unset field).
	cfg, err = ApplyDefaults(cfg)
	if err != nil {
		return nil, fmt.Errorf("applying defaults: %w", err)
	}

	// 8. Sanitize user-supplied name and domain after defaults.
	if cfg.ProjectName != "" {
		sanitized, err := SanitizeName(cfg.ProjectName)
		if err != nil {
			return nil, fmt.Errorf("PROJECT_NAME: %w", err)
		}
		if sanitized != cfg.ProjectName {
			slog.Warn("PROJECT_NAME sanitized for Docker compatibility",
				"original", cfg.ProjectName,
				"sanitized", sanitized,
			)
		}
		cfg.ProjectName = sanitized
	}
	if cfg.BaseDomain != "" {
		sanitized, err := SanitizeDomain(cfg.BaseDomain)
		if err != nil {
			return nil, fmt.Errorf("BASE_DOMAIN: %w", err)
		}
		cfg.BaseDomain = sanitized
	}

	return cfg, nil
}
