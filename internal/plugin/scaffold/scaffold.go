// Package scaffold provides the canonical plugin scaffolding logic shared
// between the nself CLI (plugin new command) and the standalone new-plugin
// binary in plugin-sdk-go/devkit.
//
// Both entry points call scaffold.Run with a Params struct; the output is
// identical so that plugin authors get the same result regardless of whether
// they have nself installed or use the SDK devkit directly.
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
	files["plugin.json"] = render(`{
  "name": "{{.Name}}",
  "version": "0.1.0",
  "description": "{{.Description}}",
  "author": "{{.Author}}",
  "license": {{if eq .Tier "pro"}}"Source-Available"{{else}}"MIT"{{end}},
  "isCommercial": {{if eq .Tier "pro"}}true{{else}}false{{end}},
  {{- if eq .Tier "pro"}}
  "licenseType": "pro",
  "requiredEntitlements": ["pro"],
  "requires_license": true,
  {{- end}}
  "homepage": "https://nself.org/plugins",
  "minNselfVersion": "{{.MinCLI}}",
  "minSdkVersion": "{{.MinSDK}}",
  "category": "{{.Category}}",
  {{- if .Bundle}}
  "bundle": "{{.Bundle}}",
  {{- end}}
  "tags": ["{{.Name}}"]
}
`, p)

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

	// Common files present in every scaffold.
	files["Dockerfile"] = buildDockerfile(p)
	files["docker-compose.plugin.yml"] = render(tmplCompose, p)
	files[".dockerignore"] = tmplDockerignore
	files[".air.toml"] = render(tmplAirToml, p)
	files["README.md"] = render(tmplReadme, p)
	files[".github/workflows/ci.yml"] = render(tmplCI, p)

	return files
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

// --- template strings ---

const tmplCompose = `# docker-compose.plugin.yml for {{.Name}}
# Merged into the generated stack by ` + "`nself build`" + `. Do not hand-edit.
services:
  {{.Name}}:
    image: nself/{{.Name}}:${{"{"}}{{.EnvPrefix}}_VERSION:-latest}
    container_name: ${PROJECT_NAME:-nself}_{{.Name}}
    restart: unless-stopped
    environment:
      LOG_LEVEL: ${LOG_LEVEL:-info}
      DATABASE_URL: ${DATABASE_URL}
      {{.EnvPrefix}}_LISTEN_ADDR: ":{{.Port}}"
    ports:
      - "127.0.0.1:{{.Port}}:{{.Port}}"
    networks:
      - nself_net
networks:
  nself_net:
    external: true
`

const tmplDockerignore = `.git
.gitignore
README.md
Dockerfile
docker-compose*.yml
.air.toml
tmp/
*.test
coverage.out
`

const tmplAirToml = `# air.toml — hot-reload for {{.Name}} dev (pair with nself plugin dev)
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/{{.Name}} ./cmd"
  bin = "tmp/{{.Name}}"
  delay = 500
  include_ext = ["go", "yaml", "yml"]
  exclude_dir = ["tmp", "vendor", ".git"]

[log]
  time = true

[color]
  app = "magenta"
`

const tmplReadme = `# {{.PascalName}} Plugin

{{.Description}}

Tier: ` + "`{{.Tier}}`" + `{{if .Bundle}}  ·  Bundle: ` + "`{{.Bundle}}`" + `{{end}}  ·  Category: ` + "`{{.Category}}`" + `

## Local development

` + "```bash" + `
go mod tidy
go test ./...
go run ./cmd        # runs on :{{.Port}}
` + "```" + `

With hot-reload (install [air](https://github.com/air-verse/air)):

` + "```bash" + `
nself plugin dev {{.Name}}
` + "```" + `

## Endpoints

- ` + "`GET /healthz`" + ` — liveness
- ` + "`GET /readyz`" + ` — readiness
- ` + "`GET /metrics`" + ` — Prometheus metrics
- ` + "`GET /version`" + ` — plugin version
- ` + "`GET /v1/hello`" + ` — starter handler

## License

{{if eq .Tier "pro"}}Source-Available (pro tier). Requires an active nSelf license key.{{else}}MIT.{{end}}
`

const tmplCI = `name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - name: Test
        run: go test ./...
      - name: Build Docker image
        run: docker build -t {{.Name}}:test .
      - name: Health check
        run: |
          docker run -d -p {{.Port}}:{{.Port}} --name test {{.Name}}:test
          sleep 2
          curl -f http://localhost:{{.Port}}/healthz
          docker stop test
`

const tmplMain = `// Package main is the entrypoint for the {{.Name}} plugin.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/config"
	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/server"

	"github.com/nself-org/plugin-sdk-go/logger"
)

// Version is stamped at build time via -ldflags.
var Version = "0.1.0"

func main() {
	cfg := config.FromEnv()
	log := logger.New(logger.Options{
		Plugin:  "{{.Name}}",
		Version: Version,
		Level:   logger.ParseLevel(cfg.LogLevel),
	})

	srv := server.New(server.Deps{Config: cfg, Logger: log, Version: Version})

	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("{{.Name}} listening", "addr", cfg.ListenAddr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	log.Info("{{.Name}} stopped cleanly")
}
`

const tmplConfig = `// Package config loads {{.Name}} config from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds runtime config.
type Config struct {
	ListenAddr  string
	LogLevel    string
	DatabaseURL string
}

// FromEnv reads config from env vars with sensible defaults.
func FromEnv() Config {
	return Config{
		ListenAddr:  envOr("{{.EnvPrefix}}_LISTEN_ADDR", ":{{.Port}}"),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

// Validate returns an error if required fields are missing.
func (c Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("{{.Name}}: listen address must not be empty")
	}
	return nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
`

const tmplServer = `// Package server wires the HTTP router for the {{.Name}} plugin.
package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/config"

	sdkmetrics "github.com/nself-org/plugin-sdk-go/metrics"
	sdkserver "github.com/nself-org/plugin-sdk-go/server"
)

// Deps wires runtime dependencies.
type Deps struct {
	Config  config.Config
	Logger  *slog.Logger
	Version string
}

type readyFn func(ctx context.Context) error

func (f readyFn) Ready(ctx context.Context) error { return f(ctx) }

// New returns a ready-to-serve http.Handler.
func New(d Deps) http.Handler {
	return sdkserver.New(sdkserver.Options{
		Plugin:  "{{.Name}}",
		Version: d.Version,
		Ready: readyFn(func(ctx context.Context) error {
			return d.Config.Validate()
		}),
		Routes: func(r chi.Router, m *sdkmetrics.Registry) {
			r.Route("/v1", func(r chi.Router) {
				r.With(m.Middleware("/v1/hello")).Get("/hello", helloHandler(d.Logger))
			})
		},
	})
}

func helloHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if log != nil {
			log.Info("hello called", "method", r.Method, "path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(` + "`" + `{"plugin":"{{.Name}}","hello":"world"}` + "`" + `))
	}
}
`

const tmplServerTest = `package server

import (
	"net/http/httptest"
	"testing"

	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/config"
)

func TestHelloEndpoint(t *testing.T) {
	h := New(Deps{Config: config.Config{ListenAddr: ":{{.Port}}"}, Version: "test"})
	req := httptest.NewRequest("GET", "/v1/hello", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHealthz(t *testing.T) {
	h := New(Deps{Config: config.Config{ListenAddr: ":{{.Port}}"}, Version: "test"})
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 from /healthz, got %d", rr.Code)
	}
}
`
