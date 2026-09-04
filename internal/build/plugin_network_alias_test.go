package build

import (
	"strings"
	"testing"
)

// TestNormalizeComposeNetworkAliases_ShortForm verifies that a service with
// the short-form networks list gets a `hostname: plugin-{name}` sibling key
// so inter-plugin DNS works with the CLI-injected PLUGIN_*_INTERNAL_URL env
// vars, and that the networks: block itself is left untouched (it is already
// valid Compose — see the doc comment on normalizeComposeNetworkAliases for
// why a map-form `${DOCKER_NETWORK}: {aliases: [...]}` rewrite is invalid).
func TestNormalizeComposeNetworkAliases_ShortForm(t *testing.T) {
	in := `services:
  mux:
    build:
      context: ${NSELF_PLUGIN_DIR}/mux
      dockerfile: Dockerfile.golang
    container_name: ${COMPOSE_PROJECT_NAME}_mux
    restart: unless-stopped
    networks:
      - ${DOCKER_NETWORK}
    ports:
      - "127.0.0.1:3711:3711"
`
	out := string(normalizeComposeNetworkAliases([]byte(in), "mux"))
	if !strings.Contains(out, "hostname: plugin-mux") {
		t.Fatalf("expected hostname: plugin-mux, got:\n%s", out)
	}
	// The networks: block must remain in its original, valid list form —
	// rewriting it into a map keyed by ${DOCKER_NETWORK} is the bug this
	// test guards against (Compose does not interpolate map keys).
	if !strings.Contains(out, "networks:\n      - ${DOCKER_NETWORK}") {
		t.Fatalf("networks: list form must be preserved unchanged, got:\n%s", out)
	}
	if strings.Contains(out, "${DOCKER_NETWORK}:") {
		t.Fatalf("must never rewrite ${DOCKER_NETWORK} into a map key, got:\n%s", out)
	}
	// Rest of the service block must remain intact.
	if !strings.Contains(out, "container_name: ${COMPOSE_PROJECT_NAME}_mux") {
		t.Fatalf("container_name line was clobbered, got:\n%s", out)
	}
	if !strings.Contains(out, `ports:`) || !strings.Contains(out, `127.0.0.1:3711:3711`) {
		t.Fatalf("ports block was clobbered, got:\n%s", out)
	}
	if err := assertValidComposeYAML(out); err != nil {
		t.Fatalf("normalized output is not valid YAML: %v\n%s", err, out)
	}
}

// TestNormalizeComposeNetworkAliases_Idempotent verifies that running the
// normalizer twice does not duplicate the hostname line.
func TestNormalizeComposeNetworkAliases_Idempotent(t *testing.T) {
	in := `services:
  ai:
    container_name: ${COMPOSE_PROJECT_NAME}_ai
    networks:
      - ${DOCKER_NETWORK}
`
	once := normalizeComposeNetworkAliases([]byte(in), "ai")
	twice := normalizeComposeNetworkAliases(once, "ai")
	if string(once) != string(twice) {
		t.Fatalf("normalizer is not idempotent:\nonce:\n%s\n---\ntwice:\n%s", string(once), string(twice))
	}
	if strings.Count(string(twice), "hostname: plugin-ai") != 1 {
		t.Fatalf("hostname line should appear exactly once, got:\n%s", string(twice))
	}
}

// TestNormalizeComposeNetworkAliases_NoShortForm verifies that a fragment
// without the short-form networks list is left untouched.
func TestNormalizeComposeNetworkAliases_NoShortForm(t *testing.T) {
	in := `services:
  custom:
    container_name: ${COMPOSE_PROJECT_NAME}_custom
    image: some/image:latest
`
	out := normalizeComposeNetworkAliases([]byte(in), "custom")
	if string(out) != in {
		t.Fatalf("fragment without networks list should be unchanged, got:\n%s", string(out))
	}
}

// TestNormalizeComposeNetworkAliases_EmptyName verifies that an empty plugin
// name is a no-op (defensive — should never happen in practice).
func TestNormalizeComposeNetworkAliases_EmptyName(t *testing.T) {
	in := `services:
  mux:
    networks:
      - ${DOCKER_NETWORK}
`
	out := normalizeComposeNetworkAliases([]byte(in), "")
	if string(out) != in {
		t.Fatalf("empty plugin name should be no-op, got:\n%s", string(out))
	}
}

// TestNormalizeComposeNetworkAliases_RealFixtureIsValidYAML is a regression
// test for P6-E3-W2-S1-T5's finding: the previous map-form rewrite produced
// YAML that parsed fine (it was syntactically valid YAML) but that Docker
// Compose's schema rejected at the semantic level
// ("additional properties '${DOCKER_NETWORK}' not allowed"), which the old
// substring-only tests never caught. Uses the real notifications plugin
// compose fragment to guard against re-introducing that class of bug.
func TestNormalizeComposeNetworkAliases_RealFixtureIsValidYAML(t *testing.T) {
	in := `# NSELF-FIRST EXCEPTION: plugin-fragment
services:
  notifications:
    image: nself/nself-notifications:latest
    container_name: ${COMPOSE_PROJECT_NAME}_notifications
    restart: unless-stopped
    networks:
      - ${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "127.0.0.1:3102:3102"
`
	out := string(normalizeComposeNetworkAliases([]byte(in), "notifications"))
	if !strings.Contains(out, "hostname: plugin-notifications") {
		t.Fatalf("expected hostname: plugin-notifications, got:\n%s", out)
	}
	if strings.Contains(out, "${DOCKER_NETWORK}:") {
		t.Fatalf("must never emit ${DOCKER_NETWORK} as a map key, got:\n%s", out)
	}
	if err := assertValidComposeYAML(out); err != nil {
		t.Fatalf("normalized output is not valid YAML: %v\n%s", err, out)
	}
}
