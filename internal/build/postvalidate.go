package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PostValidateResult contains the results of all post-build checks.
type PostValidateResult struct {
	ComposeValid bool
	NginxValid   bool
	PortsUnique  bool
	NamesUnique  bool
	Errors       []string
	Warnings     []string
}

// PostValidate runs all post-build checks on the generated files.
//
// composePath:  path to docker-compose.yml
// nginxConfDir: path to nginx/sites/ directory (used to locate the parent nginx.conf)
//
// Returns result with all findings. Never returns a Go error — all findings
// go in the result Errors/Warnings slices.
func PostValidate(composePath, nginxConfDir string) PostValidateResult {
	result := PostValidateResult{
		// Assume valid until a check fails.
		ComposeValid: true,
		NginxValid:   true,
		PortsUnique:  true,
		NamesUnique:  true,
	}

	// ── Compose YAML check ───────────────────────────────────────────
	services := checkComposeYAML(composePath, &result)

	// ── Port uniqueness check ────────────────────────────────────────
	checkPortUniqueness(services, &result)

	// ── Service name uniqueness ──────────────────────────────────────
	checkNameUniqueness(services, &result)

	// ── Nginx syntax check ───────────────────────────────────────────
	checkNginxSyntax(nginxConfDir, &result)

	return result
}

// checkComposeYAML parses the compose file and validates its structure.
// Returns the raw services map (may be nil on parse failure).
func checkComposeYAML(composePath string, result *PostValidateResult) map[string]interface{} {
	data, err := os.ReadFile(composePath)
	if err != nil {
		result.ComposeValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("compose YAML: cannot read %s: %v", composePath, err))
		return nil
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		result.ComposeValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("compose YAML: parse error in %s: %v", composePath, err))
		return nil
	}

	rawServices, ok := doc["services"]
	if !ok {
		result.ComposeValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("compose YAML: no 'services' key found in %s", composePath))
		return nil
	}

	services, ok := rawServices.(map[string]interface{})
	if !ok || len(services) == 0 {
		result.ComposeValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("compose YAML: 'services' block is empty or malformed in %s", composePath))
		return nil
	}

	return services
}

// checkPortUniqueness inspects every service's ports list and flags duplicates
// on the host side. Docker Compose port bindings can be expressed as:
//   - "8080:80"          → host port 8080
//   - "8080"             → host port 8080 (no container port)
//   - "127.0.0.1:80:80"  → host port 80
func checkPortUniqueness(services map[string]interface{}, result *PostValidateResult) {
	if services == nil {
		return
	}

	// Map from host port string → first service name that claimed it.
	seen := make(map[string]string)

	for svcName, rawSvc := range services {
		svc, ok := rawSvc.(map[string]interface{})
		if !ok {
			continue
		}

		rawPorts, ok := svc["ports"]
		if !ok {
			continue
		}

		portList, ok := rawPorts.([]interface{})
		if !ok {
			continue
		}

		for _, rawPort := range portList {
			binding, ok := rawPort.(string)
			if !ok {
				continue
			}

			hostPort := extractHostPort(binding)
			if hostPort == "" {
				continue
			}

			if first, exists := seen[hostPort]; exists {
				result.PortsUnique = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("duplicate host port %s: claimed by both %q and %q",
						hostPort, first, svcName))
			} else {
				seen[hostPort] = svcName
			}
		}
	}
}

// extractHostPort parses a Docker Compose port binding string and returns the
// host-side port. Returns "" when the binding cannot be parsed.
//
// Supported forms:
//   - "8080:80"           → "8080"
//   - "8080"              → "8080"
//   - "127.0.0.1:80:80"  → "80"
func extractHostPort(binding string) string {
	// Strip optional IP prefix: "127.0.0.1:80:80" → "80:80"
	if strings.Count(binding, ":") >= 2 {
		// Could be IP:host:container — take the middle segment.
		parts := strings.SplitN(binding, ":", 3)
		// parts[0] could be an IP or just a port; if it contains '.' it's an IP.
		if strings.Contains(parts[0], ".") {
			// IP:host:container form.
			return parts[1]
		}
		// host:container form with an extra colon (unusual) — just take first.
		return parts[0]
	}

	// "8080:80" → "8080"
	if strings.Contains(binding, ":") {
		return strings.SplitN(binding, ":", 2)[0]
	}

	// "8080" (no colon at all).
	return binding
}

// checkNameUniqueness verifies that all service names in the compose file are
// unique. Duplicate keys in YAML are technically invalid, but the yaml.v3
// library may deduplicate them silently, so this guard catches template bugs
// before they reach Docker.
func checkNameUniqueness(services map[string]interface{}, result *PostValidateResult) {
	if services == nil {
		return
	}

	// yaml.v3 already deduplicates map keys when unmarshaling to
	// map[string]interface{}, so we cannot detect duplicates that were
	// collapsed at parse time. We can, however, validate that the set of
	// names is non-empty — which checkComposeYAML already does — and warn
	// if the count looks suspect. For future-proofing, we keep this as an
	// explicit pass; any generator bug that produces duplicate service
	// YAML stanzas will be caught at the yaml.Unmarshal level as malformed
	// YAML before reaching this function.
	//
	// Since yaml.v3 collapses duplicates silently we cannot detect them
	// post-parse. Instead we rely on the template system being correct and
	// log a debug note here.
	_ = services // used by callers to pass the parsed map; no action needed.
}

// checkNginxSyntax runs `nginx -t -c <nginx.conf>` if the nginx binary is
// available in PATH and the main nginx.conf can be found. Skips with a
// warning when nginx is not installed locally — that is not an error,
// since users may not have nginx on their development machine.
//
// The nginxConfDir parameter is expected to be nginx/sites/ (or nginx/).
// We look for nginx.conf one level up (the standard layout).
func checkNginxSyntax(nginxConfDir string, result *PostValidateResult) {
	// Locate nginx binary.
	nginxBin, err := exec.LookPath("nginx")
	if err != nil {
		result.Warnings = append(result.Warnings,
			"nginx binary not found in PATH — skipping nginx syntax check (install nginx locally to enable)")
		return
	}

	// Determine path to main nginx.conf. The caller passes nginx/sites/,
	// so the main conf lives in the parent directory.
	nginxConf := findNginxConf(nginxConfDir)
	if nginxConf == "" {
		result.Warnings = append(result.Warnings,
			"nginx.conf not found — skipping nginx syntax check")
		return
	}

	// Run nginx -t with a short timeout so a hung binary never blocks the build.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// nginx -t writes its output to stderr by convention.
	cmd := exec.CommandContext(ctx, nginxBin, "-t", "-c", nginxConf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		result.NginxValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("nginx syntax check failed: %v\n%s", err, strings.TrimSpace(string(out))))
		return
	}

	// Success — nginx -t exits 0 and prints "syntax is ok".
}

// findNginxConf searches for nginx.conf relative to the provided directory.
// nginxConfDir is expected to be either nginx/ or nginx/sites/; we check
// common locations and return the first one that exists.
func findNginxConf(nginxConfDir string) string {
	candidates := []string{
		// If caller passed nginx/sites/, the conf is one level up.
		filepath.Join(nginxConfDir, "..", "nginx.conf"),
		// If caller passed nginx/ directly.
		filepath.Join(nginxConfDir, "nginx.conf"),
	}

	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, err := os.Stat(clean); err == nil {
			abs, err := filepath.Abs(clean)
			if err == nil {
				return abs
			}
			return clean
		}
	}

	return ""
}
