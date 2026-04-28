package commands

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// clawConfigPath returns ~/.nself/claw/config.yaml
func clawConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nself", "claw", "config.yaml")
}

// clawHistoryPath returns ~/.nself/claw/history
func clawHistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nself", "claw", "history")
}

// ClawConfig holds the CLI-to-claw-server configuration.
type ClawConfig struct {
	APIKey    string `yaml:"api_key,omitempty"`
	ServerURL string `yaml:"server_url,omitempty"`
}

// loadClawConfig reads config from disk.
func loadClawConfig() (*ClawConfig, error) {
	cfg := &ClawConfig{}
	data, err := os.ReadFile(clawConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading claw config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing claw config: %w", err)
	}
	return cfg, nil
}

// saveClawConfig writes config to disk.
func saveClawConfig(cfg *ClawConfig) error {
	path := clawConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling claw config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing claw config: %w", err)
	}
	return nil
}

// clawAPIKey returns the API key, checking env first then config file.
func clawAPIKey() string {
	if key := os.Getenv("NSELF_CLAW_API_KEY"); key != "" {
		return key
	}
	cfg, err := loadClawConfig()
	if err != nil {
		return ""
	}
	return cfg.APIKey
}

// clawServerURL returns the server URL, checking env first then config file.
// Mirrors clawAPIKey()'s env-first cascade so headless flows (CI, Docker)
// can override without writing to ~/.nself/claw/config.yaml.
func clawServerURL() string {
	if url := os.Getenv("NSELF_CLAW_SERVER"); url != "" {
		return strings.TrimRight(url, "/")
	}
	cfg, err := loadClawConfig()
	if err != nil {
		return ""
	}
	if cfg.ServerURL != "" {
		return strings.TrimRight(cfg.ServerURL, "/")
	}
	return ""
}

// clawClient returns an *http.Client with Bearer auth and the base URL.
// Returns (client, baseURL, error). Exits with code 2 if no API key.
func clawClient() (*http.Client, string, error) {
	apiKey := clawAPIKey()
	if apiKey == "" {
		return nil, "", fmt.Errorf("no API key configured. Set one with:\n  nself claw config set api-key <key>\n  or export NSELF_CLAW_API_KEY=<key>")
	}
	serverURL := clawServerURL()
	if serverURL == "" {
		return nil, "", fmt.Errorf("no server URL configured. Set one with:\n  nself claw config set server <url>")
	}
	client := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &clawAuthTransport{
			apiKey: apiKey,
			base:   http.DefaultTransport,
		},
	}
	return client, serverURL, nil
}

// clawAuthTransport injects Bearer token into requests.
type clawAuthTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *clawAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.base.RoundTrip(req)
}

// --- Commands ---

var clawConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or modify nClaw CLI configuration",
	Long: `Manage the nClaw CLI configuration stored at ~/.nself/claw/config.yaml.

Without subcommands, shows the current configuration.

Subcommands:
  config set api-key <key>    Save API key
  config set server <url>     Set server URL`,
	RunE: runClawConfigShow,
}

var clawConfigSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Supported keys:
  api-key    API key for authenticating with the claw server
  server     Server URL (e.g. https://claw.example.com)`,
	Args: cobra.ExactArgs(2),
	RunE: runClawConfigSet,
}

func init() {
	clawConfigCmd.AddCommand(clawConfigSetCmd)
}

func runClawConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := loadClawConfig()
	if err != nil {
		return err
	}

	fmt.Println("nClaw CLI Configuration")
	fmt.Printf("  Config file: %s\n", clawConfigPath())
	fmt.Println()

	// API key
	apiKey := clawAPIKey()
	if apiKey == "" {
		fmt.Println("  api_key:    (not set)")
	} else {
		source := "config file"
		if os.Getenv("NSELF_CLAW_API_KEY") != "" {
			source = "NSELF_CLAW_API_KEY env"
		}
		masked := apiKey
		if len(masked) > 8 {
			masked = masked[:4] + "..." + masked[len(masked)-4:]
		}
		fmt.Printf("  api_key:    %s (%s)\n", masked, source)
	}

	// Server URL
	if cfg.ServerURL == "" {
		fmt.Println("  server_url: (not set)")
	} else {
		fmt.Printf("  server_url: %s\n", cfg.ServerURL)
	}

	return nil
}

func runClawConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	cfg, err := loadClawConfig()
	if err != nil {
		return err
	}

	switch key {
	case "api-key":
		cfg.APIKey = value
		if err := saveClawConfig(cfg); err != nil {
			return err
		}
		fmt.Println("API key saved.")
	case "server":
		cfg.ServerURL = strings.TrimRight(value, "/")
		if err := saveClawConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("Server URL set to %s\n", cfg.ServerURL)
	default:
		return fmt.Errorf("unknown config key %q. Supported: api-key, server", key)
	}

	return nil
}
