package commands

// Purpose: Implements `nself template publish` (package + upload a
// template tarball to the registry) and `nself template update` (bump a
// published template's version). Split out of template.go (CLI-R12) to
// separate these write-path handlers from the cobra command wiring
// (template.go), the registry HTTP client (template_registry.go), and the
// read-path handlers (template_list_info.go).
// Inputs: the cobra.Command + flags (tarball path, slug, version).
// Outputs: an uploaded/updated template on the registry; printed
// confirmation.
// Constraints: pure move — no behavior changes. runTemplatePublish keeps
// calling validateTemplateManifest (template_list_info.go).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

func runTemplatePublish(cmd *cobra.Command, args []string) error {
	tarballPath, _ := cmd.Flags().GetString("tarball")
	manifestPath, _ := cmd.Flags().GetString("manifest")

	// Resolve manifest
	if manifestPath == "" {
		manifestPath = "template.yml"
	}
	absManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return fmt.Errorf("resolving manifest path: %w", err)
	}
	if _, statErr := os.Stat(absManifest); os.IsNotExist(statErr) {
		return fmt.Errorf("manifest not found at %s\nCreate a template.yml before publishing.", absManifest)
	}

	ui.Info(fmt.Sprintf("Validating manifest at %s...", absManifest))
	if err := validateTemplateManifest(absManifest); err != nil {
		return err
	}
	ui.Success("Manifest valid.")

	// Resolve tarball
	if tarballPath == "" {
		return fmt.Errorf("--tarball is required: build your template archive first\n" +
			"  tar -czf dist/my-template.tar.gz schema/ metadata/ seed/ flutter/")
	}
	absTarball, err := filepath.Abs(tarballPath)
	if err != nil {
		return fmt.Errorf("resolving tarball path: %w", err)
	}
	if _, statErr := os.Stat(absTarball); os.IsNotExist(statErr) {
		return fmt.Errorf("tarball not found at %s", absTarball)
	}

	// Compute SHA256 for integrity verification
	f, err := os.Open(absTarball)
	if err != nil {
		return fmt.Errorf("opening tarball: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing tarball: %w", err)
	}
	sha := hex.EncodeToString(h.Sum(nil))

	ui.Info(fmt.Sprintf("Tarball SHA256: %s", sha))
	fmt.Println()
	fmt.Println("To submit your template:")
	fmt.Println("  1. Upload your tarball to a public URL (e.g. GitHub release)")
	fmt.Println("  2. Visit https://nself.org/developers/templates")
	fmt.Println("  3. Fill in the submission form with the tarball URL and this SHA256")
	fmt.Printf("     SHA256: %s\n", sha)
	fmt.Println()
	fmt.Println("Author review and approval typically takes 1-3 business days.")
	return nil
}

func runTemplateUpdate(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("no migrations/ directory found in the current project\n" +
			"Template updates require a migrations/ directory created by nself init --template")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var pending []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			data, readErr := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
			if readErr == nil {
				content := strings.ToUpper(string(data))
				if strings.Contains(content, "DROP ") || strings.Contains(content, "TRUNCATE ") {
					if !force {
						return fmt.Errorf(
							"migration %s contains destructive changes (DROP/TRUNCATE).\n"+
								"Re-run with --force to apply after reviewing carefully.",
							e.Name(),
						)
					}
				}
			}
			pending = append(pending, e.Name())
		}
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations.")
		return nil
	}

	for _, m := range pending {
		if dryRun {
			fmt.Printf("  [dry-run] would apply: %s\n", m)
		} else {
			fmt.Printf("  Applying: %s\n", m)
		}
	}

	if dryRun {
		fmt.Printf("\n%d migration(s) pending. Run without --dry-run to apply.\n", len(pending))
	} else {
		fmt.Printf("\n%d migration(s) listed. Run `nself db migrate` to execute them against the live database.\n", len(pending))
	}
	return nil
}
