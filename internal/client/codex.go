package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// CodexEntry represents a character, worldbuilding lore, or setting note in Prosie.
type CodexEntry struct {
	ID        int     `json:"id"`
	StoryID   *int    `json:"story_id,omitempty"`
	SeriesID  *int    `json:"series_id,omitempty"`
	ParentID  *int    `json:"parent_id,omitempty"`
	Category  string  `json:"category,omitempty"`
	Type      string  `json:"type,omitempty"`
	Name      string  `json:"name"`
	Aliases   *string `json:"aliases,omitempty"`
	Content   string  `json:"content,omitempty"`
	Details   string  `json:"details,omitempty"`
	Order     int     `json:"order,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
}

func displayCodexEntryType(e *CodexEntry) string {
	if e.Type != "" {
		return e.Type
	}
	if e.Category != "" {
		return e.Category
	}
	return "lore"
}

func displayCodexEntryDetails(e *CodexEntry) string {
	if e.Details != "" {
		return e.Details
	}
	if e.Content != "" {
		return e.Content
	}
	return "-"
}

func displayCodexEntryAliases(e *CodexEntry) string {
	if e.Aliases != nil && strings.TrimSpace(*e.Aliases) != "" {
		return *e.Aliases
	}
	return "-"
}

func (e *CodexEntry) normalize() {
	if e.Type == "" {
		e.Type = e.Category
	}
	if e.Category == "" {
		e.Category = e.Type
	}
	if e.Details == "" {
		e.Details = e.Content
	}
	if e.Content == "" {
		e.Content = e.Details
	}
	if e.Type == "" {
		e.Type = "lore"
		e.Category = "lore"
	}
}

// CreateCodexParams holds input parameters for creating a codex entry.
type CreateCodexParams struct {
	Name     string  `json:"name"`
	Type     string  `json:"type,omitempty"`
	Category string  `json:"category,omitempty"`
	Details  string  `json:"details,omitempty"`
	Content  string  `json:"content,omitempty"`
	Aliases  *string `json:"aliases,omitempty"`
	Order    *int    `json:"order,omitempty"`
}

// UpdateCodexParams holds input parameters for updating a codex entry.
type UpdateCodexParams struct {
	Name     *string `json:"name,omitempty"`
	Type     *string `json:"type,omitempty"`
	Category *string `json:"category,omitempty"`
	Details  *string `json:"details,omitempty"`
	Content  *string `json:"content,omitempty"`
	Aliases  *string `json:"aliases,omitempty"`
	Order    *int    `json:"order,omitempty"`
}

// ListCodexEntries fetches all standalone codex entries for a book.
func (c *CodexCollection) ListCodexEntries(ctx context.Context, bookID int) ([]CodexEntry, error) {
	path := fmt.Sprintf("/api/stories/%d/codex-entries", bookID)
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data []CodexEntry `json:"data"`
	}
	var entries []CodexEntry
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		entries = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode codex entries response: %w", err)
	}

	for i := range entries {
		entries[i].normalize()
	}

	return entries, nil
}

// ListSeriesCodexEntries fetches shared base codex entries for a series.
func (c *CodexCollection) ListSeriesCodexEntries(ctx context.Context, seriesID int) ([]CodexEntry, error) {
	path := fmt.Sprintf("/api/series/%d/codex-entries", seriesID)
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data []CodexEntry `json:"data"`
	}
	var entries []CodexEntry
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		entries = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode series codex entries response: %w", err)
	}

	for i := range entries {
		entries[i].normalize()
	}

	return entries, nil
}

// GetCodexEntry fetches a single codex entry by ID.
func (c *CodexCollection) GetCodexEntry(ctx context.Context, id int) (*CodexEntry, error) {
	path := fmt.Sprintf("/api/codex-entries/%d", id)
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var entry CodexEntry
	entry, err = decodeCodexEntry(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode codex entry response: %w", err)
	}

	entry.normalize()
	return &entry, nil
}

// CreateCodexEntry creates a new codex entry for a book.
func (c *CodexCollection) CreateCodexEntry(ctx context.Context, bookID int, params CreateCodexParams) (*CodexEntry, error) {
	category := params.Category
	if category == "" {
		category = params.Type
	}
	if category == "" {
		category = "lore"
	}

	content := params.Content
	if content == "" {
		content = params.Details
	}

	body := map[string]any{
		"name":     params.Name,
		"category": category,
		"type":     category,
		"content":  content,
		"details":  content,
	}
	if params.Aliases != nil {
		body["aliases"] = *params.Aliases
	}
	if params.Order != nil {
		body["order"] = *params.Order
	}

	path := fmt.Sprintf("/api/stories/%d/codex-entries", bookID)
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	var entry CodexEntry
	entry, err = decodeCodexEntry(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode create codex entry response: %w", err)
	}

	entry.normalize()
	return &entry, nil
}

// UpdateCodexEntry updates an existing codex entry.
func (c *CodexCollection) UpdateCodexEntry(ctx context.Context, id int, params UpdateCodexParams) (*CodexEntry, error) {
	body := codexUpdateBody(params)

	path := fmt.Sprintf("/api/codex-entries/%d", id)
	bodyBytes, err := c.transport.request(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var entry CodexEntry
	entry, err = decodeCodexEntry(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode update codex entry response: %w", err)
	}

	entry.normalize()
	return &entry, nil
}

// DeleteCodexEntry deletes a codex entry by ID.
func (c *CodexCollection) DeleteCodexEntry(ctx context.Context, id int) error {
	path := fmt.Sprintf("/api/codex-entries/%d", id)
	req, err := c.transport.NewRequest(ctx, http.MethodDelete, path, nil)
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

func decodeCodexEntry(data []byte) (CodexEntry, error) {
	var envelope struct {
		Data CodexEntry `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Data.ID != 0 {
		return envelope.Data, nil
	}
	var resource CodexEntry
	err := json.Unmarshal(data, &resource)
	return resource, err
}

func codexUpdateBody(params UpdateCodexParams) map[string]any {
	body := map[string]any{}

	if params.Name != nil {
		body["name"] = *params.Name
	}
	if params.Category != nil {
		body["category"] = *params.Category
		body["type"] = *params.Category
	} else if params.Type != nil {
		body["category"] = *params.Type
		body["type"] = *params.Type
	}
	if params.Content != nil {
		body["content"] = *params.Content
		body["details"] = *params.Content
	} else if params.Details != nil {
		body["content"] = *params.Details
		body["details"] = *params.Details
	}
	if params.Aliases != nil {
		body["aliases"] = *params.Aliases
	}
	if params.Order != nil {
		body["order"] = *params.Order
	}

	return body
}

// CodexCollection owns codex operations over the shared authenticated transport.
type CodexCollection struct{ transport *Client }

// Codex returns the codex module for this client.
func (c *Client) Codex() *CodexCollection { return &CodexCollection{transport: c} }

// CodexEntryDisplay contains the text used to display a codexentry.
type CodexEntryDisplay struct{ Type, Details, Aliases string }

// Display returns the text fields with their user-facing fallbacks.
func (e *CodexEntry) Display() CodexEntryDisplay {
	return CodexEntryDisplay{Type: displayCodexEntryType(e), Details: displayCodexEntryDetails(e), Aliases: displayCodexEntryAliases(e)}
}
