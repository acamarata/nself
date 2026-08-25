package doctor

// dogfood_hexcolors.go — brand-token compliance scan (DOGFOOD-HEX-01), split
// out of dogfood_checks.go (CLI-R12) as a pure move.
//
// Inputs: the project directory (checkNoHexColorsInSrc) or a directory root
// to walk (scanHexColors, filepathWalk).
// Outputs: one CheckResult per dogfood subapp, or a list of offending
// relative file paths.
// Constraints: depends on dogfoodSubapps, defined in dogfood_checks.go.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// hexColorRegex matches a 3- or 6-digit hex color literal in source code.
// Used to enforce the brand token convention (no raw hex; use design tokens).
var hexColorRegex = regexp.MustCompile(`#[0-9a-fA-F]{6}\b|#[0-9a-fA-F]{3}\b`)

// checkNoHexColorsInSrc scans every dogfood subapp's src/ tree for raw hex
// color literals. Brand convention (per F15) requires using design tokens.
// CSS files inside .vendor/ or generated/ subtrees are skipped.
func checkNoHexColorsInSrc(projectDir string) []CheckResult {
	var results []CheckResult
	for _, name := range dogfoodSubapps {
		if name == "backend" {
			continue
		}
		checkName := fmt.Sprintf("DOGFOOD-HEX-01: no hex colors in %s/src", name)
		srcDir := filepath.Join(projectDir, name, "src")
		if _, err := os.Stat(srcDir); err != nil {
			// Subapp without a src/ tree (e.g., legacy structure) — skip.
			continue
		}
		offenders := scanHexColors(srcDir)
		if len(offenders) > 0 {
			// Cap message length so we don't flood JSON output.
			msg := fmt.Sprintf("%d file(s) contain hex color literals", len(offenders))
			if len(offenders) <= 3 {
				msg += ": " + strings.Join(offenders, ", ")
			}
			results = append(results, CheckResult{
				Section: "dogfood", Name: checkName, Status: "warn",
				Message: msg,
				FixCmd:  "replace hex colors with design tokens from packages/ui/tokens",
			})
			continue
		}
		results = append(results, CheckResult{
			Section: "dogfood", Name: checkName, Status: "pass",
			Message: "no hex colors found",
		})
	}
	return results
}

// scanHexColors walks a directory tree and returns relative paths of files
// containing hex color literals. .vendor/, node_modules/, .next/, build/,
// dist/, and generated/ subtrees are skipped.
func scanHexColors(root string) []string {
	var offenders []string
	skipDirs := map[string]bool{
		"node_modules": true, ".next": true, "build": true, "dist": true,
		"generated": true, ".vendor": true, "vendor": true, ".turbo": true,
	}
	scannedExts := map[string]bool{
		".css": true, ".scss": true, ".ts": true, ".tsx": true,
		".js": true, ".jsx": true, ".vue": true,
	}
	_ = filepathWalk(root, func(path string, isDir bool) error {
		base := filepath.Base(path)
		if isDir {
			if skipDirs[base] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !scannedExts[ext] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if hexColorRegex.Find(data) != nil {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			offenders = append(offenders, rel)
		}
		return nil
	})
	return offenders
}

// filepathWalk wraps filepath.Walk with a simpler callback signature.
// Errors reading individual entries are swallowed so a single permission-
// denied node does not abort the entire scan — matches doctor's
// "best-effort scan" posture. filepath.SkipDir from the callback is passed
// through unchanged so callers can short-circuit subtrees.
func filepathWalk(root string, fn func(path string, isDir bool) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip unreadable entries; never abort the scan.
			return nil
		}
		return fn(path, info.IsDir())
	})
}
