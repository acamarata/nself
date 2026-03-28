package config

import (
	"testing"
)

func TestRouteToFQDN(t *testing.T) {
	tests := []struct {
		name       string
		route      string
		baseDomain string
		want       string
		wantErr    bool
	}{
		{name: "empty route", route: "", baseDomain: "example.com", wantErr: true},
		{name: "empty domain", route: "api", baseDomain: "", wantErr: true},
		{name: "both empty", route: "", baseDomain: "", wantErr: true},
		{name: "slashes-only route", route: "///", baseDomain: "example.com", wantErr: true},
		{name: "whitespace-only domain", route: "api", baseDomain: "   ", wantErr: true},
		{name: "valid inputs", route: "api", baseDomain: "example.com", want: "api.example.com"},
		{name: "trailing dot on domain trimmed", route: "auth", baseDomain: "example.com.", want: "auth.example.com"},
		{name: "leading slash on route trimmed", route: "/api", baseDomain: "example.com", want: "api.example.com"},
		{name: "uppercase lowercased", route: "AUTH", baseDomain: "EXAMPLE.COM", want: "auth.example.com"},
		{name: "whitespace trimmed from both", route: "  api  ", baseDomain: "  example.com  ", want: "api.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RouteToFQDN(tt.route, tt.baseDomain)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RouteToFQDN(%q, %q) expected error, got %q", tt.route, tt.baseDomain, got)
				}
				return
			}
			if err != nil {
				t.Errorf("RouteToFQDN(%q, %q) unexpected error: %v", tt.route, tt.baseDomain, err)
				return
			}
			if got != tt.want {
				t.Errorf("RouteToFQDN(%q, %q) = %q, want %q", tt.route, tt.baseDomain, got, tt.want)
			}
		})
	}
}
