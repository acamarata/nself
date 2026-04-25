package federation

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// supergraphYAMLTemplate is the rover supergraph compose config template.
// Each subgraph entry becomes one [[subgraphs]] block.
// See: https://www.apollographql.com/docs/rover/commands/supergraphs
const supergraphYAMLTemplate = `federation_version: =2.6.1
subgraphs:
{{- range .Subgraphs}}
  {{.Name}}:
    routing_url: {{.URL}}
{{- if .SchemaPath}}
    schema:
      file: {{.SchemaPath}}
{{- else}}
    schema:
      subgraph_url: {{.URL}}
{{- end}}
{{- end}}
`

// supergraphTemplateData is the data struct passed to supergraphYAMLTemplate.
type supergraphTemplateData struct {
	Subgraphs []SubgraphEntry
}

// BuildSupergraphYAML renders the rover supergraph compose configuration from
// the provided subgraph entries and returns the YAML content as a byte slice.
// The caller is responsible for writing the result to disk (typically at
// .nself/federation/supergraph.yaml).
//
// entries must be non-empty. At minimum, the Hasura subgraph must be present.
func BuildSupergraphYAML(entries []SubgraphEntry) ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one subgraph entry is required")
	}
	tmpl, err := template.New("supergraph").Parse(supergraphYAMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing supergraph template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, supergraphTemplateData{Subgraphs: entries}); err != nil {
		return nil, fmt.Errorf("rendering supergraph YAML: %w", err)
	}
	return buf.Bytes(), nil
}

// ComposeOptions controls the behaviour of RunRoverCompose.
type ComposeOptions struct {
	// SupergraphYAMLPath is the path to the rover supergraph compose config.
	SupergraphYAMLPath string
	// OutputPath is where rover writes supergraph.graphql.
	OutputPath string
	// RoverBin is the path to the rover binary. Defaults to "rover" on PATH.
	RoverBin string
}

// RunRoverCompose executes `rover supergraph compose --config <yaml> --output <out>`.
// It returns an error if rover is not found, exits non-zero, or produces no
// output file. This satisfies the acceptance criterion:
// "Plugin with invalid schema fails nself build with clear error."
func RunRoverCompose(ctx context.Context, opts ComposeOptions) error {
	roverBin := opts.RoverBin
	if roverBin == "" {
		roverBin = "rover"
	}
	if _, err := exec.LookPath(roverBin); err != nil {
		return fmt.Errorf("rover CLI not found (%q): install with `curl -sSL https://rover.apollo.dev/nix/latest | sh`: %w", roverBin, err)
	}
	if opts.SupergraphYAMLPath == "" {
		return fmt.Errorf("SupergraphYAMLPath must not be empty")
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("OutputPath must not be empty")
	}

	args := []string{
		"supergraph", "compose",
		"--config", opts.SupergraphYAMLPath,
		"--output", opts.OutputPath,
	}
	cmd := exec.CommandContext(ctx, roverBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("rover supergraph compose failed: %s", msg)
	}

	// Verify output was written.
	if _, err := os.Stat(opts.OutputPath); err != nil {
		return fmt.Errorf("rover did not write supergraph schema to %q: %w", opts.OutputPath, err)
	}
	return nil
}
