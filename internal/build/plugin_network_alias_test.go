package build

import (
	"strings"
	"testing"
)

// TestNormalizeComposeNetworkAliases_ShortForm verifies that the short-form
// networks list is rewritten to the long form with a plugin-{name} alias so
// inter-plugin DNS works with the CLI-injected PLUGIN_*_INTERNAL_URL env vars.
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
	if !strings.Contains(out, "aliases:") {
		t.Fatalf("expected aliases block, got:\n%s", out)
	}
	if !strings.Contains(out, "- plugin-mux") {
		t.Fatalf("expected plugin-mux alias, got:\n%s", out)
	}
	if !strings.Contains(out, "${DOCKER_NETWORK}:") {
		t.Fatalf("expected long-form network key, got:\n%s", out)
	}
	// Rest of the service block must remain intact.
	if !strings.Contains(out, "container_name: ${COMPOSE_PROJECT_NAME}_mux") {
		t.Fatalf("container_name line was clobbered, got:\n%s", out)
	}
	if !strings.Contains(out, `ports:`) || !strings.Contains(out, `127.0.0.1:3711:3711`) {
		t.Fatalf("ports block was clobbered, got:\n%s", out)
	}
}

// TestNormalizeComposeNetworkAliases_Idempotent verifies that running the
// normalizer twice does not duplicate the aliases block.
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
	if strings.Count(string(twice), "- plugin-ai") != 1 {
		t.Fatalf("alias should appear exactly once, got:\n%s", string(twice))
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
