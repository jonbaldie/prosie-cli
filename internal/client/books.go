package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// Book represents a novel or book manuscript in Prosie.
type Book struct {
	BookWritingSettings
	ID                    int       `json:"id"`
	Title                 string    `json:"title"`
	Subtitle              *string   `json:"subtitle,omitempty"`
	SeriesID              *int      `json:"series_id,omitempty"`
	Lore                  *string   `json:"lore,omitempty"`
	Characters            *string   `json:"characters,omitempty"`
	StorySoFar            *string   `json:"story_so_far,omitempty"`
	Premise               *string   `json:"premise,omitempty"`
	WordCount             int       `json:"word_count"`
	TargetProgressPercent *int      `json:"target_progress_percent,omitempty"`
	CreatedAt             string    `json:"created_at,omitempty"`
	UpdatedAt             string    `json:"updated_at,omitempty"`
	Chapters              []Chapter `json:"chapters,omitempty"`
}

func displayBookPremise(b *Book) string {
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

func displayBookLore(b *Book) string {
	if b.Lore != nil && *b.Lore != "" {
		return *b.Lore
	}
	return "-"
}

func displayBookCharacters(b *Book) string {
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
func (c *BookCollection) ListBooks(ctx context.Context) ([]Book, error) {
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, "/api/stories", nil)
	if err != nil {
		return nil, err
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
func (c *BookCollection) GetBook(ctx context.Context, id int) (*Book, error) {
	path := fmt.Sprintf("/api/stories/%d", id)
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	book, err := decodeBook(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode book response: %w", err)
	}

	if len(book.Chapters) == 0 {
		book.Chapters = c.bookChapters(ctx, id)
	}

	book.normalize()
	return &book, nil
}

// CreateBook creates a new book manuscript.
func (c *BookCollection) CreateBook(ctx context.Context, params CreateBookParams) (*Book, error) {
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

	bodyBytes, err := c.transport.request(ctx, http.MethodPost, "/api/stories", body)
	if err != nil {
		return nil, err
	}

	book, err := decodeBook(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode create book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// UpdateBook updates an existing book manuscript.
func (c *BookCollection) UpdateBook(ctx context.Context, id int, params UpdateBookParams) (*Book, error) {
	body := bookUpdateBody(params)

	path := fmt.Sprintf("/api/stories/%d", id)
	bodyBytes, err := c.transport.request(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	book, err := decodeBook(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode update book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// DeleteBook removes a book manuscript by ID.
func (c *BookCollection) DeleteBook(ctx context.Context, id int) error {
	path := fmt.Sprintf("/api/stories/%d", id)
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

// DuplicateBook clones a book with all its chapters and codex entries.
func (c *BookCollection) DuplicateBook(ctx context.Context, id int) (*Book, error) {
	path := fmt.Sprintf("/api/stories/%d/duplicate", id)
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	book, err := decodeBook(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode duplicate book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// ExportStory downloads the complete book prose in markdown or docx format.
func (c *BookCollection) ExportStory(ctx context.Context, id int, format string) ([]byte, error) {
	if format == "" {
		format = "markdown"
	}
	path := fmt.Sprintf("/api/stories/%d/export?format=%s", id, url.QueryEscape(format))
	req, err := c.transport.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(req)
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
func (c *BookCollection) ImportDocx(ctx context.Context, filePath string, title string) (*Book, error) {
	title = importedBookTitle(filePath, title)
	req, err := c.transport.uploadRequest(ctx, "/api/stories/import-docx", filePath, "document", map[string]string{"title": title})
	if err != nil {
		return nil, err
	}
	bodyBytes, err := c.transport.readResponse(req, "/api/stories/import-docx")
	if err != nil {
		return nil, err
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

func decodeBook(data []byte) (Book, error) {
	var envelope struct {
		Data Book `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Data.ID != 0 {
		return envelope.Data, nil
	}
	var resource Book
	err := json.Unmarshal(data, &resource)
	return resource, err
}

// bookChapters supplies chapters for servers that omit them from a book response.
// A failed secondary request must not prevent access to the book itself.
func (c *BookCollection) bookChapters(ctx context.Context, id int) []Chapter {
	path := fmt.Sprintf("/api/stories/%d/scenes", id)
	req, err := c.transport.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil
	}
	resp, err := c.transport.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	data, _ := io.ReadAll(resp.Body)
	var envelope struct {
		Data []Chapter `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && len(envelope.Data) > 0 {
		return envelope.Data
	}
	var chapters []Chapter
	if json.Unmarshal(data, &chapters) != nil {
		return nil
	}
	return chapters
}

func bookUpdateBody(params UpdateBookParams) map[string]any {
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

	return body
}

// BookWritingSettings holds the writing choices used for prose generation.
type BookWritingSettings struct {
	POV                   string  `json:"pov,omitempty"`
	Tense                 string  `json:"tense,omitempty"`
	ProseStyle            *string `json:"prose_style,omitempty"`
	TargetWordCount       *int    `json:"target_word_count,omitempty"`
	FilterUsingStorySoFar bool    `json:"filter_using_story_so_far"`
}

// BookCollection owns books operations over the shared authenticated transport.
type BookCollection struct{ transport *Client }

// Books returns the books module for this client.
func (c *Client) Books() *BookCollection { return &BookCollection{transport: c} }

// BookDisplay contains the text used to display a book.
type BookDisplay struct{ Premise, Lore, Characters string }

// Display returns the text fields with their user-facing fallbacks.
func (b *Book) Display() BookDisplay {
	return BookDisplay{Premise: displayBookPremise(b), Lore: displayBookLore(b), Characters: displayBookCharacters(b)}
}

func importedBookTitle(filePath, title string) string {
	if title == "" {
		base := filepath.Base(filePath)
		title = strings.TrimSuffix(base, filepath.Ext(base))
		if title == "" {
			title = "Imported Book"
		}
	}
	return title
}
