package domain

import (
	"strings"
	"testing"
)

// TestRequiredRecords_ReturnsARecord verifies that RequiredRecords always
// includes an A record pointing to the server IP.
func TestRequiredRecords_ReturnsARecord(t *testing.T) {
	records := RequiredRecords("myapp.example.com", "1.2.3.4")
	if len(records) == 0 {
		t.Fatal("RequiredRecords returned no records")
	}

	found := false
	for _, r := range records {
		if r.Type == "A" && r.Name == "myapp.example.com" && r.Value == "1.2.3.4" {
			found = true
		}
	}
	if !found {
		t.Errorf("RequiredRecords did not include A record for domain/IP: %+v", records)
	}
}

// TestRequiredRecords_TableDriven verifies RequiredRecords for several
// domain/IP combinations.
func TestRequiredRecords_TableDriven(t *testing.T) {
	cases := []struct {
		domain string
		ip     string
	}{
		{"app.nself.dev", "10.0.0.1"},
		{"api.staging.example.com", "192.168.1.100"},
		{"custom-domain.io", "5.75.235.42"},
	}
	for _, tc := range cases {
		t.Run(tc.domain, func(t *testing.T) {
			records := RequiredRecords(tc.domain, tc.ip)
			if len(records) == 0 {
				t.Fatal("no records returned")
			}
			for _, r := range records {
				if r.Type == "" {
					t.Errorf("record has empty Type: %+v", r)
				}
				if r.Name == "" {
					t.Errorf("record has empty Name: %+v", r)
				}
				if r.Value == "" {
					t.Errorf("record has empty Value: %+v", r)
				}
			}
		})
	}
}

// TestNginxServerBlock_NoSSL verifies the nginx block for HTTP-only mode.
func TestNginxServerBlock_NoSSL(t *testing.T) {
	block := NginxServerBlock("myapp.example.com", "http://backend:8080", "/etc/nself/ssl", false)
	if !strings.Contains(block, "myapp.example.com") {
		t.Error("nginx block should contain the domain name")
	}
	if !strings.Contains(block, "server {") {
		t.Error("nginx block should contain 'server {' directive")
	}
	// Should NOT contain ssl_certificate for HTTP-only.
	if strings.Contains(block, "ssl_certificate") {
		t.Error("HTTP-only block should not contain ssl_certificate directive")
	}
}

// TestNginxServerBlock_WithSSL verifies the nginx block for HTTPS mode.
func TestNginxServerBlock_WithSSL(t *testing.T) {
	block := NginxServerBlock("myapp.example.com", "http://backend:8080", "/etc/nself/ssl", true)
	if !strings.Contains(block, "myapp.example.com") {
		t.Error("nginx block should contain the domain name")
	}
	// HTTPS block should contain SSL directives.
	if !strings.Contains(block, "ssl") {
		t.Error("HTTPS block should reference SSL in some form")
	}
}

// TestNginxServerBlock_ContainsUpstream verifies the upstream is referenced
// in the proxy_pass directive.
func TestNginxServerBlock_ContainsUpstream(t *testing.T) {
	upstream := "http://127.0.0.1:9000"
	block := NginxServerBlock("example.com", upstream, "/ssl", false)
	if !strings.Contains(block, upstream) {
		t.Errorf("nginx block should contain upstream %q", upstream)
	}
}
