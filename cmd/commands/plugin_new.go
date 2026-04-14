package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var validPluginName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// pluginTemplateData holds the data passed to scaffold templates.
type pluginTemplateData struct {
	Name        string
	Description string
	Author      string
	License     string
	Year        int
	Language    string
}

var pluginNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new plugin project",
	Long: `Create a new plugin project with all required files.

Templates: go (default), rust, node, static

  nself plugin new my-plugin                    # Go plugin
  nself plugin new my-plugin --template rust    # Rust plugin
  nself plugin new my-plugin --template node    # Node.js plugin`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginNew,
}

func init() {
	pluginNewCmd.Flags().String("template", "go", "Plugin template: go, rust, node, static")
	pluginNewCmd.Flags().String("description", "", "Short description of the plugin")
	pluginNewCmd.Flags().String("author", "", "Plugin author name")
	pluginNewCmd.Flags().String("license", "MIT", "License type")
	pluginCmd.AddCommand(pluginNewCmd)
}

func runPluginNew(cmd *cobra.Command, args []string) error {
	name := args[0]
	tmplType, _ := cmd.Flags().GetString("template")
	description, _ := cmd.Flags().GetString("description")
	author, _ := cmd.Flags().GetString("author")
	licenseName, _ := cmd.Flags().GetString("license")

	if !validPluginName.MatchString(name) {
		return fmt.Errorf("invalid plugin name %q: must start with a letter, contain only lowercase letters, numbers, and hyphens", name)
	}

	if description == "" {
		description = fmt.Sprintf("nSelf plugin: %s", name)
	}

	data := pluginTemplateData{
		Name:        name,
		Description: description,
		Author:      author,
		License:     licenseName,
		Year:        time.Now().Year(),
		Language:    tmplType,
	}

	// Validate template type
	switch tmplType {
	case "go", "rust", "node", "static":
	default:
		return fmt.Errorf("unknown template %q: must be go, rust, node, or static", tmplType)
	}

	outDir := filepath.Join(".", name)
	if _, err := os.Stat(outDir); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	ui.Info(fmt.Sprintf("Scaffolding %s plugin %q...", tmplType, name))

	// Create directory structure
	dirs := []string{
		outDir,
		filepath.Join(outDir, "docs"),
		filepath.Join(outDir, "scripts"),
		filepath.Join(outDir, "tests"),
	}
	if tmplType == "go" {
		dirs = append(dirs, filepath.Join(outDir, "cmd"))
		dirs = append(dirs, filepath.Join(outDir, "internal"))
	}
	if tmplType == "rust" {
		dirs = append(dirs, filepath.Join(outDir, "src"))
	}
	if tmplType == "node" {
		dirs = append(dirs, filepath.Join(outDir, "src"))
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Generate all files
	files := pluginScaffoldFiles(data, tmplType)
	for relPath, content := range files {
		fullPath := filepath.Join(outDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("creating parent dir for %s: %w", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", relPath, err)
		}
	}

	ui.Success(fmt.Sprintf("Plugin %q scaffolded in ./%s/", name, name))
	ui.Dimmed("Next steps:")
	ui.Dimmed(fmt.Sprintf("  cd %s", name))
	switch tmplType {
	case "go":
		ui.Dimmed("  go mod tidy && go build ./...")
	case "rust":
		ui.Dimmed("  cargo build")
	case "node":
		ui.Dimmed("  pnpm install && pnpm build")
	case "static":
		ui.Dimmed("  # Edit static/ directory with your content")
	}

	return nil
}

// pluginScaffoldFiles returns a map of relative path -> file content for the scaffold.
func pluginScaffoldFiles(data pluginTemplateData, tmplType string) map[string]string {
	files := make(map[string]string)

	// plugin.yaml manifest (common to all templates)
	files["plugin.yaml"] = renderTmpl(`name: {{.Name}}
version: 0.1.0
description: {{.Description}}
category: custom
license: {{.License}}
language: {{.Language}}
author: {{.Author}}
port: 8080
health_endpoint: /healthz
dev:
  hot: true
  watch:
    - "src/**"
    - "*.go"
`, data)

	// README.md
	files["README.md"] = renderTmpl(`# {{.Name}}

{{.Description}}

## Installation

`+"```bash"+`
nself plugin install {{.Name}}
`+"```"+`

## Development

`+"```bash"+`
nself dev --hot
`+"```"+`

## License

{{.License}}
`, data)

	// LICENSE (MIT)
	files["LICENSE"] = renderTmpl(`MIT License

Copyright (c) {{.Year}} {{.Author}}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`, data)

	// CHANGELOG.md
	files["CHANGELOG.md"] = renderTmpl(`# Changelog

## 0.1.0

- Initial release of {{.Name}}
`, data)

	// scripts/dev.sh
	files["scripts/dev.sh"] = renderTmpl(`#!/usr/bin/env bash
set -euo pipefail

echo "Starting {{.Name}} in development mode..."
`, data)

	// Dockerfile (template-specific)
	switch tmplType {
	case "go":
		files["Dockerfile"] = renderTmpl(`FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /plugin ./cmd/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /plugin /usr/local/bin/plugin
EXPOSE 8080
CMD ["plugin"]
`, data)

		files["go.mod"] = renderTmpl(`module github.com/nself-org/plugins/{{.Name}}

go 1.22

`, data)

		files["go.sum"] = ""

		files["cmd/main.go"] = renderTmpl(`package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	log.Println("{{.Name}} plugin starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`, data)

	case "rust":
		files["Dockerfile"] = renderTmpl(`FROM rust:1.77-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev
COPY Cargo.toml Cargo.lock ./
COPY src/ src/
RUN cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/{{.Name}} /usr/local/bin/plugin
EXPOSE 8080
CMD ["plugin"]
`, data)

		files["Cargo.toml"] = renderTmpl(`[package]
name = "{{.Name}}"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
tokio = { version = "1", features = ["full"] }
`, data)

		files["Cargo.lock"] = ""

		files["src/main.rs"] = renderTmpl(`use actix_web::{web, App, HttpServer, HttpResponse};

async fn healthz() -> HttpResponse {
    HttpResponse::Ok().body("ok")
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    println!("{{.Name}} plugin starting on :8080");
    HttpServer::new(|| {
        App::new()
            .route("/healthz", web::get().to(healthz))
    })
    .bind("0.0.0.0:8080")?
    .run()
    .await
}
`, data)

	case "node":
		files["Dockerfile"] = renderTmpl(`FROM node:20-alpine AS builder
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
EXPOSE 8080
CMD ["node", "dist/index.js"]
`, data)

		files["package.json"] = renderTmpl(`{
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
`, data)

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

		files["src/index.ts"] = renderTmpl(`import { createServer } from "http";

const server = createServer((req, res) => {
  if (req.url === "/healthz") {
    res.writeHead(200);
    res.end("ok");
    return;
  }
  res.writeHead(404);
  res.end("not found");
});

server.listen(8080, () => {
  console.log("{{.Name}} plugin listening on :8080");
});
`, data)

	case "static":
		files["Dockerfile"] = renderTmpl(`FROM nginx:alpine
COPY static/ /usr/share/nginx/html/
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
`, data)

		files["static/index.html"] = renderTmpl(`<!DOCTYPE html>
<html>
<head><title>{{.Name}}</title></head>
<body><h1>{{.Name}} plugin</h1></body>
</html>
`, data)
	}

	// CI workflow
	files[".github/workflows/ci.yml"] = renderTmpl(`name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: docker build -t {{.Name}}:test .
      - name: Health check
        run: |
          docker run -d -p 8080:8080 --name test {{.Name}}:test
          sleep 2
          curl -f http://localhost:8080/healthz
          docker stop test
`, data)

	// Integration test stub
	files["tests/integration.test.ts"] = renderTmpl(`import { describe, it, expect } from "vitest";

describe("{{.Name}}", () => {
  it("should respond to health check", async () => {
    const res = await fetch("http://localhost:8080/healthz");
    expect(res.status).toBe(200);
  });
});
`, data)

	// docs/README.md
	files["docs/README.md"] = renderTmpl(`# {{.Name}} Documentation

## Overview

{{.Description}}

## Configuration

Add environment variables to your nSelf project's .env file.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| /healthz | GET    | Health check |
`, data)

	return files
}

// renderTmpl executes a Go template string with the given data.
func renderTmpl(tmplStr string, data pluginTemplateData) string {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return tmplStr
	}
	return buf.String()
}
