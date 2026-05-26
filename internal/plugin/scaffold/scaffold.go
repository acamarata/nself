// Package scaffold provides the canonical plugin scaffolding logic shared
// between the nself CLI (plugin new command) and the standalone new-plugin
// binary in plugin-sdk-go/devkit.
//
// Both entry points call scaffold.Run with a Params struct; the output is
// identical so that plugin authors get the same result regardless of whether
// they have nself installed or use the SDK devkit directly.
//
// Purpose: Core types and all logic functions for plugin scaffold generation.
//          Template strings are in scaffold_templates_infra.go (infrastructure,
//          devops, metadata templates) and scaffold_templates_code.go (Go code
//          templates: main, config, server, server_test).
// Inputs:  Options struct — name, tier, language, tenancy mode, overrides.
// Outputs: Result struct — output directory path and list of emitted files.
// Constraints: Must remain import-compatible with plugin-sdk-go/devkit.
//              Template strings must live in the _templates_*.go files, not here.
// SPORT:   cli/internal/plugin/scaffold — decomposed from scaffold.go (T-E2-06).
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// SlugRE is the valid plugin name regexp.
// A slug must start with a lowercase letter, be at least 2 chars, at most 41
// chars total, contain only lowercase letters, digits, and internal hyphens,
// and must NOT end with a hyphen.
var SlugRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}[a-z0-9]$`)

// TenancyMode controls which multi-tenant column(s) the scaffold emits.
// Matches the --tenancy flag and the interactive prompt choices.
type TenancyMode string

const (
	// TenancyNone omits all tenancy columns. Use for plugins with no per-user Postgres tables.
	TenancyNone TenancyMode = "none"
	// TenancyAppIsolation emits source_account_id TEXT NOT NULL DEFAULT 'primary'.
	// Correct for multi-app isolation within one nSelf deploy.
	TenancyAppIsolation TenancyMode = "app-isolation"
	// TenancyCloudTenant emits tenant_id UUID (nullable) + Hasura row filter.
	// Correct for Cloud SaaS where each paying customer is isolated.
	TenancyCloudTenant TenancyMode = "cloud-tenant"
	// TenancyBoth emits both columns. Use when unsure — the developer can remove one later.
	TenancyBoth TenancyMode = "both"
)

// Params carries all values available inside scaffold templates.
type Params struct {
	Name        string // plugin slug, e.g. "mywidget"
	PascalName  string // e.g. "Mywidget"
	EnvPrefix   string // e.g. "MYWIDGET" (upper-cased, dashes to underscores)
	RepoBucket  string // "paid" or "free"
	Tier        string // "free" or "pro"
	Bundle      string // bundle display name, e.g. "nClaw" (empty allowed for free)
	Description string
	Author      string
	License     string
	Language    string // "go" (default), "rust", "node", "static"
	MinCLI      string
	MinSDK      string
	Category    string
	Port        int
	Year        int
	Tenancy     TenancyMode // multi-tenant column choice; empty == TenancyNone
}

// Options configures a scaffold run.
type Options struct {
	// Name is required. Must match SlugRE.
	Name string
	// Tier is "free" or "pro". Default "free".
	Tier string
	// Bundle is the bundle display name (optional for free plugins).
	Bundle string
	// Description defaults to "nSelf <name> plugin."
	Description string
	// Author is optional.
	Author string
	// Category defaults to "custom".
	Category string
	// Language is the plugin language: go, rust, node, static. Default "go".
	Language string
	// MinCLI is the minimum nSelf CLI version required. Default "1.0.9".
	MinCLI string
	// MinSDK is the minimum plugin-sdk-go version required. Default "0.1.0".
	MinSDK string
	// Port is the default listen port. Default 8080.
	Port int
	// OutDir overrides the output directory. Default: ./<name>.
	OutDir string
	// Force allows overwriting an existing directory.
	Force bool
	// Tenancy controls multi-tenant column scaffolding. Default TenancyNone.
	// When empty string it is treated as TenancyNone.
	Tenancy TenancyMode
}

// Result describes what was emitted.
type Result struct {
	Dir   string
	Files []string
}

// Run executes the scaffold and returns the result.
func Run(opts Options) (*Result, error) {
	// Apply defaults.
	if !SlugRE.MatchString(opts.Name) {
		return nil, fmt.Errorf("invalid plugin name %q: must match %s", opts.Name, SlugRE)
	}
	if opts.Tier == "" {
		opts.Tier = "free"
	}
	if opts.Tier != "free" && opts.Tier != "pro" {
		return nil, fmt.Errorf("--tier must be 'free' or 'pro', got %q", opts.Tier)
	}
	if opts.Language == "" {
		opts.Language = "go"
	}
	if opts.Category == "" {
		opts.Category = "custom"
	}
	if opts.MinCLI == "" {
		opts.MinCLI = "1.0.9"
	}
	if opts.MinSDK == "" {
		opts.MinSDK = "0.1.0"
	}
	if opts.Port == 0 {
		opts.Port = 8080
	}
	if opts.Description == "" {
		opts.Description = fmt.Sprintf("nSelf %s plugin.", opts.Name)
	}
	if opts.Tenancy == "" {
		opts.Tenancy = TenancyNone
	}
	switch opts.Tenancy {
	case TenancyNone, TenancyAppIsolation, TenancyCloudTenant, TenancyBoth:
	default:
		return nil, fmt.Errorf("--tenancy must be none, app-isolation, cloud-tenant, or both; got %q", opts.Tenancy)
	}

	repoBucket := "paid"
	if opts.Tier == "free" {
		repoBucket = "free"
	}

	outDir := opts.OutDir
	if outDir == "" {
		outDir = filepath.Join(".", opts.Name)
	}

	// Safety check.
	if !opts.Force {
		if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 {
			return nil, fmt.Errorf("destination %q is not empty (use Force to overwrite)", outDir)
		}
	}

	params := Params{
		Name:        opts.Name,
		PascalName:  toPascal(opts.Name),
		EnvPrefix:   toEnvPrefix(opts.Name),
		RepoBucket:  repoBucket,
		Tier:        opts.Tier,
		Bundle:      opts.Bundle,
		Description: opts.Description,
		Author:      opts.Author,
		License:     licenseForTier(opts.Tier),
		Language:    opts.Language,
		Category:    opts.Category,
		MinCLI:      opts.MinCLI,
		MinSDK:      opts.MinSDK,
		Port:        opts.Port,
		Year:        time.Now().Year(),
		Tenancy:     opts.Tenancy,
	}

	fileList := buildFiles(params)

	var emitted []string
	for relPath, content := range fileList {
		fullPath := filepath.Join(outDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("creating dir for %s: %w", relPath, err)
		}
		mode := os.FileMode(0644)
		if strings.HasSuffix(relPath, ".sh") {
			mode = 0750
		}
		if err := os.WriteFile(fullPath, []byte(content), mode); err != nil {
			return nil, fmt.Errorf("writing %s: %w", relPath, err)
		}
		emitted = append(emitted, relPath)
	}

	return &Result{Dir: outDir, Files: emitted}, nil
}

// buildFiles returns the map of relative-path -> rendered content for a scaffold.
func buildFiles(p Params) map[string]string {
	files := map[string]string{}

	// plugin.json — canonical manifest.
	// multiApp section is always present; values depend on Tenancy choice.
	files["plugin.json"] = renderPluginJSON(p)

	// Language-specific files.
	switch p.Language {
	case "go", "":
		addGoFiles(files, p)
	case "rust":
		addRustFiles(files, p)
	case "node":
		addNodeFiles(files, p)
	case "static":
		addStaticFiles(files, p)
	}

	// Tenancy artifacts — migration stub + Hasura metadata stub.
	addTenancyFiles(files, p)

	// Common files present in every scaffold.
	files["Dockerfile"] = buildDockerfile(p)
	files["docker-compose.plugin.yml"] = render(tmplCompose, p)
	files[".dockerignore"] = tmplDockerignore
	files[".air.toml"] = render(tmplAirToml, p)
	files["README.md"] = render(tmplReadme, p)
	files[".github/workflows/ci.yml"] = render(tmplCI, p)

	return files
}

// addTenancyFiles emits migration.sql and hasura_metadata.json stubs whose
// content depends on the tenancy mode selected by the developer.
// TenancyNone produces an empty (comment-only) migration so there is always a
// predictable file for tooling to consume.
func addTenancyFiles(files map[string]string, p Params) {
	switch p.Tenancy {
	case TenancyAppIsolation:
		files["migrations/001_init.sql"] = render(tmplMigrationAppIsolation, p)
		files["hasura/metadata.json"] = render(tmplHasuraNoFilter, p)
	case TenancyCloudTenant:
		files["migrations/001_init.sql"] = render(tmplMigrationCloudTenant, p)
		files["hasura/metadata.json"] = render(tmplHasuraCloudFilter, p)
	case TenancyBoth:
		files["migrations/001_init.sql"] = render(tmplMigrationBoth, p)
		files["hasura/metadata.json"] = render(tmplHasuraCloudFilter, p)
	default: // TenancyNone or empty
		files["migrations/001_init.sql"] = render(tmplMigrationNone, p)
		files["hasura/metadata.json"] = render(tmplHasuraNoFilter, p)
	}
}

// renderPluginJSON renders plugin.json with multiApp fields that reflect the
// chosen tenancy mode.
func renderPluginJSON(p Params) string {
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
	return renderAny(tmplPluginJSON, jp)
}

func addGoFiles(files map[string]string, p Params) {
	files["go.mod"] = render(`module github.com/nself-org/{{.RepoBucket}}/{{.Name}}

go 1.23.0

require github.com/nself-org/plugin-sdk-go v0.1.0
`, p)
	files["go.sum"] = ""
	files["cmd/main.go"] = render(tmplMain, p)
	files["internal/config/config.go"] = render(tmplConfig, p)
	files["internal/server/server.go"] = render(tmplServer, p)
	files["internal/server/server_test.go"] = render(tmplServerTest, p)
}

func addRustFiles(files map[string]string, p Params) {
	files["Cargo.toml"] = render(`[package]
name = "{{.Name}}"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
tokio = { version = "1", features = ["full"] }
`, p)
	files["Cargo.lock"] = ""
	files["src/main.rs"] = render(`use actix_web::{web, App, HttpServer, HttpResponse};

async fn healthz() -> HttpResponse {
    HttpResponse::Ok().body("ok")
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    println!("{{.Name}} plugin starting on :{{.Port}}");
    HttpServer::new(|| {
        App::new().route("/healthz", web::get().to(healthz))
    })
    .bind("0.0.0.0:{{.Port}}")?
    .run()
    .await
}
`, p)
}

func addNodeFiles(files map[string]string, p Params) {
	files["package.json"] = render(`{
  "name": "nself-{{.Name}}",
  "version": "0.1.0",
  "description": "{{.Description}}",
  "main": "dist/index.js",
  "scripts": {
    "build": "tsc",
    "dev": "tsx watch src/index.ts",
    "start": "node dist/index.js"
  },
  "license": "{{.License}}"
}
`, p)
	files["tsconfig.json"] = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src"]
}
`
	files["src/index.ts"] = render(`import { createServer } from "http";

const server = createServer((req, res) => {
  if (req.url === "/healthz") {
    res.writeHead(200);
    res.end("ok");
    return;
  }
  res.writeHead(404);
  res.end("not found");
});

server.listen({{.Port}}, () => {
  console.log("{{.Name}} plugin listening on :{{.Port}}");
});
`, p)
}

func addStaticFiles(files map[string]string, p Params) {
	files["static/index.html"] = render(`<!DOCTYPE html>
<html>
<head><title>{{.Name}}</title></head>
<body><h1>{{.Name}} plugin</h1></body>
</html>
`, p)
}

func buildDockerfile(p Params) string {
	switch p.Language {
	case "rust":
		return render(`FROM rust:1.77-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev
COPY Cargo.toml Cargo.lock ./
COPY src/ src/
RUN cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/{{.Name}} /usr/local/bin/plugin
EXPOSE {{.Port}}
CMD ["plugin"]
`, p)
	case "node":
		return render(`FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json .
EXPOSE {{.Port}}
CMD ["node", "dist/index.js"]
`, p)
	case "static":
		return render(`FROM nginx:alpine
COPY static/ /usr/share/nginx/html/
EXPOSE {{.Port}}
CMD ["nginx", "-g", "daemon off;"]
`, p)
	default: // go
		return render(`# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG VERSION=0.1.0
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /out/{{.Name}} ./cmd

FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
COPY --from=build /out/{{.Name}} /{{.Name}}
EXPOSE {{.Port}}
ENTRYPOINT ["/{{.Name}}"]
`, p)
	}
}

// render executes a Go template with the given params, returning the result.
// Panics on parse failure (template strings are literals in this package).
func render(tmpl string, p Params) string {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		// Template parse errors are programming errors, not runtime errors.
		panic(fmt.Sprintf("scaffold: template parse error: %v", err))
	}
	var buf strings.Builder
	if err := t.Execute(&buf, p); err != nil {
		// Execution errors are also programming errors for literal templates.
		panic(fmt.Sprintf("scaffold: template execute error: %v", err))
	}
	return buf.String()
}

// renderAny is like render but accepts any data value, not just Params.
// Used when the template data is a struct that embeds Params with extra fields.
func renderAny(tmpl string, data any) string {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		panic(fmt.Sprintf("scaffold: template parse error: %v", err))
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("scaffold: template execute error: %v", err))
	}
	return buf.String()
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
