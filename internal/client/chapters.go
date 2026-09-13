package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Chapter represents a chapter or scene in a book manuscript.
type Chapter struct {
	ID              int     `json:"id"`
	StoryID         int     `json:"story_id,omitempty"`
	Order           int     `json:"order"`
	Name            string  `json:"name,omitempty"`
	Title           string  `json:"title,omitempty"`
	CustomPOV       *string `json:"custom_pov,omitempty"`
	CustomCharacter *string `json:"custom_character,omitempty"`
	Content         string  `json:"content,omitempty"`
	Summary         *string `json:"summary,omitempty"`
	WordCount       int     `json:"word_count,omitempty"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

// DisplayTitle returns the chapter title, name, or a fallback chapter number.
func (c Chapter) DisplayTitle() string {
	if c.Title != "" {
		return c.Title
	}
	if c.Name != "" {
		return c.Name
	}
	return fmt.Sprintf("Chapter %d", c.Order+1)
}

// DisplaySummary returns the chapter summary or a dash when empty.
func (c Chapter) DisplaySummary() string {
	if c.Summary != nil && *c.Summary != "" {
		return *c.Summary
	}
	return "-"
}

func (c *Chapter) normalize() {
	if c.Title == "" && c.Name != "" {
		c.Title = c.Name
	}
	if c.Name == "" && c.Title != "" {
		c.Name = c.Title
	}
}

// CreateChapterParams holds parameters for creating a new chapter.
type CreateChapterParams struct {
	Title   *string `json:"title,omitempty"`
	Name    *string `json:"name,omitempty"`
	Content *string `json:"content,omitempty"`
	Summary *string `json:"summary,omitempty"`
	Order   *int    `json:"order,omitempty"`
}

// UpdateChapterParams holds parameters for updating an existing chapter.
type UpdateChapterParams struct {
	Title   *string `json:"title,omitempty"`
	Name    *string `json:"name,omitempty"`
	Content *string `json:"content,omitempty"`
	Summary *string `json:"summary,omitempty"`
	Order   *int    `json:"order,omitempty"`
}

// ListChapters fetches all chapters in a book.
func (c *Client) ListChapters(ctx context.Context, bookID string) ([]Chapter, error) {
	path := fmt.Sprintf("/api/stories/%s/scenes", url.PathEscape(bookID))
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data []Chapter `json:"data"`
	}
	var chapters []Chapter
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		chapters = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &chapters); err != nil {
		return nil, fmt.Errorf("failed to decode chapters response: %w", err)
	}

	for i := range chapters {
		chapters[i].normalize()
	}

	return chapters, nil
}

// GetChapter fetches a single chapter by ID.
func (c *Client) GetChapter(ctx context.Context, id string) (*Chapter, error) {
	path := fmt.Sprintf("/api/scenes/%s", url.PathEscape(id))
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data Chapter `json:"data"`
	}
	var chapter Chapter
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		chapter = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &chapter); err != nil {
		return nil, fmt.Errorf("failed to decode chapter response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// CreateChapter creates a new chapter within a book.
func (c *Client) CreateChapter(ctx context.Context, bookID string, params CreateChapterParams) (*Chapter, error) {
	body := make(map[string]any)
	if params.Title != nil {
		body["name"] = *params.Title
		body["title"] = *params.Title
	}
	if params.Name != nil {
		body["name"] = *params.Name
	}
	if params.Content != nil {
		body["content"] = *params.Content
	}
	if params.Summary != nil {
		body["summary"] = *params.Summary
	}
	if params.Order != nil {
		body["order"] = *params.Order
	}

	path := fmt.Sprintf("/api/stories/%s/scenes", url.PathEscape(bookID))
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data Chapter `json:"data"`
	}
	var chapter Chapter
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		chapter = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &chapter); err != nil {
		return nil, fmt.Errorf("failed to decode create chapter response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// UpdateChapter updates an existing chapter by ID.
func (c *Client) UpdateChapter(ctx context.Context, id string, params UpdateChapterParams) (*Chapter, error) {
	body := make(map[string]any)
	if params.Title != nil {
		body["name"] = *params.Title
		body["title"] = *params.Title
	}
	if params.Name != nil {
		body["name"] = *params.Name
	}
	if params.Content != nil {
		body["content"] = *params.Content
	}
	if params.Summary != nil {
		body["summary"] = *params.Summary
	}
	if params.Order != nil {
		body["order"] = *params.Order
	}

	path := fmt.Sprintf("/api/scenes/%s", url.PathEscape(id))
	req, err := c.NewRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data Chapter `json:"data"`
	}
	var chapter Chapter
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		chapter = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &chapter); err != nil {
		return nil, fmt.Errorf("failed to decode update chapter response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// ReorderChapters updates chapter sequence order for a book.
func (c *Client) ReorderChapters(ctx context.Context, bookID string, order []string) ([]Chapter, error) {
	sceneIDs := make([]any, 0, len(order))
	for _, idStr := range order {
		trimmed := strings.TrimSpace(idStr)
		if intVal, err := strconv.Atoi(trimmed); err == nil {
			sceneIDs = append(sceneIDs, intVal)
		} else {
			sceneIDs = append(sceneIDs, trimmed)
		}
	}

	body := map[string]any{
		"scene_ids": sceneIDs,
	}

	path := fmt.Sprintf("/api/stories/%s/scenes/reorder", url.PathEscape(bookID))
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data []Chapter `json:"data"`
	}
	var chapters []Chapter
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		chapters = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &chapters); err != nil {
		return nil, fmt.Errorf("failed to decode reorder chapters response: %w", err)
	}

	for i := range chapters {
		chapters[i].normalize()
	}

	return chapters, nil
}

// DeleteChapter removes a chapter by ID.
func (c *Client) DeleteChapter(ctx context.Context, id string) error {
	path := fmt.Sprintf("/api/scenes/%s", url.PathEscape(id))
	req, err := c.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	return CheckResponse(resp)
}

// ExportChapter downloads the chapter prose in markdown or specified format.
func (c *Client) ExportChapter(ctx context.Context, id string, format string) ([]byte, error) {
	if format == "" {
		format = "markdown"
	}
	path := fmt.Sprintf("/api/scenes/%s/export?format=%s", url.PathEscape(id), url.QueryEscape(format))
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}
