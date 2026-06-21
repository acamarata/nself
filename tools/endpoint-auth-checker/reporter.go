// Package main — reporter.go
// Formats CheckResult violations for different output modes.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AnnotationFormat controls the output format for check results (github, text, or json).
type AnnotationFormat string

const (
	FormatGitHub AnnotationFormat = "github"
	FormatText   AnnotationFormat = "text"
	FormatJSON   AnnotationFormat = "json"
)

// Report prints all failures to stdout in the requested format and returns the exit code.
// Exit 1 if failures exist and failOnUnknown is true; 0 otherwise.
func Report(failures []CheckResult, format AnnotationFormat, failOnUnknown bool) int {
	if len(failures) == 0 {
		if format != FormatJSON {
			fmt.Println("endpoint-auth-gate: OK — all routes have recognized auth middleware.")
		} else {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"status":     "ok",
				"violations": []interface{}{},
			})
		}
		return 0
	}

	switch format {
	case FormatGitHub:
		for _, cr := range failures {
			mws := strings.Join(cr.UnknownMWs, ", ")
			if mws == "" {
				mws = "(none)"
			}
			// GHA workflow command: ::error file=<path>,line=<n>::<message>
			fmt.Printf("::error file=%s,line=%d::endpoint-auth-gate: route %s %q has no recognized auth middleware. Middlewares found: [%s]. Add an allowlisted middleware or register this path in ExemptRoutes.\n",
				cr.Route.File, cr.Route.Line, cr.Route.Method, cr.Route.Path, mws)
		}
	case FormatJSON:
		type jsonViolation struct {
			File        string   `json:"file"`
			Line        int      `json:"line"`
			Method      string   `json:"method"`
			Path        string   `json:"path"`
			Middlewares []string `json:"middlewares_found"`
		}
		var vs []jsonViolation
		for _, cr := range failures {
			vs = append(vs, jsonViolation{
				File:        cr.Route.File,
				Line:        cr.Route.Line,
				Method:      cr.Route.Method,
				Path:        cr.Route.Path,
				Middlewares: cr.UnknownMWs,
			})
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "fail",
			"violations": vs,
		})
	default: // text
		fmt.Printf("endpoint-auth-gate: %d violation(s) found:\n\n", len(failures))
		for _, cr := range failures {
			mws := strings.Join(cr.UnknownMWs, ", ")
			if mws == "" {
				mws = "(none)"
			}
			fmt.Printf("  %s:%d  %s %q  middlewares=[%s]\n",
				cr.Route.File, cr.Route.Line, cr.Route.Method, cr.Route.Path, mws)
		}
		fmt.Println("\nFix: wrap the handler with an allowlisted middleware, or add the route to ExemptRoutes with a documented reason.")
	}

	if failOnUnknown {
		return 1
	}
	return 0
}
