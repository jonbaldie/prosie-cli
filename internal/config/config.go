package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultApiURL is the fallback base URL for the Prosie API.
const DefaultApiURL = "https://prosie.app"

// Config stores user credentials and CLI preferences.
type Config struct {
	ApiURL string   `json:"api_url,omitempty"`
	Token  string   `json:"token,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

// DefaultConfigDir returns the default directory path for configuration files.
func DefaultConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "prosie"), nil
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "prosie"), nil
	}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "prosie"), nil
}

// DefaultConfigPath returns the default file path for config.json.
func DefaultConfigPath() (string, error) {
	if custom := os.Getenv("PROSIE_CONFIG_PATH"); custom != "" {
		return custom, nil
	}
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load loads the configuration from disk. If the file does not exist,
// an empty configuration is returned without error.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("error reading config.json at %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error reading config.json at %s: %w", path, err)
	}

	return &cfg, nil
}

// Save writes the configuration to disk with restricted permissions (0600).
func Save(path string, cfg *Config) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(path, data, 0600)
}

// Clear removes the configuration file from disk.
func Clear(path string) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ResolveApiURL returns the active API base URL, checking PROSIE_API_URL first,
// then the configuration file, and finally falling back to DefaultApiURL.
func ResolveApiURL(cfg *Config) string {
	if env := os.Getenv("PROSIE_API_URL"); env != "" {
		return strings.TrimRight(env, "/")
	}
	if cfg != nil && cfg.ApiURL != "" {
		return strings.TrimRight(cfg.ApiURL, "/")
	}
	return DefaultApiURL
}

// ResolveToken returns the active API token and its source ("environment", "config", or "none").
func ResolveToken(cfg *Config) (token string, source string) {
	if env := os.Getenv("PROSIE_API_TOKEN"); env != "" {
		return env, "environment"
	}
	if cfg != nil && cfg.Token != "" {
		return cfg.Token, "config"
	}
	return "", "none"
}
