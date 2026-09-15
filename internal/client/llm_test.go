package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLlmConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	config, err := cli.LLM().GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if config.Provider == nil || *config.Provider != "openrouter" || config.Model == nil || *config.Model != "openai/gpt-5.6-luna" {
		t.Fatalf("unexpected LLM config: %+v", config)
	}
	if config.HasOpenAIKey || config.HasAnthropicKey || !config.HasOpenRouterKey {
		t.Fatalf("unexpected key status: %+v", config)
	}
}

func TestGetLlmModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	models, err := cli.LLM().GetModels(context.Background())
	if err != nil {
		t.Fatalf("GetModels returned error: %v", err)
	}
	if models.Providers["openai"]["gpt-5.6"] != "GPT-5.6 Sol" {
		t.Fatalf("unexpected provider models: %+v", models.Providers)
	}
	if len(models.Configured) != 1 || models.Configured[0] != "openrouter" {
		t.Fatalf("unexpected configured providers: %+v", models.Configured)
	}
}

func TestUpdateLlmConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/llm-config" {
			http.NotFound(w, r)
			return
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if payload["provider"] != "openrouter" || payload["model"] != "openai/gpt-5.6-luna" {
			t.Fatalf("unexpected settings payload: %+v", payload)
		}
		if payload["openrouter_api_key"] != "secret-openrouter-key" {
			t.Fatalf("missing OpenRouter key: %+v", payload)
		}
		if _, exists := payload["openai_api_key"]; exists {
			t.Fatalf("unexpected OpenAI key field: %+v", payload)
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
	}))
	defer server.Close()

	provider := "openrouter"
	model := "openai/gpt-5.6-luna"
	openRouterKey := "secret-openrouter-key"
	cli := New(server.URL, "test-token", server.Client())
	config, err := cli.LLM().UpdateConfig(context.Background(), UpdateLlmConfigParams{
		Provider:         &provider,
		Model:            &model,
		OpenRouterAPIKey: &openRouterKey,
	})
	if err != nil {
		t.Fatalf("UpdateConfig returned error: %v", err)
	}
	if config.Provider == nil || *config.Provider != provider || config.Model == nil || *config.Model != model || !config.HasOpenRouterKey {
		t.Fatalf("unexpected updated config: %+v", config)
	}
}

func TestGetLlmConfigRejectsMissingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	_, err := cli.LLM().GetConfig(context.Background())
	if err == nil {
		t.Fatal("expected missing data to return an error")
	}
}

func TestUpdateLlmConfigRejectsMissingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	model := "gpt-5.6"
	cli := New(server.URL, "test-token", server.Client())
	_, err := cli.LLM().UpdateConfig(context.Background(), UpdateLlmConfigParams{Model: &model})
	if err == nil {
		t.Fatal("expected missing data to return an error")
	}
}

func TestGetLlmModelsRejectsMissingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	_, err := cli.LLM().GetModels(context.Background())
	if err == nil {
		t.Fatal("expected missing data to return an error")
	}
}
