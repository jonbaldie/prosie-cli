package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Book represents a novel or book manuscript in Prosie.
type Book struct {
	ID                    int       `json:"id"`
	Title                 string    `json:"title"`
	Subtitle              *string   `json:"subtitle,omitempty"`
	SeriesID              *int      `json:"series_id,omitempty"`
	POV                   string    `json:"pov,omitempty"`
	Tense                 string    `json:"tense,omitempty"`
	Lore                  *string   `json:"lore,omitempty"`
	Characters            *string   `json:"characters,omitempty"`
	StorySoFar           *string   `json:"story_so_far,omitempty"`
	Premise               *string   `json:"premise,omitempty"`
	ProseStyle            *string   `json:"prose_style,omitempty"`
	TargetWordCount       *int      `json:"target_word_count,omitempty"`
	FilterUsingStorySoFar bool      `json:"filter_using_story_so_far"`
	WordCount             int       `json:"word_count"`
	TargetProgressPercent *int      `json:"target_progress_percent,omitempty"`
	CreatedAt             string    `json:"created_at,omitempty"`
	UpdatedAt             string    `json:"updated_at,omitempty"`
	Chapters              []Chapter `json:"chapters"`
}

// Chapter represents a chapter or scene in a book.
type Chapter struct {
	ID        int    `json:"id"`
	StoryID   int    `json:"story_id,omitempty"`
	Order     int    `json:"order"`
	Name      string `json:"name,omitempty"`
	Title     string `json:"title,omitempty"`
	WordCount int    `json:"word_count,omitempty"`
}

// DisplayTitle returns the chapter name, title, or fallback chapter number.
func (c Chapter) DisplayTitle() string {
	if c.Title != "" {
		return c.Title
	}
	if c.Name != "" {
		return c.Name
	}
	return fmt.Sprintf("Chapter %d", c.Order+1)
}

// DisplayPremise returns the best available premise or summary string.
func (b *Book) DisplayPremise() string {
	if b.Premise != nil && *b.Premise != "" {
		return *b.Premise
	}
	if b.StorySoFar != nil && *b.StorySoFar != "" {
		return *b.StorySoFar
	}
	if b.Subtitle != nil && *b.Subtitle != "" {
		return *b.Subtitle
	}
	return "-"
}

// DisplayLore returns the lore content or a dash.
func (b *Book) DisplayLore() string {
	if b.Lore != nil && *b.Lore != "" {
		return *b.Lore
	}
	return "-"
}

// DisplayCharacters returns the character list or a dash.
func (b *Book) DisplayCharacters() string {
	if b.Characters != nil && *b.Characters != "" {
		return *b.Characters
	}
	return "-"
}

func (b *Book) normalize() {
	if b.Premise == nil && b.StorySoFar != nil {
		b.Premise = b.StorySoFar
	}
	if b.StorySoFar == nil && b.Premise != nil {
		b.StorySoFar = b.Premise
	}
	if b.Chapters == nil {
		b.Chapters = []Chapter{}
	}
}

// CreateBookParams holds input fields for creating a book.
type CreateBookParams struct {
	Title           string  `json:"title"`
	Premise         *string `json:"premise,omitempty"`
	StorySoFar      *string `json:"story_so_far,omitempty"`
	Lore            *string `json:"lore,omitempty"`
	Characters      *string `json:"characters,omitempty"`
	TargetWordCount *int    `json:"target_word_count,omitempty"`
}

// UpdateBookParams holds input fields for updating a book.
type UpdateBookParams struct {
	Title           *string `json:"title,omitempty"`
	Premise         *string `json:"premise,omitempty"`
	StorySoFar      *string `json:"story_so_far,omitempty"`
	Lore            *string `json:"lore,omitempty"`
	Characters      *string `json:"characters,omitempty"`
	TargetWordCount *int    `json:"target_word_count,omitempty"`
}

// ListBooks fetches all books owned by the authenticated user.
func (c *Client) ListBooks(ctx context.Context) ([]Book, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/stories", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/stories failed: %w", err)
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
		Data []Book `json:"data"`
	}
	var books []Book
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		books = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &books); err != nil {
		return nil, fmt.Errorf("failed to decode books response: %w", err)
	}

	for i := range books {
		books[i].normalize()
	}

	return books, nil
}

// GetBook fetches a single book by ID and populates its chapters.
func (c *Client) GetBook(ctx context.Context, id int) (*Book, error) {
	path := fmt.Sprintf("/api/stories/%d", id)
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
		Data Book `json:"data"`
	}
	var book Book
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		book = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &book); err != nil {
		return nil, fmt.Errorf("failed to decode book response: %w", err)
	}

	if len(book.Chapters) == 0 {
		scenesPath := fmt.Sprintf("/api/stories/%d/scenes", id)
		scenesReq, err := c.NewRequest(ctx, http.MethodGet, scenesPath, nil)
		if err == nil {
			scenesResp, err := c.Do(scenesReq)
			if err == nil {
				defer scenesResp.Body.Close()
				if scenesResp.StatusCode == http.StatusOK {
					scenesBytes, _ := io.ReadAll(scenesResp.Body)
					var scenesEnv struct {
						Data []Chapter `json:"data"`
					}
					if err := json.Unmarshal(scenesBytes, &scenesEnv); err == nil && len(scenesEnv.Data) > 0 {
						book.Chapters = scenesEnv.Data
					} else {
						var chapters []Chapter
						if err := json.Unmarshal(scenesBytes, &chapters); err == nil {
							book.Chapters = chapters
						}
					}
				}
			}
		}
	}

	book.normalize()
	return &book, nil
}

// CreateBook creates a new book manuscript.
func (c *Client) CreateBook(ctx context.Context, params CreateBookParams) (*Book, error) {
	body := map[string]any{
		"title": params.Title,
	}
	if params.Premise != nil {
		body["premise"] = *params.Premise
		body["story_so_far"] = *params.Premise
	}
	if params.StorySoFar != nil {
		body["story_so_far"] = *params.StorySoFar
	}
	if params.Lore != nil {
		body["lore"] = *params.Lore
	}
	if params.Characters != nil {
		body["characters"] = *params.Characters
	}
	if params.TargetWordCount != nil {
		body["target_word_count"] = *params.TargetWordCount
	}

	req, err := c.NewRequest(ctx, http.MethodPost, "/api/stories", body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/stories failed: %w", err)
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
		Data Book `json:"data"`
	}
	var book Book
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		book = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &book); err != nil {
		return nil, fmt.Errorf("failed to decode create book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// UpdateBook updates an existing book manuscript.
func (c *Client) UpdateBook(ctx context.Context, id int, params UpdateBookParams) (*Book, error) {
	body := map[string]any{}
	if params.Title != nil {
		body["title"] = *params.Title
	}
	if params.Premise != nil {
		body["premise"] = *params.Premise
		body["story_so_far"] = *params.Premise
	}
	if params.StorySoFar != nil {
		body["story_so_far"] = *params.StorySoFar
	}
	if params.Lore != nil {
		body["lore"] = *params.Lore
	}
	if params.Characters != nil {
		body["characters"] = *params.Characters
	}
	if params.TargetWordCount != nil {
		body["target_word_count"] = *params.TargetWordCount
	}

	path := fmt.Sprintf("/api/stories/%d", id)
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
		Data Book `json:"data"`
	}
	var book Book
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		book = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &book); err != nil {
		return nil, fmt.Errorf("failed to decode update book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// DeleteBook removes a book manuscript by ID.
func (c *Client) DeleteBook(ctx context.Context, id int) error {
	path := fmt.Sprintf("/api/stories/%d", id)
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

// DuplicateBook clones a book with all its chapters and codex entries.
func (c *Client) DuplicateBook(ctx context.Context, id int) (*Book, error) {
	path := fmt.Sprintf("/api/stories/%d/duplicate", id)
	req, err := c.NewRequest(ctx, http.MethodPost, path, nil)
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
		Data Book `json:"data"`
	}
	var book Book
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		book = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &book); err != nil {
		return nil, fmt.Errorf("failed to decode duplicate book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// ExportStory downloads the complete book prose in markdown or docx format.
func (c *Client) ExportStory(ctx context.Context, id int, format string) ([]byte, error) {
	if format == "" {
		format = "markdown"
	}
	path := fmt.Sprintf("/api/stories/%d/export?format=%s", id, url.QueryEscape(format))
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

// ImportDocx uploads a DOCX file to create a new book with chapters.
func (c *Client) ImportDocx(ctx context.Context, filePath string, title string) (*Book, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if title == "" {
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		title = strings.TrimSuffix(base, ext)
		if title == "" {
			title = "Imported Book"
		}
	}

	if err := writer.WriteField("title", title); err != nil {
		return nil, fmt.Errorf("failed to write title field: %w", err)
	}

	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create document form field: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := c.BaseURL + "/api/stories/import-docx"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/stories/import-docx failed: %w", err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Data   Book      `json:"data"`
		Scenes []Chapter `json:"scenes"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err == nil && result.Data.ID != 0 {
		book := result.Data
		if len(result.Scenes) > 0 && len(book.Chapters) == 0 {
			book.Chapters = result.Scenes
		}
		book.normalize()
		return &book, nil
	}

	var book Book
	if err := json.Unmarshal(bodyBytes, &book); err != nil {
		return nil, fmt.Errorf("failed to decode import response: %w", err)
	}
	book.normalize()
	return &book, nil
}
