package scaffold

// scaffold_files.go — file-set assembly for a scaffolded plugin.
//
// Purpose: assemble the common file set (including tenancy files and plugin.json) for a new plugin, used by Run in scaffold.go, split out for file size.
// Inputs: the resolved Params/Options for the plugin being scaffolded.
// Outputs: a list of files (path + rendered content) to write.
// Constraints: pure move from scaffold.go (CLI-R12 Batch F); no behaviour change. Template strings stay in scaffold_templates_infra.go / scaffold_templates_code.go per the existing decomposition.

import (
	"fmt"
)

// buildFiles returns the map of relative-path -> rendered content for a scaffold.
func buildFiles(p Params) (map[string]string, error) {
	files := map[string]string{}

	// plugin.json — canonical manifest.
	// multiApp section is always present; values depend on Tenancy choice.
	pluginJSON, err := renderPluginJSON(p)
	if err != nil {
		return nil, fmt.Errorf("scaffold: render plugin.json: %w", err)
	}
	files["plugin.json"] = pluginJSON

	// Language-specific files.
	switch p.Language {
	case "go", "":
		if err := addGoFiles(files, p); err != nil {
			return nil, err
		}
	case "rust":
		if err := addRustFiles(files, p); err != nil {
			return nil, err
		}
	case "node":
		if err := addNodeFiles(files, p); err != nil {
			return nil, err
		}
	case "static":
		if err := addStaticFiles(files, p); err != nil {
			return nil, err
		}
	}

	// Tenancy artifacts — migration stub + Hasura metadata stub.
	if err := addTenancyFiles(files, p); err != nil {
		return nil, err
	}

	// Common files present in every scaffold.
	dockerfile, err := buildDockerfile(p)
	if err != nil {
		return nil, err
	}
	files["Dockerfile"] = dockerfile
	compose, err := render(tmplCompose, p)
	if err != nil {
		return nil, err
	}
	files["docker-compose.plugin.yml"] = compose
	files[".dockerignore"] = tmplDockerignore
	airToml, err := render(tmplAirToml, p)
	if err != nil {
		return nil, err
	}
	files[".air.toml"] = airToml
	readme, err := render(tmplReadme, p)
	if err != nil {
		return nil, err
	}
	files["README.md"] = readme
	ci, err := render(tmplCI, p)
	if err != nil {
		return nil, err
	}
	files[".github/workflows/ci.yml"] = ci

	return files, nil
}

// addTenancyFiles emits migration.sql and hasura_metadata.json stubs whose
// content depends on the tenancy mode selected by the developer.
// TenancyNone produces an empty (comment-only) migration so there is always a
// predictable file for tooling to consume.
func addTenancyFiles(files map[string]string, p Params) error {
	switch p.Tenancy {
	case TenancyAppIsolation:
		migration, err := render(tmplMigrationAppIsolation, p)
		if err != nil {
			return err
		}
		files["migrations/001_init.sql"] = migration
		metadata, err := render(tmplHasuraNoFilter, p)
		if err != nil {
			return err
		}
		files["hasura/metadata.json"] = metadata
	case TenancyCloudTenant:
		migration, err := render(tmplMigrationCloudTenant, p)
		if err != nil {
			return err
		}
		files["migrations/001_init.sql"] = migration
		metadata, err := render(tmplHasuraCloudFilter, p)
		if err != nil {
			return err
		}
		files["hasura/metadata.json"] = metadata
	case TenancyBoth:
		migration, err := render(tmplMigrationBoth, p)
		if err != nil {
			return err
		}
		files["migrations/001_init.sql"] = migration
		metadata, err := render(tmplHasuraCloudFilter, p)
		if err != nil {
			return err
		}
		files["hasura/metadata.json"] = metadata
	default: // TenancyNone or empty
		migration, err := render(tmplMigrationNone, p)
		if err != nil {
			return err
		}
		files["migrations/001_init.sql"] = migration
		metadata, err := render(tmplHasuraNoFilter, p)
		if err != nil {
			return err
		}
		files["hasura/metadata.json"] = metadata
	}
	return nil
}

// renderPluginJSON renders plugin.json with multiApp fields that reflect the
// chosen tenancy mode. Returns an error if template parsing or execution fails.
func renderPluginJSON(p Params) (string, error) {
	// Determine multiApp field values from tenancy choice.
	multiAppSupported := p.Tenancy == TenancyAppIsolation || p.Tenancy == TenancyBoth
	isolationColumn := ""
	if multiAppSupported {
		isolationColumn = "source_account_id"
	}

	type jsonParams struct {
		Params
		MultiAppSupported bool
		IsolationColumn   string
	}
	jp := jsonParams{
		Params:            p,
		MultiAppSupported: multiAppSupported,
		IsolationColumn:   isolationColumn,
	}
	return renderAnyErr(tmplPluginJSON, jp)
}
