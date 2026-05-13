// Package scaffold provides CS_N custom service scaffolding for nself projects.
// It handles slot assignment, template rendering, and .env.dev updates.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// supportedLangs is the ordered list of language templates available for
// custom service scaffolding.
var supportedLangs = []string{"go", "node", "python", "static", "rust", "other"}

// IsValidLang reports whether lang is a supported template language.
func IsValidLang(lang string) bool {
	for _, l := range supportedLangs {
		if l == lang {
			return true
		}
	}
	return false
}

// SupportedLangs returns the list of supported language template names.
func SupportedLangs() []string {
	cp := make([]string, len(supportedLangs))
	copy(cp, supportedLangs)
	return cp
}

// Options configures a custom service scaffold run.
type Options struct {
	// Name is the service name (e.g. "my-api"). Validated against customServiceNameRe.
	Name string
	// Lang is one of: go, node, python, rust, other. Default "go".
	Lang string
	// ProjectDir is the nSelf project root. Defaults to the current working directory.
	ProjectDir string
	// Force allows overwriting an existing service directory.
	Force bool
	// DryRun prints what would be done without writing any files.
	DryRun bool
}

// Result describes what was emitted.
type Result struct {
	// Slot is the assigned CS_N slot number (1-10).
	Slot int
	// EnvKey is the env var name that was written (e.g. "CS_3").
	EnvKey string
	// EnvValue is the value that was written (e.g. "my-api:go:8003").
	EnvValue string
	// ServiceDir is the directory that was created.
	ServiceDir string
	// EnvFile is the .env file that was updated.
	EnvFile string
	// Files lists the relative (to ServiceDir) paths of emitted files.
	Files []string
}

// Run scaffolds a custom service into the nSelf project at opts.ProjectDir.
// It finds the next free CS_N slot (1-10), creates the service directory,
// and writes CS_N=<name>:<lang>:<port> into .env.dev.
func Run(opts Options) (*Result, error) {
	if opts.Lang == "" {
		opts.Lang = "go"
	}
	if !IsValidLang(opts.Lang) {
		return nil, fmt.Errorf("unsupported language %q; choose one of: %s",
			opts.Lang, strings.Join(SupportedLangs(), ", "))
	}

	projectDir := opts.ProjectDir
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		projectDir = cwd
	}

	// Find the next free CS_N slot.
	slot, err := nextFreeSlot(projectDir)
	if err != nil {
		return nil, err
	}

	// Determine the port: 8000 + N auto-assignment (matching config.ParseCustomServices).
	port := 8000 + slot

	envKey := fmt.Sprintf("CS_%d", slot)
	// Format: name:template[:port] — matches config.ParseCustomServices expectations.
	envValue := fmt.Sprintf("%s:%s:%d", opts.Name, opts.Lang, port)

	envFile := filepath.Join(projectDir, ".env.dev")

	serviceDir := filepath.Join(projectDir, "services", opts.Name)

	if opts.DryRun {
		// Report what would happen without writing.
		return &Result{
			Slot:       slot,
			EnvKey:     envKey,
			EnvValue:   envValue,
			ServiceDir: serviceDir,
			EnvFile:    envFile,
			Files:      scaffoldFileList(opts.Name, opts.Lang),
		}, nil
	}

	// Create service directory.
	if !opts.Force {
		if entries, readErr := os.ReadDir(serviceDir); readErr == nil && len(entries) > 0 {
			return nil, fmt.Errorf("service directory %s already exists and is not empty (use --force to overwrite)", serviceDir)
		}
	}
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return nil, fmt.Errorf("creating service directory: %w", err)
	}

	// Write scaffold files.
	files, err := writeScaffold(serviceDir, opts.Name, opts.Lang, port)
	if err != nil {
		return nil, fmt.Errorf("writing scaffold files: %w", err)
	}

	// Update .env.dev.
	if err := setEnvKey(envFile, envKey, envValue); err != nil {
		return nil, fmt.Errorf("updating %s: %w", envFile, err)
	}
	// Also write port env var: <UPPER_NAME>_PORT=<port>
	portKey := fmt.Sprintf("%s_PORT", strings.ToUpper(strings.ReplaceAll(opts.Name, "-", "_")))
	if err := setEnvKey(envFile, portKey, fmt.Sprintf("%d", port)); err != nil {
		return nil, fmt.Errorf("updating %s for port: %w", envFile, err)
	}

	return &Result{
		Slot:       slot,
		EnvKey:     envKey,
		EnvValue:   envValue,
		ServiceDir: serviceDir,
		EnvFile:    envFile,
		Files:      files,
	}, nil
}

// NextCSSlot returns the next CS_N slot number (1-10) not already set in .env.dev.
// Returns an error if all 10 slots are in use.
// This is the exported variant of the internal nextFreeSlot helper (G-006).
func NextCSSlot(projectRoot string) (int, error) {
	return nextFreeSlot(projectRoot)
}

// nextFreeSlot returns the next CS_N slot (1-10) not already set in .env.dev.
// Returns an error if all 10 slots are in use.
func nextFreeSlot(projectDir string) (int, error) {
	envFile := filepath.Join(projectDir, ".env.dev")
	values := readEnvFile(envFile)

	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("CS_%d", i)
		if _, exists := values[key]; !exists || values[key] == "" {
			return i, nil
		}
	}
	return 0, fmt.Errorf("all 10 CS_N slots are in use; remove an existing custom service to add a new one")
}

// readEnvFile reads KEY=VALUE pairs from an env file.
// Returns an empty map if the file does not exist.
func readEnvFile(filename string) map[string]string {
	out := make(map[string]string)
	data, err := os.ReadFile(filename)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		out[line[:idx]] = line[idx+1:]
	}
	return out
}

// setEnvKey reads filename, replaces or appends KEY=value, and writes it back.
// Safe if the file does not exist yet.
func setEnvKey(filename, key, value string) error {
	var lines []string
	if data, err := os.ReadFile(filename); err == nil {
		lines = strings.Split(string(data), "\n")
		// Remove trailing empty element from Split on trailing newline.
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	}

	prefix := key + "="
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, key+"="+value)
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}
	return nil
}

// scaffoldFileList returns the list of files that would be scaffolded.
func scaffoldFileList(name, lang string) []string {
	base := []string{"README.md", "Dockerfile", ".dockerignore"}
	switch lang {
	case "go":
		return append(base, "go.mod", "main.go")
	case "node":
		return append(base, "package.json", "src/index.ts", "tsconfig.json")
	case "python":
		return append(base, "requirements.txt", "main.py")
	case "static":
		return append(base, "index.html", "nginx.conf")
	case "rust":
		return append(base, "Cargo.toml", "src/main.rs")
	default:
		return append(base, "main.sh")
	}
}

// writeScaffold creates the starter files for the custom service.
func writeScaffold(dir, name, lang string, port int) ([]string, error) {
	type fileSpec struct {
		rel     string
		content string
		mode    os.FileMode
	}

	upperName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	var specs []fileSpec

	readme := fmt.Sprintf("# %s\n\nCustom nSelf service scaffolded via `nself service add`.\n\n## Start\n\n```bash\nnself start\n```\n\n## Port\n\n%d (auto-assigned CS_%s)\n", name, port, upperName)
	specs = append(specs, fileSpec{"README.md", readme, 0644})
	specs = append(specs, fileSpec{".dockerignore", ".git\ntmp/\n*.test\n", 0644})

	switch lang {
	case "go":
		specs = append(specs,
			fileSpec{"go.mod", fmt.Sprintf("module %s\n\ngo 1.23.0\n", name), 0644},
			fileSpec{"main.go", fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("%s_PORT")
	if port == "" {
		port = "%d"
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	fmt.Printf("%s listening on :%s\n", "%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, upperName, port, "%s", name), 0644},
			fileSpec{"Dockerfile", fmt.Sprintf(`FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/%s .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/%s /%s
EXPOSE %d
ENTRYPOINT ["/%s"]
`, name, name, name, port, name), 0644},
		)
	case "node":
		specs = append(specs,
			fileSpec{"package.json", fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "scripts": { "start": "tsx src/index.ts", "dev": "tsx watch src/index.ts" },
  "license": "MIT"
}
`, name), 0644},
			fileSpec{"tsconfig.json", `{"compilerOptions":{"target":"ES2022","module":"NodeNext","moduleResolution":"NodeNext","strict":true}}`, 0644},
			fileSpec{"src/index.ts", fmt.Sprintf(`import { createServer } from "http";
const port = process.env.%s_PORT ?? "%d";
createServer((req, res) => {
  if (req.url === "/healthz") { res.writeHead(200); res.end("ok"); return; }
  res.writeHead(404); res.end("not found");
}).listen(port, () => console.log("%s listening on :" + port));
`, upperName, port, name), 0644},
			fileSpec{"Dockerfile", fmt.Sprintf(`FROM node:20-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
EXPOSE %d
CMD ["node", "-e", "require('./src/index.ts')"]
`, port), 0644},
		)
	case "python":
		specs = append(specs,
			fileSpec{"requirements.txt", "fastapi>=0.110\nuvicorn>=0.29\n", 0644},
			fileSpec{"main.py", fmt.Sprintf(`import os
from fastapi import FastAPI
import uvicorn

app = FastAPI()

@app.get("/healthz")
def healthz():
    return {"status": "ok"}

if __name__ == "__main__":
    port = int(os.environ.get("%s_PORT", %d))
    uvicorn.run(app, host="0.0.0.0", port=port)
`, upperName, port), 0644},
			fileSpec{"Dockerfile", fmt.Sprintf(`FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE %d
CMD ["python", "main.py"]
`, port), 0644},
		)
	case "static":
		specs = append(specs,
			fileSpec{"index.html", fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>%s</title></head>
<body><h1>%s</h1><p>Served by nSelf custom service on port %d.</p></body>
</html>
`, name, name, port), 0644},
			fileSpec{"nginx.conf", fmt.Sprintf(`# nSelf custom service — static nginx snippet
# Include this from your nginx conf.d/ directory.
server {
    listen %d;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;
    location / { try_files $uri $uri/ =404; }
}
`, port), 0644},
			fileSpec{"Dockerfile", fmt.Sprintf(`FROM nginx:alpine
COPY index.html /usr/share/nginx/html/index.html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE %d
CMD ["nginx", "-g", "daemon off;"]
`, port), 0644},
		)
	case "rust":
		specs = append(specs,
			fileSpec{"Cargo.toml", fmt.Sprintf(`[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
tokio = { version = "1", features = ["full"] }
`, name), 0644},
			fileSpec{"src/main.rs", fmt.Sprintf(`use actix_web::{web, App, HttpServer, HttpResponse};

async fn healthz() -> HttpResponse { HttpResponse::Ok().body("ok") }

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let port = std::env::var("%s_PORT").unwrap_or_else(|_| "%d".to_string());
    println!("%s listening on :{}", port);
    HttpServer::new(|| App::new().route("/healthz", web::get().to(healthz)))
        .bind(format!("0.0.0.0:{}", port))?
        .run().await
}
`, upperName, port, name), 0644},
			fileSpec{"Dockerfile", fmt.Sprintf(`FROM rust:1.77-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev
COPY Cargo.toml ./
COPY src/ src/
RUN cargo build --release
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/%s /usr/local/bin/service
EXPOSE %d
CMD ["service"]
`, name, port), 0644},
		)
	default: // "other"
		specs = append(specs,
			fileSpec{"main.sh", fmt.Sprintf(`#!/bin/sh
PORT="${%s_PORT:-%d}"
echo "%s listening on $PORT"
# Replace with your service entrypoint.
exec nc -l -p "$PORT"
`, upperName, port, name), 0750},
			fileSpec{"Dockerfile", fmt.Sprintf(`FROM alpine:3.19
COPY main.sh /main.sh
RUN chmod +x /main.sh
EXPOSE %d
CMD ["/main.sh"]
`, port), 0644},
		)
	}

	var written []string
	for _, spec := range specs {
		fullPath := filepath.Join(dir, spec.rel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", spec.rel, err)
		}
		if err := os.WriteFile(fullPath, []byte(spec.content), spec.mode); err != nil {
			return nil, fmt.Errorf("writing %s: %w", spec.rel, err)
		}
		written = append(written, spec.rel)
	}
	return written, nil
}
