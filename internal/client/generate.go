package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Usage tracks prompt, completion, and total tokens used by an LLM operation.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ContinueResult represents the output of a chapter prose continuation.
type ContinueResult struct {
	Prose                 string `json:"prose"`
	HTML                  string `json:"html,omitempty"`
	Persisted             bool   `json:"persisted"`
	Model                 string `json:"model,omitempty"`
	Usage                 *Usage `json:"usage,omitempty"`
	EstimatedPromptTokens int    `json:"estimated_prompt_tokens,omitempty"`
}

// RewriteParams holds parameters for rewriting a selection of text.
type RewriteParams struct {
	Selection   string `json:"selection"`
	Instruction string `json:"instruction,omitempty"`
	Action      string `json:"action,omitempty"`
	Persist     bool   `json:"persist"`
}

// RewriteResult represents the output of a text rewrite.
type RewriteResult struct {
	Prose                 string `json:"prose"`
	Model                 string `json:"model,omitempty"`
	Persisted             bool   `json:"persisted"`
	AppliedContent        string `json:"applied_content,omitempty"`
	Usage                 *Usage `json:"usage,omitempty"`
	EstimatedPromptTokens int    `json:"estimated_prompt_tokens,omitempty"`
}

// SummaryResult represents the output of a chapter summary generation.
type SummaryResult struct {
	Summary               string `json:"summary"`
	Persisted             bool   `json:"persisted"`
	Model                 string `json:"model,omitempty"`
	Usage                 *Usage `json:"usage,omitempty"`
	EstimatedPromptTokens int    `json:"estimated_prompt_tokens,omitempty"`
}

// Continue requests AI continuation for a chapter without streaming.
func (c *Generation) Continue(ctx context.Context, chapterID string, persist bool) (*ContinueResult, error) {
	body := map[string]any{
		"persist": persist,
	}

	path := fmt.Sprintf("/api/scenes/%s/continue", url.PathEscape(chapterID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var result ContinueResult
	if dataBytes, ok := raw["data"]; ok {
		if err := json.Unmarshal(dataBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode continue data: %w", err)
		}
	} else {
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode continue response: %w", err)
		}
	}

	return &result, nil
}

// StreamContinue requests AI continuation for a chapter and streams tokens in real time.
func (c *Generation) StreamContinue(ctx context.Context, chapterID string, persist bool, onToken func(string)) (*ContinueResult, error) {
	body := map[string]any{
		"persist": persist,
	}

	path := fmt.Sprintf("/api/scenes/%s/continue/stream", url.PathEscape(chapterID))
	resp, err := c.transport.openGenerationStream(ctx, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res, accumulated, err := streamSSE[ContinueResult](ctx, resp.Body, onToken)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = &ContinueResult{
			Prose:     accumulated,
			Persisted: persist,
		}
	}

	return res, nil
}

// CancelContinue cancels an in-flight continuation generation on the server.
func (c *Generation) CancelContinue(ctx context.Context, chapterID string) error {
	path := fmt.Sprintf("/api/scenes/%s/continue/cancel", url.PathEscape(chapterID))
	req, err := c.transport.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.transport.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	return CheckResponse(resp)
}

// RejectContinuation reverts the latest AI continuation run on the chapter.
func (c *Generation) RejectContinuation(ctx context.Context, chapterID string) (*Chapter, error) {
	path := fmt.Sprintf("/api/scenes/%s/reject-continuation", url.PathEscape(chapterID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data Chapter `json:"data"`
	}
	var chapter Chapter
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		chapter = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &chapter); err != nil {
		return nil, fmt.Errorf("failed to decode reject continuation response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// Rewrite rewrites a selection of text using prompt instructions or an action key.
func (c *Generation) Rewrite(ctx context.Context, chapterID string, params RewriteParams) (*RewriteResult, error) {
	body := map[string]any{
		"selection": params.Selection,
		"persist":   params.Persist,
	}
	if params.Instruction != "" {
		body["instruction"] = params.Instruction
	}
	if params.Action != "" {
		body["action"] = params.Action
	}

	path := fmt.Sprintf("/api/scenes/%s/rewrite", url.PathEscape(chapterID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var result RewriteResult
	if dataBytes, ok := raw["data"]; ok {
		if err := json.Unmarshal(dataBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode rewrite data: %w", err)
		}
	} else {
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode rewrite response: %w", err)
		}
	}

	return &result, nil
}

// StreamRewrite rewrites a selection of text and streams tokens in real time.
func (c *Generation) StreamRewrite(ctx context.Context, chapterID string, params RewriteParams, onToken func(string)) (*RewriteResult, error) {
	body := map[string]any{
		"selection": params.Selection,
		"persist":   params.Persist,
	}
	if params.Instruction != "" {
		body["instruction"] = params.Instruction
	}
	if params.Action != "" {
		body["action"] = params.Action
	}

	path := fmt.Sprintf("/api/scenes/%s/rewrite/stream", url.PathEscape(chapterID))
	resp, err := c.transport.openGenerationStream(ctx, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res, accumulated, err := streamSSE[RewriteResult](ctx, resp.Body, onToken)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = &RewriteResult{
			Prose:     accumulated,
			Persisted: params.Persist,
		}
	}

	return res, nil
}

// Summarize requests chapter summary generation and stores it on the server.
func (c *Generation) Summarize(ctx context.Context, chapterID string) (*SummaryResult, error) {
	body := map[string]any{
		"persist": true,
	}

	path := fmt.Sprintf("/api/scenes/%s/summarize", url.PathEscape(chapterID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var result SummaryResult
	if dataBytes, ok := raw["data"]; ok {
		if err := json.Unmarshal(dataBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode summary data: %w", err)
		}
	} else {
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode summary response: %w", err)
		}
	}

	return &result, nil
}

// streamSSE reads and decodes Server-Sent Events from an HTTP response stream.
func streamSSE[T any](ctx context.Context, body io.Reader, onToken func(string)) (*T, string, error) {
	state := generationEvents[T]{tokens: streamTokens{onToken: onToken}}
	err := readEvents(body, false, state.accept)
	if ctx.Err() != nil {
		return nil, state.tokens.content.String(), ctx.Err()
	}
	if err != nil {
		return nil, state.tokens.content.String(), err
	}
	if state.err != nil {
		return nil, state.tokens.content.String(), state.err
	}
	return state.done, state.tokens.content.String(), nil
}

// Generation owns generation operations over the shared authenticated transport.
type Generation struct{ transport *Client }

// Generation returns the generation module for this client.
func (c *Client) Generation() *Generation { return &Generation{transport: c} }

func (c *Client) openGenerationStream(ctx context.Context, path string, body any) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	httpClient := c.HTTPClient
	if httpClient.Timeout > 0 {
		clone := *httpClient
		clone.Timeout = 0
		httpClient = &clone
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}

	if err := CheckResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}
