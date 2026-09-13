package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/auth"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func newTestRootCmd(configPath string, httpClient *http.Client) (*RootCmd, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &RootCmd{
		In:            &bytes.Buffer{},
		Out:           out,
		Err:           errOut,
		ConfigPath:    configPath,
		HTTPClient:    httpClient,
		BrowserOpener: func(url string) error { return nil },
	}
	return cmd, out, errOut
}

func TestVersionCommand(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd("", nil)
		code := cmd.Execute([]string{"version"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "prosie version") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd("", nil)
		code := cmd.Execute([]string{"version", "--json"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		var payload map[string]string
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("invalid json output: %v, raw: %s", err, out.String())
		}
		if payload["version"] == "" {
			t.Fatalf("expected version in json, got %+v", payload)
		}
	})
}

func TestHelpCommand(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"--help"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage text, got %s", out.String())
	}
}

func TestAuthLoginWithToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user" {
			if r.Header.Get("Authorization") == "Bearer valid-pat-token" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":    7,
					"name":  "Jane Developer",
					"email": "jane@example.com",
				})
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated."})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	t.Setenv("PROSIE_API_URL", server.URL)

	t.Run("successful login with token", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "login", "--token", "valid-pat-token"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Logged in as Jane Developer") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}

		// Verify stored configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			t.Fatalf("failed to load saved config: %v", err)
		}
		if cfg.Token != "valid-pat-token" {
			t.Fatalf("expected stored token, got %q", cfg.Token)
		}
	})

	t.Run("successful login with token --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "login", "--token", "valid-pat-token", "--json"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res auth.LoginResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("expected valid JSON: %v, got %s", err, out.String())
		}
		if res.Status != "authenticated" || res.User.Email != "jane@example.com" {
			t.Fatalf("unexpected login result: %+v", res)
		}
	})

	t.Run("rejected with invalid token", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "login", "--token", "invalid-token"})
		if code != 1 {
			t.Fatalf("expected exit code 1 for invalid token, got %d", code)
		}
		if !strings.Contains(errOut.String(), "authentication failed") {
			t.Fatalf("expected authentication error in stderr, got: %s", errOut.String())
		}
	})
}

func TestAuthLoginDeviceFlow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	tokenIssued := false
	var openedURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/oauth/device/code":
			_ = json.NewEncoder(w).Encode(auth.DeviceCodeResponse{
				DeviceCode:              "dev-oauth-code",
				UserCode:                "FLOW-1234",
				VerificationURI:         "https://example.com/oauth/device",
				VerificationURIComplete: "https://example.com/oauth/device?user_code=FLOW-1234",
				ExpiresIn:               60,
				Interval:                1,
			})
		case "/oauth/token":
			if !tokenIssued {
				tokenIssued = true
				_ = json.NewEncoder(w).Encode(auth.TokenResponse{
					AccessToken: "oauth-minted-token",
					TokenType:   "Bearer",
					Scope:       "read write generate",
				})
				return
			}
		case "/api/user":
			if r.Header.Get("Authorization") == "Bearer oauth-minted-token" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":    99,
					"name":  "OAuth User",
					"email": "oauth@example.com",
				})
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("PROSIE_API_URL", server.URL)

	cmd, out, errOut := newTestRootCmd(configPath, server.Client())
	cmd.BrowserOpener = func(u string) error {
		openedURL = u
		return nil
	}

	code := cmd.Execute([]string{"auth", "login"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}

	if openedURL != "https://example.com/oauth/device?user_code=FLOW-1234" {
		t.Fatalf("unexpected browser url: %s", openedURL)
	}
	if !strings.Contains(out.String(), "FLOW-1234") {
		t.Fatalf("expected user code FLOW-1234 in stdout: %s", out.String())
	}
	if !strings.Contains(out.String(), "Logged in as OAuth User") {
		t.Fatalf("expected user confirmation in stdout: %s", out.String())
	}

	// Verify saved credentials
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if cfg.Token != "oauth-minted-token" {
		t.Fatalf("expected oauth-minted-token, got %q", cfg.Token)
	}
	if len(cfg.Scopes) != 3 {
		t.Fatalf("expected 3 scopes saved, got %v", cfg.Scopes)
	}
}

func TestAuthStatus(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer my-token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    10,
				"name":  "Status Tester",
				"email": "tester@example.com",
			})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated."})
	}))
	defer server.Close()

	t.Setenv("PROSIE_API_URL", server.URL)

	t.Run("status when not logged in", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		_ = config.Clear(configPath)

		cmd, _, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "status"})
		if code != 1 {
			t.Fatalf("expected exit code 1 when not logged in, got %d", code)
		}
		if !strings.Contains(errOut.String(), "You are not logged in") {
			t.Fatalf("expected not logged in message, got %s", errOut.String())
		}
	})

	t.Run("status when not logged in --json", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		_ = config.Clear(configPath)

		cmd, out, _ := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "status", "--json"})
		if code != 1 {
			t.Fatalf("expected exit code 1 when not logged in, got %d", code)
		}
		var res auth.StatusResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res.Authenticated {
			t.Fatalf("expected Authenticated: false")
		}
	})

	t.Run("status when logged in via config", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		_ = config.Save(configPath, &config.Config{
			ApiURL: server.URL,
			Token:  "my-token",
			Scopes: []string{"read", "write", "generate"},
		})

		cmd, out, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "status"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Status Tester") || !strings.Contains(out.String(), "tester@example.com") {
			t.Fatalf("missing user details in stdout: %s", out.String())
		}
		if !strings.Contains(out.String(), "read, write, generate") {
			t.Fatalf("missing scopes in stdout: %s", out.String())
		}
	})

	t.Run("status when logged in via config --json", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		_ = config.Save(configPath, &config.Config{
			ApiURL: server.URL,
			Token:  "my-token",
			Scopes: []string{"read", "write", "generate"},
		})

		cmd, out, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "status", "--json"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res auth.StatusResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if !res.Authenticated || res.User.Name != "Status Tester" || res.TokenSource != "config" {
			t.Fatalf("unexpected status result: %+v", res)
		}
	})

	t.Run("status when logged in via PROSIE_API_TOKEN override", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "my-token")
		// Config has different token
		_ = config.Save(configPath, &config.Config{
			ApiURL: server.URL,
			Token:  "stale-token",
		})

		cmd, out, errOut := newTestRootCmd(configPath, server.Client())
		code := cmd.Execute([]string{"auth", "status", "--json"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res auth.StatusResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if !res.Authenticated || res.TokenSource != "environment" {
			t.Fatalf("expected environment source, got %+v", res)
		}
	})
}

func TestAuthLogout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	_ = config.Save(configPath, &config.Config{
		Token: "active-token",
	})

	t.Run("logout clears saved config", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "")
		cmd, out, errOut := newTestRootCmd(configPath, nil)
		code := cmd.Execute([]string{"auth", "logout"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Logged out from Prosie") {
			t.Fatalf("expected logout text, got: %s", out.String())
		}

		cfg, _ := config.Load(configPath)
		if cfg.Token != "" {
			t.Fatalf("token should be cleared from config")
		}
	})

	t.Run("logout --json with PROSIE_API_TOKEN set", func(t *testing.T) {
		t.Setenv("PROSIE_API_TOKEN", "some-env-token")
		cmd, out, errOut := newTestRootCmd(configPath, nil)
		code := cmd.Execute([]string{"auth", "logout", "--json"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["status"] != "logged_out" || res["env_token_override"] != true {
			t.Fatalf("unexpected logout json result: %+v", res)
		}
	})
}

func TestAuthSubcommandHelp(t *testing.T) {
	for _, subcommand := range []string{"login", "status", "logout"} {
		t.Run(subcommand, func(t *testing.T) {
			cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), nil)
			code := cmd.Execute([]string{"auth", subcommand, "--help"})
			if code != 0 || errOut.Len() != 0 {
				t.Fatalf("help must exit 0 without stderr; exit=%d stderr=%s", code, errOut.String())
			}
			if !strings.Contains(out.String(), "prosie auth "+subcommand) || !strings.Contains(out.String(), "--json") {
				t.Fatalf("missing command usage and flags: %s", out.String())
			}
		})
	}
}
