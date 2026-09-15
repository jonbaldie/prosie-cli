package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

type failingLlmWriter struct{}

func (failingLlmWriter) Write([]byte) (int, error) {
	return 0, errors.New("output unavailable")
}

func setupTestLlmEnv(t *testing.T, handler http.HandlerFunc) (string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	configPath := filepath.Join(t.TempDir(), "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-llm-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-llm-token")

	return configPath, server.Client()
}

func TestLlmShow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/llm-config" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"provider":           "openrouter",
				"model":              "openai/gpt-5.6-luna",
				"has_openai_key":     false,
				"has_anthropic_key":  false,
				"has_openrouter_key": true,
			},
		})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)

	cmd, out, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "show"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	stdout := out.String()
	for _, expected := range []string{
		"Provider:       openrouter",
		"Model:          openai/gpt-5.6-luna",
		"OpenAI key:     not configured",
		"Anthropic key:  not configured",
		"OpenRouter key: configured",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("missing %q in output: %s", expected, stdout)
		}
	}
}

func TestLlmShowJSONPreservesUnsetSettings(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"provider":           nil,
				"model":              nil,
				"has_openai_key":     false,
				"has_anthropic_key":  false,
				"has_openrouter_key": false,
			},
		})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)

	cmd, out, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "show", "--json"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON output: %v, raw: %s", err, out.String())
	}
	if payload["provider"] != nil || payload["model"] != nil {
		t.Fatalf("expected unset settings to remain null, got: %+v", payload)
	}
	if _, exists := payload["openrouter_api_key"]; exists {
		t.Fatalf("secret key appeared in output: %+v", payload)
	}
}

func TestLlmModels(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/llm-models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"providers": map[string]any{
					"openai": map[string]string{
						"gpt-5.6": "GPT-5.6 Sol",
					},
					"openrouter": map[string]string{
						"openai/gpt-5.6-luna": "OpenRouter: OpenAI: GPT-5.6 Luna",
					},
				},
				"configured": []string{"openrouter"},
			},
		})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)

	cmd, out, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "models"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	stdout := out.String()
	for _, expected := range []string{
		"PROVIDER", "KEY", "MODEL", "LABEL",
		"openai", "not configured", "gpt-5.6", "GPT-5.6 Sol",
		"openrouter", "configured", "openai/gpt-5.6-luna", "OpenRouter: OpenAI: GPT-5.6 Luna",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("missing %q in output: %s", expected, stdout)
		}
	}
}

func TestLlmUpdateUsesEnvironmentKeys(t *testing.T) {
	const openAISecret = "secret-openai-key"
	const anthropicSecret = "secret-anthropic-key"
	const openRouterSecret = "secret-openrouter-key"
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/llm-config" {
			http.NotFound(w, r)
			return
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode update payload: %v", err)
		}
		if payload["provider"] != "openrouter" || payload["model"] != "openai/gpt-5.6-luna" {
			t.Fatalf("unexpected update payload: %+v", payload)
		}
		if payload["openai_api_key"] != openAISecret || payload["anthropic_api_key"] != anthropicSecret || payload["openrouter_api_key"] != openRouterSecret {
			t.Fatalf("missing environment keys in update payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"provider":           "openrouter",
				"model":              "openai/gpt-5.6-luna",
				"has_openai_key":     false,
				"has_anthropic_key":  false,
				"has_openrouter_key": true,
			},
		})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)
	t.Setenv("PROSIE_OPENAI_API_KEY", openAISecret)
	t.Setenv("PROSIE_ANTHROPIC_API_KEY", anthropicSecret)
	t.Setenv("PROSIE_OPENROUTER_API_KEY", openRouterSecret)

	cmd, out, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "update", "--provider", "openrouter", "--model", "openai/gpt-5.6-luna", "--json"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	for _, secret := range []string{openAISecret, anthropicSecret, openRouterSecret} {
		if strings.Contains(out.String(), secret) || strings.Contains(errOut.String(), secret) {
			t.Fatalf("secret key appeared in command output")
		}
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON output: %v, raw: %s", err, out.String())
	}
	if payload["provider"] != "openrouter" || payload["has_openrouter_key"] != true {
		t.Fatalf("unexpected update output: %+v", payload)
	}
}

func TestLlmUpdateRequiresASetting(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("update request must not be sent without a setting")
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)
	t.Setenv("PROSIE_OPENAI_API_KEY", "")
	t.Setenv("PROSIE_ANTHROPIC_API_KEY", "")
	t.Setenv("PROSIE_OPENROUTER_API_KEY", "")

	cmd, _, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "update"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "provide --provider, --model, or a PROSIE_*_API_KEY environment variable") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestLlmHelp(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"llm", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	for _, commandName := range []string{"show", "models", "update"} {
		if !strings.Contains(out.String(), commandName) {
			t.Fatalf("missing %q in help output: %s", commandName, out.String())
		}
	}

	cmd, out, errOut = newTestRootCmd("", nil)
	code = cmd.Execute([]string{"llm", "update", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "PROSIE_OPENROUTER_API_KEY") {
		t.Fatalf("missing secure key instructions: %s", out.String())
	}
}

func TestLlmModelsJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"providers": map[string]any{
					"anthropic": map[string]string{"claude-sonnet-5": "Claude Sonnet 5"},
				},
				"configured": []string{"anthropic"},
			},
		})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)

	cmd, out, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "models", "--json"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	var models client.LlmModels
	if err := json.Unmarshal(out.Bytes(), &models); err != nil {
		t.Fatalf("invalid JSON output: %v, raw: %s", err, out.String())
	}
	if models.Providers["anthropic"]["claude-sonnet-5"] != "Claude Sonnet 5" {
		t.Fatalf("unexpected models output: %+v", models)
	}
}

func TestLlmRejectsAPIKeysInArguments(t *testing.T) {
	const secret = "must-not-appear"
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"llm", "update", "--openrouter-api-key", secret})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if strings.Contains(out.String(), secret) || strings.Contains(errOut.String(), secret) {
		t.Fatal("secret key appeared in command output")
	}
}

func TestLlmCommandsRejectPositionalArguments(t *testing.T) {
	tests := [][]string{
		{"llm", "show", "unexpected"},
		{"llm", "models", "unexpected"},
		{"llm", "update", "--provider", "openai", "secret-value"},
	}

	for _, args := range tests {
		cmd, _, errOut := newTestRootCmd("", nil)
		code := cmd.Execute(args)
		if code != 1 {
			t.Fatalf("expected code 1 for %v, got %d", args, code)
		}
		if !strings.Contains(errOut.String(), "does not accept positional arguments") {
			t.Fatalf("unexpected stderr for %v: %s", args, errOut.String())
		}
		if strings.Contains(errOut.String(), "secret-value") {
			t.Fatalf("positional value appeared in stderr for %v", args)
		}
	}
}

func TestLlmJSONOutputReportsWriteErrors(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"provider":           "openrouter",
				"model":              "openai/gpt-5.6-luna",
				"has_openai_key":     false,
				"has_anthropic_key":  false,
				"has_openrouter_key": true,
			},
		})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)

	cmd, _, errOut := newTestRootCmd(configPath, httpClient)
	cmd.Out = failingLlmWriter{}
	code := cmd.Execute([]string{"llm", "show", "--json"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "error writing output: output unavailable") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestLlmAPIErrorsUseStandardError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "insufficient_scope"})
	}
	configPath, httpClient := setupTestLlmEnv(t, handler)

	cmd, out, errOut := newTestRootCmd(configPath, httpClient)
	code := cmd.Execute([]string{"llm", "show"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty stdout, got: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "API error (403): insufficient_scope") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}
