package database

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// Seed executes SQL seed files against the postgres database.
// If file is specified, only that file is run. If file is empty, all *.sql files
// in the seeds/ directory are executed in alphabetical order.
func Seed(ctx context.Context, cfg *config.Config, file string) error {
	if file != "" {
		return runSeedFile(ctx, cfg, file)
	}

	// Scan seeds/ directory for .sql files.
	dir := "seeds"
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("seeds directory not found: %s", dir)
		}
		return fmt.Errorf("read seeds directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	// Apply env-based seed filtering if DB_ENV_SEEDS is configured
	dbEnvSeeds := os.Getenv("DB_ENV_SEEDS")
	if dbEnvSeeds != "" {
		files = SeedFilter(files, dbEnvSeeds, cfg.Env)
	}

	if len(files) == 0 {
		return nil
	}

	for _, f := range files {
		if err := runSeedFile(ctx, cfg, f); err != nil {
			return err
		}
	}
	return nil
}

// SeedFilter returns the subset of seedFiles that should run in currentEnv.
// DB_ENV_SEEDS format: "dev:file1.sql,staging:file2.sql"
// Files not mentioned in dbEnvSeeds run in all environments.
// Files mentioned in dbEnvSeeds only run when their env matches currentEnv.
func SeedFilter(seedFiles []string, dbEnvSeeds string, currentEnv string) []string {
	// Build env-scoped map from DB_ENV_SEEDS
	envScoped := map[string]string{} // filename → required env
	for _, entry := range strings.Split(dbEnvSeeds, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) == 2 {
			envScoped[strings.TrimSpace(parts[1])] = strings.TrimSpace(parts[0])
		}
	}

	var result []string
	for _, f := range seedFiles {
		base := filepath.Base(f)
		requiredEnv, scoped := envScoped[base]
		if !scoped || requiredEnv == currentEnv {
			result = append(result, f)
		}
	}
	return result
}

// runSeedFile pipes a single SQL file into psql inside the postgres container.
func runSeedFile(ctx context.Context, cfg *config.Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed file %s: %w", path, err)
	}

	args := []string{
		"exec", "-i", containerName(cfg),
		"psql",
		"-U", cfg.Postgres.User,
		"-d", cfg.Postgres.DB,
		"-v", "ON_ERROR_STOP=1",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("seed %s: %s: %w", filepath.Base(path), strings.TrimSpace(stderr.String()), err)
	}
	return nil
}
