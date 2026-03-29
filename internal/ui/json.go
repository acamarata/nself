package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON marshals v to indented JSON (2-space indent) and writes it to stdout.
func PrintJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

// PrintJSONError writes a JSON error object {"error": "message"} to stderr.
func PrintJSONError(err error) {
	msg := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	data, _ := json.Marshal(msg)
	fmt.Fprintln(os.Stderr, string(data))
}
