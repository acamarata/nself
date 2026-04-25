// Package k8s provides a thin wrapper around the Helm CLI for managing
// nSelf Helm chart deployments (B50).
//
// Status: PLANNED — deferred. Requires UD-12 minor release approval.
// The nSelf Helm chart lives in charts/nself/ at the repo root.
package k8s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HelmReleaseName is the default Helm release name used by nself k8s install.
const HelmReleaseName = "nself"

// ChartRepo is the Helm repository URL for the nSelf chart.
const ChartRepo = "https://charts.nself.org"

// ErrHelmNotFound is returned when the helm binary is not in PATH.
var ErrHelmNotFound = fmt.Errorf("k8s: helm binary not found; install from https://helm.sh")

// helmBinary locates the helm binary.
func helmBinary() (string, error) {
	path, err := exec.LookPath("helm")
	if err != nil {
		return "", ErrHelmNotFound
	}
	return path, nil
}

// Install runs `helm install` for the nSelf chart.
func Install(ctx context.Context, opts InstallOptions) error {
	helm, err := helmBinary()
	if err != nil {
		return err
	}
	if opts.ReleaseName == "" {
		opts.ReleaseName = HelmReleaseName
	}
	args := []string{
		"install", opts.ReleaseName, "nself/nself",
		"--set", fmt.Sprintf("domain=%s", opts.Domain),
		"--wait",
		"--timeout", "5m",
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig", opts.Kubeconfig)
	}
	if opts.LicenseKey != "" {
		args = append(args, "--set", fmt.Sprintf("license.key=%s", opts.LicenseKey))
	}
	for _, plugin := range opts.Plugins {
		args = append(args, "--set-string", fmt.Sprintf("plugins.install[%d]=%s", len(args), plugin))
	}
	cmd := exec.CommandContext(ctx, helm, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("k8s: helm install: %w", err)
	}
	return nil
}

// Upgrade runs `helm upgrade` for the nSelf chart with rolling update semantics.
func Upgrade(ctx context.Context, opts InstallOptions) error {
	helm, err := helmBinary()
	if err != nil {
		return err
	}
	if opts.ReleaseName == "" {
		opts.ReleaseName = HelmReleaseName
	}
	args := []string{
		"upgrade", opts.ReleaseName, "nself/nself",
		"--reuse-values",
		"--wait",
		"--timeout", "5m",
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig", opts.Kubeconfig)
	}
	cmd := exec.CommandContext(ctx, helm, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("k8s: helm upgrade: %w", err)
	}
	return nil
}

// Status returns the Helm release status summary.
func Status(ctx context.Context, releaseName, kubeconfig string) (string, error) {
	helm, err := helmBinary()
	if err != nil {
		return "", err
	}
	if releaseName == "" {
		releaseName = HelmReleaseName
	}
	args := []string{"status", releaseName, "--output", "json"}
	if kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	cmd := exec.CommandContext(ctx, helm, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("k8s: helm status: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RepoAdd adds the nSelf Helm repository.
func RepoAdd(ctx context.Context) error {
	helm, err := helmBinary()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, helm, "repo", "add", "nself", ChartRepo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("k8s: helm repo add: %w", err)
	}
	updateCmd := exec.CommandContext(ctx, helm, "repo", "update", "nself")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("k8s: helm repo update: %w", err)
	}
	return nil
}

// InstallOptions holds parameters for helm install/upgrade.
type InstallOptions struct {
	ReleaseName string
	Domain      string
	Kubeconfig  string
	LicenseKey  string
	Plugins     []string
}
