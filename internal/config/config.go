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
//
// Save writes a complete replacement file in the destination directory, then
// renames it over the old file. The replacement has mode 0600 before any
// credential bytes go into it, so an existing file with a permissive mode is
// replaced by a private file. If a step fails, Save removes the replacement
// and the previous configuration stays unchanged.
//
// If path is a symbolic link, Save replaces the link target. The link stays
// a link. Save syncs the replacement before the rename. It does not sync the
// directory, so the rename is not guaranteed to survive a power loss.
func Save(path string, cfg *Config) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	target, err := resolveLinks(path)
	if err != nil {
		return err
	}
	return replaceFile(target, data)
}

// maxLinks limits how many symbolic links resolveLinks follows.
const maxLinks = 40

// resolveLinks follows symbolic links at path. The final target can be absent.
func resolveLinks(path string) (string, error) {
	for range maxLinks {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return path, nil
		}
		link, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(path), link)
		}
		path = link
	}
	return "", fmt.Errorf("too many symbolic links at %s", path)
}

// tempFile is the part of *os.File that replaceFile uses.
type tempFile interface {
	Write(p []byte) (int, error)
	Sync() error
	Close() error
	Name() string
}

// createTemp and rename are replaced in tests to inject failures.
var (
	createTemp = func(dir, pattern string) (tempFile, error) { return os.CreateTemp(dir, pattern) }
	rename     = os.Rename
)

// replaceFile writes data to a private file next to path, then renames it over path.
func replaceFile(path string, data []byte) error {
	tmp, err := createTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	if err := writeAndPublish(tmp, path, data); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

func writeAndPublish(tmp tempFile, path string, data []byte) error {
	_, writeErr := tmp.Write(data)
	if writeErr == nil {
		writeErr = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return rename(tmp.Name(), path)
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
