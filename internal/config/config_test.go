package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	// Custom PROSIE_CONFIG_PATH
	t.Run("PROSIE_CONFIG_PATH takes precedence", func(t *testing.T) {
		customPath := "/tmp/custom-prosie/config.json"
		t.Setenv("PROSIE_CONFIG_PATH", customPath)

		got, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != customPath {
			t.Fatalf("expected %s, got %s", customPath, got)
		}
	})

	t.Run("XDG_CONFIG_HOME is respected", func(t *testing.T) {
		t.Setenv("PROSIE_CONFIG_PATH", "")
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")

		got, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join("/custom/xdg", "prosie", "config.json")
		if got != expected {
			t.Fatalf("expected %s, got %s", expected, got)
		}
	})
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "prosie", "config.json")

	// Load non-existent file returns empty config without error
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected nil error for missing config, got %v", err)
	}
	if cfg.Token != "" || cfg.ApiURL != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}

	// Save config
	testCfg := &Config{
		ApiURL: "https://example.com/api",
		Token:  "test-secret-token",
		Scopes: []string{"read", "write"},
	}
	if err := Save(configPath, testCfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file permissions (0600)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Fatalf("expected 0600 permissions, got %04o", perm)
	}

	// Load saved config
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if loaded.ApiURL != testCfg.ApiURL || loaded.Token != testCfg.Token {
		t.Fatalf("loaded config does not match: expected %+v, got %+v", testCfg, loaded)
	}
	if !reflect.DeepEqual(loaded.Scopes, testCfg.Scopes) {
		t.Fatalf("scopes mismatch: expected %v, got %v", testCfg.Scopes, loaded.Scopes)
	}

	// Clear config
	if err := Clear(configPath); err != nil {
		t.Fatalf("failed to clear config: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, stat err: %v", err)
	}

	// Clearing already removed file should not error
	if err := Clear(configPath); err != nil {
		t.Fatalf("unexpected error clearing non-existent file: %v", err)
	}
}

func TestResolveApiURL(t *testing.T) {
	t.Run("PROSIE_API_URL takes precedence", func(t *testing.T) {
		t.Setenv("PROSIE_API_URL", "https://staging.prosie.app/")
		cfg := &Config{ApiURL: "https://local.prosie.app"}
		got := ResolveApiURL(cfg)
		if got != "https://staging.prosie.app" {
			t.Fatalf("expected trimmed https://staging.prosie.app, got %s", got)
		}
	})

	t.Run("Config ApiURL used when env is unset", func(t *testing.T) {
		t.Setenv("PROSIE_API_URL", "")
		cfg := &Config{ApiURL: "https://custom.prosie.app/"}
		got := ResolveApiURL(cfg)
		if got != "https://custom.prosie.app" {
			t.Fatalf("expected trimmed https://custom.prosie.app, got %s", got)
		}
	})

	t.Run("Fallback to DefaultApiURL", func(t *testing.T) {
		t.Setenv("PROSIE_API_URL", "")
		cfg := &Config{}
		got := ResolveApiURL(cfg)
		if got != DefaultApiURL {
			t.Fatalf("expected %s, got %s", DefaultApiURL, got)
		}
	})
}

func TestResolveToken(t *testing.T) {
	t.Run("PROSIE_API_TOKEN takes precedence", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "env-token-123")
		cfg := &Config{Token: "cfg-token-456"}
		token, source := ResolveToken(cfg)
		if token != "env-token-123" || source != "environment" {
			t.Fatalf("expected env-token-123 from environment, got %s from %s", token, source)
		}
	})

	t.Run("Config token used when env unset", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		cfg := &Config{Token: "cfg-token-456"}
		token, source := ResolveToken(cfg)
		if token != "cfg-token-456" || source != "config" {
			t.Fatalf("expected cfg-token-456 from config, got %s from %s", token, source)
		}
	})

	t.Run("Returns empty when neither set", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		cfg := &Config{}
		token, source := ResolveToken(cfg)
		if token != "" || source != "none" {
			t.Fatalf("expected empty token and source 'none', got %s from %s", token, source)
		}
	})
}
