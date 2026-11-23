package client

import (
	"context"
	"fmt"
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

func displayChapterTitle(c *Chapter) string {
	if c.Title != "" {
		return c.Title
	}
	if c.Name != "" {
		return c.Name
	}
	return fmt.Sprintf("Chapter %d", c.Order+1)
}

func displayChapterSummary(c *Chapter) string {
	if c.Summary != nil && *c.Summary != "" {
		return *c.Summary
	}
	return "-"
}

func (c *Chapter) normalize() {
	if c.Title == "" {
		c.Title = c.Name
	}
	if c.Name == "" {
		c.Name = c.Title
	}
}

// CreateChapterParams is the request body for creating a new chapter.
type CreateChapterParams struct {
	Name            *string `json:"name,omitempty"`
	Content         *string `json:"content,omitempty"`
	Summary         *string `json:"summary,omitempty"`
	Order           *int    `json:"order,omitempty"`
	CustomPOV       *string `json:"custom_pov,omitempty"`
	CustomCharacter *string `json:"custom_character,omitempty"`
}

// UpdateChapterParams is the request body for updating an existing chapter.
// Nil fields are omitted, so the server keeps their current values.
type UpdateChapterParams struct {
	Name            *string `json:"name,omitempty"`
	Content         *string `json:"content,omitempty"`
	Summary         *string `json:"summary,omitempty"`
	Order           *int    `json:"order,omitempty"`
	CustomPOV       *string `json:"custom_pov,omitempty"`
	CustomCharacter *string `json:"custom_character,omitempty"`
}

// ListChapters fetches all chapters in a book.
func (c *ChapterCollection) ListChapters(ctx context.Context, bookID string) ([]Chapter, error) {
	path := fmt.Sprintf("/api/stories/%s/scenes", url.PathEscape(bookID))
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	chapters, err := decodeList[Chapter](bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode chapters response: %w", err)
	}

	return chapters, nil
}

// GetChapter fetches a single chapter by ID.
func (c *ChapterCollection) GetChapter(ctx context.Context, id string) (*Chapter, error) {
	path := fmt.Sprintf("/api/scenes/%s", url.PathEscape(id))
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var chapter Chapter
	chapter, err = decodeResource[Chapter](bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode chapter response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// CreateChapter creates a new chapter within a book.
func (c *ChapterCollection) CreateChapter(ctx context.Context, bookID string, params CreateChapterParams) (*Chapter, error) {
	path := fmt.Sprintf("/api/stories/%s/scenes", url.PathEscape(bookID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, err
	}

	var chapter Chapter
	chapter, err = decodeResource[Chapter](bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode create chapter response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// UpdateChapter updates an existing chapter by ID.
func (c *ChapterCollection) UpdateChapter(ctx context.Context, id string, params UpdateChapterParams) (*Chapter, error) {
	path := fmt.Sprintf("/api/scenes/%s", url.PathEscape(id))
	bodyBytes, err := c.transport.request(ctx, http.MethodPatch, path, params)
	if err != nil {
		return nil, err
	}

	var chapter Chapter
	chapter, err = decodeResource[Chapter](bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode update chapter response: %w", err)
	}

	chapter.normalize()
	return &chapter, nil
}

// ReorderChapters updates chapter sequence order for a book.
func (c *ChapterCollection) ReorderChapters(ctx context.Context, bookID string, order []string) ([]Chapter, error) {
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
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	chapters, err := decodeList[Chapter](bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode reorder chapters response: %w", err)
	}

	return chapters, nil
}

// DeleteChapter removes a chapter by ID.
func (c *ChapterCollection) DeleteChapter(ctx context.Context, id string) error {
	path := fmt.Sprintf("/api/scenes/%s", url.PathEscape(id))
	return c.transport.execute(ctx, http.MethodDelete, path, nil)
}

// ExportChapter downloads the chapter prose in markdown or specified format.
func (c *ChapterCollection) ExportChapter(ctx context.Context, id string, format string) ([]byte, error) {
	if format == "" {
		format = "markdown"
	}
	path := fmt.Sprintf("/api/scenes/%s/export?format=%s", url.PathEscape(id), url.QueryEscape(format))
	return c.transport.request(ctx, http.MethodGet, path, nil)
}

// ChapterCollection owns chapters operations over the shared authenticated transport.
type ChapterCollection struct{ transport *Client }

// Chapters returns the chapters module for this client.
func (c *Client) Chapters() *ChapterCollection { return &ChapterCollection{transport: c} }

// ChapterDisplay contains the text used to display a chapter.
type ChapterDisplay struct{ Title, Summary string }

// Display returns the text fields with their user-facing fallbacks.
func (c *Chapter) Display() ChapterDisplay {
	return ChapterDisplay{Title: displayChapterTitle(c), Summary: displayChapterSummary(c)}
}
