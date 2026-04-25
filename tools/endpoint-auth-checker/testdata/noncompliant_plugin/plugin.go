// Noncompliant plugin fixture — one route is missing auth middleware.
package main

import (
	"net/http"
)

func registerRoutes(r interface{ Handle(string, ...interface{}) }) {
	// This route is properly protected.
	r.Handle("/api/safe", RequirePluginJWT, handleSafe)
	// This route has NO auth middleware — should trigger a violation.
	r.Handle("/api/leak", handleLeak)
}

func handleSafe(w http.ResponseWriter, r *http.Request) {}
func handleLeak(w http.ResponseWriter, r *http.Request) {}
func RequirePluginJWT(next http.Handler) http.Handler { return next }
