// Exempt-health fixture — /health has no auth but is in ExemptRoutes.
package main

import (
	"net/http"
)

func registerRoutes(r interface{ Handle(string, ...interface{}) }) {
	// /health is in ExemptRoutes — should NOT trigger a violation.
	r.Handle("/health", handleHealth)
	// Protected route — compliant.
	r.Handle("/api/data", RequirePluginJWT, handleData)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {}
func handleData(w http.ResponseWriter, r *http.Request)   {}

// RequirePluginJWT is a stub middleware that satisfies the plugin JWT auth contract.
func RequirePluginJWT(next http.Handler) http.Handler { return next }
