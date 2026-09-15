package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// LlmConfig is the token owner's non-secret LLM configuration.
type LlmConfig struct {
	Provider         *string `json:"provider"`
	Model            *string `json:"model"`
	HasOpenAIKey     bool    `json:"has_openai_key"`
	HasAnthropicKey  bool    `json:"has_anthropic_key"`
	HasOpenRouterKey bool    `json:"has_openrouter_key"`
}

// LlmModels is the model catalogue and the providers with stored keys.
type LlmModels struct {
	Providers  map[string]map[string]string `json:"providers"`
	Configured []string                     `json:"configured"`
}

// UpdateLlmConfigParams contains the LLM settings supplied by the user.
type UpdateLlmConfigParams struct {
	Provider         *string `json:"provider,omitempty"`
	Model            *string `json:"model,omitempty"`
	OpenAIAPIKey     *string `json:"openai_api_key,omitempty"`
	AnthropicAPIKey  *string `json:"anthropic_api_key,omitempty"`
	OpenRouterAPIKey *string `json:"openrouter_api_key,omitempty"`
}

// GetConfig reads the token owner's LLM configuration.
func (c *LlmCollection) GetConfig(ctx context.Context) (*LlmConfig, error) {
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, "/api/llm-config", nil)
	if err != nil {
		return nil, err
	}

	return decodeLlmConfig(bodyBytes)
}

// GetModels reads the available model catalogue.
func (c *LlmCollection) GetModels(ctx context.Context) (*LlmModels, error) {
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, "/api/llm-models", nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data *LlmModels `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode LLM models response: %w", err)
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("failed to decode LLM models response: missing data")
	}

	return envelope.Data, nil
}

// UpdateConfig changes only the supplied LLM settings.
func (c *LlmCollection) UpdateConfig(ctx context.Context, params UpdateLlmConfigParams) (*LlmConfig, error) {
	bodyBytes, err := c.transport.request(ctx, http.MethodPatch, "/api/llm-config", params)
	if err != nil {
		return nil, err
	}

	return decodeLlmConfig(bodyBytes)
}

func decodeLlmConfig(bodyBytes []byte) (*LlmConfig, error) {
	var envelope struct {
		Data *LlmConfig `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode LLM config response: %w", err)
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("failed to decode LLM config response: missing data")
	}
	return envelope.Data, nil
}

// LlmCollection owns LLM configuration operations over the authenticated transport.
type LlmCollection struct{ transport *Client }

// LLM returns the LLM configuration module for this client.
func (c *Client) LLM() *LlmCollection { return &LlmCollection{transport: c} }
