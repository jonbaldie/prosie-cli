package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMalformedConfig(t *testing.T) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{broken json"), 0600); err != nil {
		t.Fatalf("failed to write malformed config: %v", err)
	}
	return configPath
}

func TestMalformedConfigIsSurfaced(t *testing.T) {
	t.Run("book list reports the config error, not logged-out state", func(t *testing.T) {
		configPath := writeMalformedConfig(t)
		cmd, _, errOut := newTestRootCmd(configPath, &http.Client{})
		code := cmd.Execute([]string{"book", "list"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		stderr := errOut.String()
		if !strings.Contains(stderr, configPath) {
			t.Fatalf("expected config path %q in stderr, got: %s", configPath, stderr)
		}
		if strings.Contains(stderr, "not logged in") {
			t.Fatalf("stderr must not claim logged-out state, got: %s", stderr)
		}
	})

	t.Run("auth status reports the config error, not token_source none", func(t *testing.T) {
		configPath := writeMalformedConfig(t)
		cmd, _, errOut := newTestRootCmd(configPath, &http.Client{})
		code := cmd.Execute([]string{"auth", "status", "--json"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		stderr := errOut.String()
		if !strings.Contains(stderr, configPath) {
			t.Fatalf("expected config path %q in stderr, got: %s", configPath, stderr)
		}
	})

	t.Run("auth login with token refuses to overwrite malformed config", func(t *testing.T) {
		configPath := writeMalformedConfig(t)
		cmd, _, errOut := newTestRootCmd(configPath, &http.Client{})
		code := cmd.Execute([]string{"auth", "login", "--token", "some-token"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		stderr := errOut.String()
		if !strings.Contains(stderr, configPath) {
			t.Fatalf("expected config path %q in stderr, got: %s", configPath, stderr)
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config after login attempt: %v", err)
		}
		if string(data) != "{broken json" {
			t.Fatalf("malformed config was modified, got: %q", string(data))
		}
	})

	t.Run("missing config still reports logged-out state", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")
		cmd, _, errOut := newTestRootCmd(configPath, &http.Client{})
		code := cmd.Execute([]string{"book", "list"})
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "not logged in") {
			t.Fatalf("expected logged-out message for missing config, got: %s", errOut.String())
		}
	})
}
