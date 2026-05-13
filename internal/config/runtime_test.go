package config

import "testing"

// TestIsJSONValue verifies detection of JSON object and array values.
func TestIsJSONValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"json object", `{"type":"HS256","key":"abc"}`, true},
		{"json array", `["a","b"]`, true},
		{"plain string", "plain-value", false},
		{"empty string", "", false},
		{"number string", "12345", false},
		{"url", "https://example.com", false},
		{"whitespace before brace", `  {"key":"val"}`, true},
		{"already single-quoted json", `'{"type":"HS256","key":"abc"}'`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSONValue(tt.input)
			if got != tt.want {
				t.Errorf("isJSONValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestQuoteEnvValue verifies that JSON values get wrapped in single quotes and
// that non-JSON values pass through unchanged.
func TestQuoteEnvValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "jwt secret json object",
			input: `{"type":"HS256","key":"abc123"}`,
			want:  `'{"type":"HS256","key":"abc123"}'`,
		},
		{
			name:  "json array",
			input: `["a","b"]`,
			want:  `'["a","b"]'`,
		},
		{
			name:  "plain value unchanged",
			input: "plain-value",
			want:  "plain-value",
		},
		{
			name:  "empty string unchanged",
			input: "",
			want:  "",
		},
		{
			name:  "already quoted is idempotent",
			input: `'{"type":"HS256","key":"abc"}'`,
			want:  `'{"type":"HS256","key":"abc"}'`,
		},
		{
			name:  "url unchanged",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "boolean string unchanged",
			input: "true",
			want:  "true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteEnvValue(tt.input)
			if got != tt.want {
				t.Errorf("QuoteEnvValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestUnquoteEnvValue verifies that single-quoted wrappers are stripped and
// that non-quoted values pass through unchanged.
func TestUnquoteEnvValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unwrap quoted jwt secret",
			input: `'{"type":"HS256","key":"abc123"}'`,
			want:  `{"type":"HS256","key":"abc123"}`,
		},
		{
			name:  "unwrap quoted array",
			input: `'["a","b"]'`,
			want:  `["a","b"]`,
		},
		{
			name:  "plain value unchanged",
			input: "plain-value",
			want:  "plain-value",
		},
		{
			name:  "empty string unchanged",
			input: "",
			want:  "",
		},
		{
			name:  "single char not stripped",
			input: "'",
			want:  "'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnquoteEnvValue(tt.input)
			if got != tt.want {
				t.Errorf("UnquoteEnvValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestQuoteUnquoteRoundTrip verifies that QuoteEnvValue → UnquoteEnvValue is
// a perfect round-trip for all value shapes that matter for nSelf env files.
func TestQuoteUnquoteRoundTrip(t *testing.T) {
	values := []string{
		`{"type":"HS256","key":"abc123"}`,
		`{"type":"RS256","key":"-----BEGIN PUBLIC KEY-----\nMIIB..."}`,
		`["https://app.example.com","https://www.example.com"]`,
		`plain-value`,
		`https://example.com`,
		``,
		`true`,
		`somepassword123`,
	}
	for _, v := range values {
		quoted := QuoteEnvValue(v)
		got := UnquoteEnvValue(quoted)
		if got != v {
			t.Errorf("round-trip failed for %q: QuoteEnvValue → %q, UnquoteEnvValue → %q", v, quoted, got)
		}
	}
}
