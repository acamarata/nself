// Package setup implements the nself init wizard: project scaffolding,
// secret generation, .env file creation, and .nself/ directory setup.
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

// Options holds all flags and settings for the init command.
type Options struct {
	Fast           bool
	Interactive    bool
	NonInteractive bool
	Template       string
	SkipValidation bool
	Wizard         bool
	Demo           bool
	Full           bool
	Force          bool
	Quiet          bool
	WorkDir        string
	Name           string
	// Domain is the BASE_DOMAIN value. When set (e.g. via --domain flag or
	// interactive wizard), it takes precedence over env vars and defaults.
	Domain string
	// DomainComment is written as a comment above BASE_DOMAIN in the generated
	// .env. It describes the chosen domain pattern for operator clarity.
	DomainComment string
	// NoPgvector skips pgvector extension and RAG scaffold on init (sets
	// PGVECTOR_ENABLED=false). Default (false) enables pgvector.
	NoPgvector bool
}

// Result holds the outcome of a successful init run.
type Result struct {
	ProjectName  string
	BaseDomain   string
	Env          string
	FilesCreated []string
	Demo         bool
}

// projectNameRe validates PROJECT_NAME: lowercase alphanumeric + hyphens,
// must start and end with alphanumeric, 2-30 chars.
var projectNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// Initialize runs the full init flow: validate inputs, generate secrets,
// write .env files, create .nself/ directory, and return a Result.
func Initialize(opts Options) (*Result, error) {
	if opts.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determining working directory: %w", err)
		}
		opts.WorkDir = wd
	}

	// Check for existing .env unless --force.
	envPath := filepath.Join(opts.WorkDir, ".env")
	if !opts.Force {
		if _, err := os.Stat(envPath); err == nil {
			return nil, fmt.Errorf(".env already exists in %s — use --force to overwrite", opts.WorkDir)
		}
	}

	// Determine project name: env var > directory name > default.
	projectName := resolveProjectName(opts)
	if !opts.SkipValidation {
		if err := validateProjectName(projectName); err != nil {
			return nil, err
		}
	}

	// Determine environment and domain.
	env := resolveEnv(opts)
	baseDomain := resolveBaseDomain(opts, env)

	// Generate secrets.
	postgresPassword, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating postgres password: %w", err)
	}
	hasuraAdminSecret, err := GenerateSecret(44)
	if err != nil {
		return nil, fmt.Errorf("generating hasura admin secret: %w", err)
	}
	hasuraJWTKey, err := GenerateSecret(44)
	if err != nil {
		return nil, fmt.Errorf("generating hasura JWT key: %w", err)
	}

	// Generate a secure MinIO password (never use the known-bad defaults).
	minioPassword, err := resolveMinioPassword("")
	if err != nil {
		return nil, fmt.Errorf("generating minio password: %w", err)
	}

	// Generate plugin secrets so they are persisted to .env on first init.
	notifySecret, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating notify secret: %w", err)
	}
	cronSecret, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating cron secret: %w", err)
	}
	pluginInternalSecret, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating plugin internal secret: %w", err)
	}

	// Build .env content.
	pgvectorEnabled := !opts.NoPgvector
	var envContent string
	if opts.Demo {
		envContent = buildDemoEnv(projectName, baseDomain, env, postgresPassword, hasuraAdminSecret, hasuraJWTKey, minioPassword, notifySecret, cronSecret, pluginInternalSecret, opts.DomainComment, pgvectorEnabled)
	} else {
		envContent = buildStandardEnv(projectName, baseDomain, env, postgresPassword, hasuraAdminSecret, hasuraJWTKey, notifySecret, cronSecret, pluginInternalSecret, opts.DomainComment, pgvectorEnabled)
	}

	// Write .env.
	if err := writeFile(envPath, envContent, 0600); err != nil {
		return nil, fmt.Errorf("writing .env: %w", err)
	}
	filesCreated := []string{".env"}

	// Write .env.example (safe to commit, no secrets).
	examplePath := filepath.Join(opts.WorkDir, ".env.example")
	exampleContent := buildExampleEnv(projectName, baseDomain, env, opts.Demo)
	if err := writeFile(examplePath, exampleContent, 0644); err != nil {
		return nil, fmt.Errorf("writing .env.example: %w", err)
	}
	filesCreated = append(filesCreated, ".env.example")

	// --full: write environment-specific files.
	if opts.Full {
		fullFiles, err := writeFullEnvFiles(opts.WorkDir, projectName, baseDomain)
		if err != nil {
			return nil, fmt.Errorf("writing environment files: %w", err)
		}
		filesCreated = append(filesCreated, fullFiles...)
	}

	// P88 Sprint 01 T-01-09 / CLI-R18: fold AI tier config (+ master secret)
	// into .env.secrets rather than a separate .env.ai cascade layer.
	// Anti-clobber: preserved across re-runs — the master secret must never
	// rotate once written.
	if created, err := writeAIConfig(opts.WorkDir); err != nil {
		return nil, fmt.Errorf("writing AI config: %w", err)
	} else if created && !slices.Contains(filesCreated, ".env.secrets") {
		filesCreated = append(filesCreated, ".env.secrets")
	}

	// Append to .gitignore.
	if err := ensureGitignore(opts.WorkDir); err != nil {
		return nil, fmt.Errorf("updating .gitignore: %w", err)
	}
	filesCreated = append(filesCreated, ".gitignore")

	// Create .nself/ working directory.
	nelfDir := filepath.Join(opts.WorkDir, ".nself")
	if err := os.MkdirAll(nelfDir, 0755); err != nil {
		return nil, fmt.Errorf("creating .nself directory: %w", err)
	}

	return &Result{
		ProjectName:  projectName,
		BaseDomain:   baseDomain,
		Env:          env,
		FilesCreated: filesCreated,
		Demo:         opts.Demo,
	}, nil
}
