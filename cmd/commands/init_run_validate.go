package commands

// init_run_validate.go — flag validation/sanitization for `nself init`.
//
// Purpose: handles --list-presets (early exit), validates --cs-template and
//          --preset, and sanitizes --name/--domain before they enter the
//          config system. Split out of init_run.go (T-P6-E2-W1-S1-T3) for
//          300-line compliance.
// Inputs:  the relevant runInit flag values.
// Outputs: sanitized name/domain, a done flag (true = runInit must return nil
//          immediately, the --list-presets case), and any validation error.
// Constraints: pure move — same checks, same error strings, same order.

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/scaffold"
	"github.com/nself-org/cli/internal/ui"
)

// validateAndSanitizeInitFlags validates --cs-template/--preset and sanitizes
// --name/--domain. When listPresets is set it prints the preset catalog and
// signals the caller to return nil immediately via done=true.
func validateAndSanitizeInitFlags(csTemplate, preset, name, domainFlag string, listPresets bool) (sanitizedName, sanitizedDomain string, done bool, err error) {
	// --list-presets: print preset catalog and exit.
	if listPresets {
		listInitPresets()
		return name, domainFlag, true, nil
	}

	// Validate --cs-template if given.
	if csTemplate != "" && !scaffold.IsValidLang(csTemplate) {
		return name, domainFlag, false, fmt.Errorf("--cs-template %q is not a supported language; choose one of: %s",
			csTemplate, strings.Join(scaffold.SupportedLangs(), ", "))
	}

	// Validate --preset if given.
	if preset != "" {
		if _, ok := initPresets[preset]; !ok {
			fmt.Fprintf(os.Stderr, "%s Unknown preset %q.\n", ui.C(ui.Yellow, ui.IconWarning), preset)
			listInitPresets()
			return name, domainFlag, false, fmt.Errorf("unknown preset %q — see presets above", preset)
		}
	}

	// Sanitize user-supplied --name before it enters the config system.
	if name != "" {
		sanitized, sErr := config.SanitizeName(name)
		if sErr != nil {
			return name, domainFlag, false, fmt.Errorf("--name: PROJECT_NAME contains no valid characters after sanitization")
		}
		name = sanitized
	}

	// Sanitize user-supplied --domain before it enters the config system.
	if domainFlag != "" {
		sanitized, sErr := config.SanitizeDomain(domainFlag)
		if sErr != nil {
			return name, domainFlag, false, fmt.Errorf("--domain: BASE_DOMAIN contains no valid characters after sanitization")
		}
		domainFlag = sanitized
	}

	return name, domainFlag, false, nil
}
