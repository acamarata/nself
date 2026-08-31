package scaffold

// scaffold_render.go — template rendering and naming helpers.
//
// Purpose: render a named template to a string, and convert plugin names into PascalCase identifiers, env var prefixes and license text, used throughout scaffold.go, split out for file size.
// Inputs: a template name/body and template data, or a raw plugin name/tier.
// Outputs: a rendered string, or a derived identifier/license string.
// Constraints: pure move from scaffold.go (CLI-R12 Batch F); no behaviour change.

import (
	"fmt"
	"strings"
	"text/template"
)

// render executes a Go template with the given params, returning the result.
// Returns an error if template parsing or execution fails.
func render(tmpl string, p Params) (string, error) {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("scaffold: template parse error: %w", err)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, p); err != nil {
		return "", fmt.Errorf("scaffold: template execute error: %w", err)
	}
	return buf.String(), nil
}

// renderAnyErr is like renderAny but returns an error instead of panicking.
// Used in paths where the caller can propagate the error (e.g. buildFiles).
func renderAnyErr(tmpl string, data any) (string, error) {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("scaffold: template parse error: %w", err)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("scaffold: template execute error: %w", err)
	}
	return buf.String(), nil
}

// --- helpers ---

func toPascal(s string) string {
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func toEnvPrefix(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_"))
}

func licenseForTier(tier string) string {
	if tier == "pro" {
		return "Source-Available"
	}
	return "MIT"
}
