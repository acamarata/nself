// Compliant plugin fixture — all routes have recognized auth middleware.
package main

import (
	"net/http"
)

func registerRoutes(r interface{ Handle(string, ...interface{}) }) {
	// All routes wrapped with an allowlisted middleware.
	r.Handle("/api/data", RequirePluginJWT, handleData)
	r.Handle("/api/admin", RequireHasuraAdminKey, handleAdmin)
	r.Handle("/api/license", RequireLicenseKey, handleLicense)
}

func handleData(w http.ResponseWriter, r *http.Request) {}
func handleAdmin(w http.ResponseWriter, r *http.Request) {}
func handleLicense(w http.ResponseWriter, r *http.Request) {}
func RequirePluginJWT(next http.Handler) http.Handler { return next }
func RequireHasuraAdminKey(next http.Handler) http.Handler { return next }
func RequireLicenseKey(next http.Handler) http.Handler { return next }
