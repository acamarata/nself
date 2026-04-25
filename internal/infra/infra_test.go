package infra

import (
	"errors"
	"testing"
)

// TestErrTerraformNotFound verifies the sentinel error is non-nil and descriptive.
func TestErrTerraformNotFound(t *testing.T) {
	if ErrTerraformNotFound == nil {
		t.Fatal("ErrTerraformNotFound is nil")
	}
	msg := ErrTerraformNotFound.Error()
	if msg == "" {
		t.Error("ErrTerraformNotFound.Error() returned empty string")
	}
	if !contains(msg, "terraform") {
		t.Errorf("ErrTerraformNotFound.Error() = %q: should mention terraform", msg)
	}
}

// TestTerraformBinaryNotInPath verifies terraformBinary returns ErrTerraformNotFound
// when terraform is absent from PATH.
func TestTerraformBinaryNotInPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := terraformBinary()
	if err == nil {
		t.Fatal("terraformBinary: expected error when terraform not on PATH")
	}
	if !errors.Is(err, ErrTerraformNotFound) {
		t.Errorf("terraformBinary: got %v, want ErrTerraformNotFound", err)
	}
}

// TestValidProviders verifies all expected cloud providers are registered.
func TestValidProviders(t *testing.T) {
	expected := []Provider{
		ProviderAWS, ProviderGCP, ProviderAzure,
		ProviderHetzner, ProviderDigitalOcean, ProviderLinode,
	}
	for _, p := range expected {
		if !ValidProviders[p] {
			t.Errorf("ValidProviders[%q] = false, want true", p)
		}
	}
}

// TestProviderConstants verifies the provider string values.
func TestProviderConstants(t *testing.T) {
	cases := []struct {
		p    Provider
		want string
	}{
		{ProviderAWS, "aws"},
		{ProviderGCP, "gcp"},
		{ProviderHetzner, "hetzner"},
		{ProviderDigitalOcean, "do"},
		{ProviderLinode, "linode"},
	}
	for _, tc := range cases {
		if string(tc.p) != tc.want {
			t.Errorf("Provider constant %q = %q, want %q", tc.p, tc.p, tc.want)
		}
	}
}

// TestModulePath verifies modulePath uses the override dir when set.
func TestModulePath(t *testing.T) {
	opts := PlanOptions{Provider: ProviderHetzner, ModulesDir: "/custom/path"}
	got := modulePath(opts)
	if got != "/custom/path" {
		t.Errorf("modulePath with ModulesDir override: got %q, want %q", got, "/custom/path")
	}
}

// TestModulePathDefault verifies modulePath derives path from provider name.
func TestModulePathDefault(t *testing.T) {
	opts := PlanOptions{Provider: ProviderHetzner}
	got := modulePath(opts)
	want := "terraform/modules/hetzner"
	if got != want {
		t.Errorf("modulePath default: got %q, want %q", got, want)
	}
}

// TestPlanErrorsWithoutTerraform verifies Plan propagates ErrTerraformNotFound.
func TestPlanErrorsWithoutTerraform(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := Plan(t.Context(), PlanOptions{Provider: ProviderHetzner, Domain: "test.local"})
	if err == nil {
		t.Fatal("Plan: expected error when terraform not on PATH")
	}
	if !errors.Is(err, ErrTerraformNotFound) {
		t.Errorf("Plan: got %v, want ErrTerraformNotFound", err)
	}
}

// TestPlanErrorsOnUnknownProvider verifies Plan rejects unknown provider strings
// even when terraform is present. We test this without terraform on PATH first —
// ErrTerraformNotFound fires before the provider check. So test the provider
// check indirectly via ValidProviders.
func TestValidProvidersRejectsUnknown(t *testing.T) {
	unknown := Provider("unknown-cloud")
	if ValidProviders[unknown] {
		t.Errorf("ValidProviders[%q] = true, want false", unknown)
	}
}

// contains is a tiny helper to avoid importing strings in test files.
func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
