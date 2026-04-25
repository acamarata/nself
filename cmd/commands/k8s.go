package commands

// k8s.go — B50: Kubernetes Helm chart deployment for nSelf.
//
// Status: PLANNED — deferred (requires UD-12 minor release approval).
//
// Commands:
//   nself k8s install  --cluster <kubeconfig> --domain <domain>
//   nself k8s upgrade  [--cluster <kubeconfig>]
//   nself k8s status   [--cluster <kubeconfig>]

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/k8s"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Manage nSelf on Kubernetes via Helm",
	Long: `Deploy and manage nSelf on any Kubernetes cluster using the official Helm chart.

Requires helm to be installed: https://helm.sh

The official chart is published at https://charts.nself.org.

Examples:
  nself k8s install --domain myapp.com
  nself k8s status
  nself k8s upgrade`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var k8sInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install nSelf on a Kubernetes cluster",
	Long: `Install the nSelf Helm chart on a Kubernetes cluster.

The chart deploys Postgres (StatefulSet), Hasura, Auth, and Nginx Ingress.
TLS is managed by cert-manager (must be pre-installed in the cluster).

Example:
  nself k8s install --domain myapp.com
  nself k8s install --domain myapp.com --cluster ~/.kube/config --release my-nself`,
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, _ := cmd.Flags().GetString("domain")
		if domain == "" {
			return fmt.Errorf("--domain is required")
		}
		cluster, _ := cmd.Flags().GetString("cluster")
		release, _ := cmd.Flags().GetString("release")
		pluginsRaw, _ := cmd.Flags().GetString("plugins")
		licenseKey := os.Getenv("NSELF_PLUGIN_LICENSE_KEY")

		var plugins []string
		if pluginsRaw != "" {
			for _, p := range strings.Split(pluginsRaw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					plugins = append(plugins, p)
				}
			}
		}

		ui.Info("Adding nSelf Helm repository...")
		if err := k8s.RepoAdd(cmd.Context()); err != nil {
			return err
		}

		opts := k8s.InstallOptions{
			ReleaseName: release,
			Domain:      domain,
			Kubeconfig:  cluster,
			LicenseKey:  licenseKey,
			Plugins:     plugins,
		}
		ui.Info(fmt.Sprintf("Installing nSelf chart (domain=%s)...", domain))
		if err := k8s.Install(cmd.Context(), opts); err != nil {
			return err
		}
		ui.Success(fmt.Sprintf("nSelf installed. Access your stack at https://%s", domain))
		return nil
	},
}

var k8sUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the nSelf Helm release",
	Long:  `Upgrade the nSelf Helm chart to the latest version with a rolling update.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster, _ := cmd.Flags().GetString("cluster")
		release, _ := cmd.Flags().GetString("release")
		opts := k8s.InstallOptions{
			ReleaseName: release,
			Kubeconfig:  cluster,
		}
		ui.Info("Upgrading nSelf Helm release...")
		if err := k8s.Upgrade(cmd.Context(), opts); err != nil {
			return err
		}
		ui.Success("nSelf upgraded successfully.")
		return nil
	},
}

var k8sStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the Helm release status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster, _ := cmd.Flags().GetString("cluster")
		release, _ := cmd.Flags().GetString("release")
		out, err := k8s.Status(cmd.Context(), release, cluster)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

func init() {
	k8sInstallCmd.Flags().String("domain", "", "Domain for the nSelf deployment (required)")
	k8sInstallCmd.Flags().String("cluster", "", "Path to kubeconfig (defaults to KUBECONFIG env or ~/.kube/config)")
	k8sInstallCmd.Flags().String("release", k8s.HelmReleaseName, "Helm release name")
	k8sInstallCmd.Flags().String("plugins", "", "Comma-separated list of plugins to install")

	k8sUpgradeCmd.Flags().String("cluster", "", "Path to kubeconfig")
	k8sUpgradeCmd.Flags().String("release", k8s.HelmReleaseName, "Helm release name")

	k8sStatusCmd.Flags().String("cluster", "", "Path to kubeconfig")
	k8sStatusCmd.Flags().String("release", k8s.HelmReleaseName, "Helm release name")

	k8sCmd.AddCommand(k8sInstallCmd)
	k8sCmd.AddCommand(k8sUpgradeCmd)
	k8sCmd.AddCommand(k8sStatusCmd)

	RootCmd.AddCommand(k8sCmd)
}
