// Package infra provides a thin wrapper around the Terraform CLI for
// provisioning nSelf infrastructure via provider-specific modules (B51/B52).
//
// Status: PLANNED — deferred. Requires UD-12 minor release approval.
// Terraform modules live in terraform/modules/<provider>/ at the repo root.
package infra

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Provider is a cloud provider supported by the Terraform modules.
type Provider string

const (
	ProviderAWS          Provider = "aws"
	ProviderGCP          Provider = "gcp"
	ProviderAzure        Provider = "azure"
	ProviderHetzner      Provider = "hetzner"
	ProviderDigitalOcean Provider = "do"
	ProviderLinode       Provider = "linode"
)

// ValidProviders is the set of supported providers.
var ValidProviders = map[Provider]bool{
	ProviderAWS:          true,
	ProviderGCP:          true,
	ProviderAzure:        true,
	ProviderHetzner:      true,
	ProviderDigitalOcean: true,
	ProviderLinode:       true,
}

// ErrTerraformNotFound is returned when the terraform binary is not in PATH.
var ErrTerraformNotFound = fmt.Errorf("infra: terraform binary not found; install from https://developer.hashicorp.com/terraform")

// PlanOptions holds parameters for terraform plan.
type PlanOptions struct {
	Provider   Provider
	Domain     string
	StateBucket string
	ModulesDir string // defaults to terraform/modules/<provider>
}

// ApplyOptions holds parameters for terraform apply.
type ApplyOptions struct {
	PlanOptions
	AutoApprove bool
}

func terraformBinary() (string, error) {
	path, err := exec.LookPath("terraform")
	if err != nil {
		return "", ErrTerraformNotFound
	}
	return path, nil
}

func modulePath(opts PlanOptions) string {
	if opts.ModulesDir != "" {
		return opts.ModulesDir
	}
	return fmt.Sprintf("terraform/modules/%s", opts.Provider)
}

// Plan runs `terraform plan` for the given provider module.
func Plan(ctx context.Context, opts PlanOptions) error {
	tf, err := terraformBinary()
	if err != nil {
		return err
	}
	if !ValidProviders[opts.Provider] {
		return fmt.Errorf("infra: unknown provider %q", opts.Provider)
	}
	dir := modulePath(opts)
	if err := tfInit(ctx, tf, dir); err != nil {
		return err
	}
	args := []string{
		"plan",
		fmt.Sprintf("-var=domain=%s", opts.Domain),
	}
	if opts.StateBucket != "" {
		args = append(args,
			fmt.Sprintf("-backend-config=bucket=%s", opts.StateBucket),
		)
	}
	cmd := exec.CommandContext(ctx, tf, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("infra: terraform plan: %w", err)
	}
	return nil
}

// Apply runs `terraform apply` for the given provider module.
func Apply(ctx context.Context, opts ApplyOptions) error {
	tf, err := terraformBinary()
	if err != nil {
		return err
	}
	if !ValidProviders[opts.Provider] {
		return fmt.Errorf("infra: unknown provider %q", opts.Provider)
	}
	dir := modulePath(opts.PlanOptions)
	if err := tfInit(ctx, tf, dir); err != nil {
		return err
	}
	args := []string{
		"apply",
		fmt.Sprintf("-var=domain=%s", opts.Domain),
	}
	if opts.AutoApprove {
		args = append(args, "-auto-approve")
	}
	if opts.StateBucket != "" {
		args = append(args,
			fmt.Sprintf("-backend-config=bucket=%s", opts.StateBucket),
		)
	}
	cmd := exec.CommandContext(ctx, tf, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("infra: terraform apply: %w", err)
	}
	return nil
}

// Destroy runs `terraform destroy` for the given provider module.
func Destroy(ctx context.Context, provider Provider, domain, modulesDir string, autoApprove bool) error {
	tf, err := terraformBinary()
	if err != nil {
		return err
	}
	if !ValidProviders[provider] {
		return fmt.Errorf("infra: unknown provider %q", provider)
	}
	opts := PlanOptions{Provider: provider, Domain: domain, ModulesDir: modulesDir}
	dir := modulePath(opts)
	args := []string{"destroy", fmt.Sprintf("-var=domain=%s", domain)}
	if autoApprove {
		args = append(args, "-auto-approve")
	}
	cmd := exec.CommandContext(ctx, tf, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("infra: terraform destroy: %w", err)
	}
	return nil
}

// tfInit runs terraform init inside the module directory.
func tfInit(ctx context.Context, tf, dir string) error {
	cmd := exec.CommandContext(ctx, tf, "init", "-input=false")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("infra: terraform init: %w", err)
	}
	return nil
}
