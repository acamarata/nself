package sdk

import (
	"encoding/json"
	"net/http"
)

// Respond writes a JSON response with the given status code and body.
// If body is nil, only the status code is written (no response body).
func Respond(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// Error writes a JSON error response with the given status code and message.
func Error(w http.ResponseWriter, status int, message string) {
	Respond(w, status, map[string]string{"error": message})
}
