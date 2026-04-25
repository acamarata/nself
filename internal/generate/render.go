package generate

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// templateFuncs provides helpers used by per-language templates.
var templateFuncs = template.FuncMap{
	"isLast": func(i int, slice interface{}) bool {
		switch v := slice.(type) {
		case []IRColumn:
			return i == len(v)-1
		default:
			return false
		}
	},
}

// loadTemplate parses a named template from the embedded FS.
func loadTemplate(name string) (*template.Template, error) {
	path := "templates/" + name
	tmplBytes, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Funcs(templateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	return tmpl, nil
}

// renderToFile executes a template against data and writes the result to path.
func renderToFile(tmpl *template.Template, data interface{}, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

// RenderOptions controls what the renderer produces.
type RenderOptions struct {
	Langs    []string // subset of: typescript dart swift kotlin python
	OutDir   string
	NoHooks  bool   // TypeScript: skip hooks.ts
	Pydantic bool   // Python: Pydantic v2 (always true per spec)
}

// GenerateSummary reports counts for the diff summary line.
type GenerateSummary struct {
	Language string
	Files    []string
	Elapsed  time.Duration
}

// Render generates all requested language outputs to outDir.
// It returns one GenerateSummary per language processed.
func Render(ir *IR, opts RenderOptions) ([]GenerateSummary, error) {
	var summaries []GenerateSummary

	for _, lang := range opts.Langs {
		start := time.Now()
		var files []string
		var err error

		switch lang {
		case "typescript":
			files, err = renderTypeScript(ir, opts)
		case "dart":
			files, err = renderDart(ir, opts)
		case "swift":
			files, err = renderSwift(ir, opts)
		case "kotlin":
			files, err = renderKotlin(ir, opts)
		case "python":
			files, err = renderPython(ir, opts)
		default:
			return nil, fmt.Errorf("unsupported language: %s", lang)
		}

		if err != nil {
			return nil, fmt.Errorf("%s: %w", lang, err)
		}

		summaries = append(summaries, GenerateSummary{
			Language: lang,
			Files:    files,
			Elapsed:  time.Since(start),
		})
	}

	return summaries, nil
}

func renderTypeScript(ir *IR, opts RenderOptions) ([]string, error) {
	dir := filepath.Join(opts.OutDir, "typescript")
	var files []string

	schemaTmpl, err := loadTemplate("typescript_schema.tmpl")
	if err != nil {
		return nil, err
	}
	schemaPath := filepath.Join(dir, "schema.ts")
	if err := renderToFile(schemaTmpl, ir, schemaPath); err != nil {
		return nil, err
	}
	files = append(files, schemaPath)

	queriesTmpl, err := loadTemplate("typescript_queries.tmpl")
	if err != nil {
		return nil, err
	}
	queriesPath := filepath.Join(dir, "queries.ts")
	if err := renderToFile(queriesTmpl, ir, queriesPath); err != nil {
		return nil, err
	}
	files = append(files, queriesPath)

	if !opts.NoHooks && len(ir.Operations) > 0 {
		hooksTmpl, err := loadTemplate("typescript_hooks.tmpl")
		if err != nil {
			return nil, err
		}
		hooksPath := filepath.Join(dir, "hooks.ts")
		if err := renderToFile(hooksTmpl, ir, hooksPath); err != nil {
			return nil, err
		}
		files = append(files, hooksPath)
	}

	return files, nil
}

func renderDart(ir *IR, opts RenderOptions) ([]string, error) {
	dir := filepath.Join(opts.OutDir, "dart")
	var files []string

	modelsTmpl, err := loadTemplate("dart_models.tmpl")
	if err != nil {
		return nil, err
	}
	modelsPath := filepath.Join(dir, "models.dart")
	if err := renderToFile(modelsTmpl, ir, modelsPath); err != nil {
		return nil, err
	}
	files = append(files, modelsPath)

	if len(ir.Operations) > 0 {
		queriesTmpl, err := loadTemplate("dart_queries.tmpl")
		if err != nil {
			return nil, err
		}
		queriesPath := filepath.Join(dir, "queries.dart")
		if err := renderToFile(queriesTmpl, ir, queriesPath); err != nil {
			return nil, err
		}
		files = append(files, queriesPath)
	}

	return files, nil
}

func renderSwift(ir *IR, opts RenderOptions) ([]string, error) {
	dir := filepath.Join(opts.OutDir, "swift")
	var files []string

	modelsTmpl, err := loadTemplate("swift_models.tmpl")
	if err != nil {
		return nil, err
	}
	modelsPath := filepath.Join(dir, "NselfModels.swift")
	if err := renderToFile(modelsTmpl, ir, modelsPath); err != nil {
		return nil, err
	}
	files = append(files, modelsPath)

	return files, nil
}

func renderKotlin(ir *IR, opts RenderOptions) ([]string, error) {
	dir := filepath.Join(opts.OutDir, "kotlin")
	var files []string

	modelsTmpl, err := loadTemplate("kotlin_models.tmpl")
	if err != nil {
		return nil, err
	}
	modelsPath := filepath.Join(dir, "NselfModels.kt")
	if err := renderToFile(modelsTmpl, ir, modelsPath); err != nil {
		return nil, err
	}
	files = append(files, modelsPath)

	return files, nil
}

func renderPython(ir *IR, opts RenderOptions) ([]string, error) {
	dir := filepath.Join(opts.OutDir, "python")
	var files []string

	modelsTmpl, err := loadTemplate("python_models.tmpl")
	if err != nil {
		return nil, err
	}
	modelsPath := filepath.Join(dir, "models.py")
	if err := renderToFile(modelsTmpl, ir, modelsPath); err != nil {
		return nil, err
	}
	files = append(files, modelsPath)

	return files, nil
}
