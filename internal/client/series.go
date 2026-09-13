package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Series represents a book series or narrative universe in Prosie.
type Series struct {
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	Title        string       `json:"title,omitempty"`
	Description  *string      `json:"description,omitempty"`
	StoryIDs     []int        `json:"story_ids,omitempty"`
	Books        []Book       `json:"books,omitempty"`
	Stories      []Book       `json:"stories,omitempty"`
	CodexEntries []CodexEntry `json:"codex_entries,omitempty"`
	CreatedAt    string       `json:"created_at,omitempty"`
	UpdatedAt    string       `json:"updated_at,omitempty"`
}

// DisplayTitle returns the series title, name, or fallback.
func (s *Series) DisplayTitle() string {
	if s.Title != "" {
		return s.Title
	}
	if s.Name != "" {
		return s.Name
	}
	return fmt.Sprintf("Series %d", s.ID)
}

// DisplayDescription returns the series description or a dash.
func (s *Series) DisplayDescription() string {
	if s.Description != nil && strings.TrimSpace(*s.Description) != "" {
		return *s.Description
	}
	return "-"
}

func (s *Series) normalize() {
	if s.Title == "" && s.Name != "" {
		s.Title = s.Name
	}
	if s.Name == "" && s.Title != "" {
		s.Name = s.Title
	}
	if len(s.Books) == 0 && len(s.Stories) > 0 {
		s.Books = s.Stories
	}
	if len(s.Stories) == 0 && len(s.Books) > 0 {
		s.Stories = s.Books
	}
	if s.Books == nil {
		s.Books = []Book{}
	}
	if s.CodexEntries == nil {
		s.CodexEntries = []CodexEntry{}
	}
}

// CreateSeriesParams holds input parameters for creating a series.
type CreateSeriesParams struct {
	Name        string  `json:"name,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdateSeriesParams holds input parameters for updating a series.
type UpdateSeriesParams struct {
	Name        *string `json:"name,omitempty"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ListSeries fetches all series owned by the authenticated user and populates member books.
func (c *Client) ListSeries(ctx context.Context) ([]Series, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/series", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/series failed: %w", err)
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
		Data []Series `json:"data"`
	}
	var seriesList []Series
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		seriesList = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &seriesList); err != nil {
		return nil, fmt.Errorf("failed to decode series response: %w", err)
	}

	for i := range seriesList {
		seriesList[i].normalize()
	}

	// Resolve member books if missing
	needBooks := false
	for _, s := range seriesList {
		if len(s.Books) == 0 && (len(s.StoryIDs) > 0 || len(seriesList) > 0) {
			needBooks = true
			break
		}
	}

	if needBooks {
		books, err := c.ListBooks(ctx)
		if err == nil && len(books) > 0 {
			bookMap := make(map[int]Book)
			for _, b := range books {
				bookMap[b.ID] = b
			}
			for i := range seriesList {
				if len(seriesList[i].Books) == 0 {
					if len(seriesList[i].StoryIDs) > 0 {
						for _, bid := range seriesList[i].StoryIDs {
							if b, ok := bookMap[bid]; ok {
								seriesList[i].Books = append(seriesList[i].Books, b)
							}
						}
					} else {
						for _, b := range books {
							if b.SeriesID != nil && *b.SeriesID == seriesList[i].ID {
								seriesList[i].Books = append(seriesList[i].Books, b)
							}
						}
					}
				}
				seriesList[i].normalize()
			}
		}
	}

	return seriesList, nil
}

// GetSeries fetches a single series by ID with member books and shared codex entries.
func (c *Client) GetSeries(ctx context.Context, id int) (*Series, error) {
	path := fmt.Sprintf("/api/series/%d", id)
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
		Data Series `json:"data"`
	}
	var s Series
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		s = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &s); err != nil {
		return nil, fmt.Errorf("failed to decode series response: %w", err)
	}

	s.normalize()

	// Resolve member books if not populated
	if len(s.Books) == 0 {
		books, err := c.ListBooks(ctx)
		if err == nil {
			if len(s.StoryIDs) > 0 {
				bookMap := make(map[int]Book)
				for _, b := range books {
					bookMap[b.ID] = b
				}
				for _, bid := range s.StoryIDs {
					if b, ok := bookMap[bid]; ok {
						s.Books = append(s.Books, b)
					}
				}
			} else {
				for _, b := range books {
					if b.SeriesID != nil && *b.SeriesID == s.ID {
						s.Books = append(s.Books, b)
					}
				}
			}
		}
	}

	// Resolve shared codex entries if not populated
	if len(s.CodexEntries) == 0 {
		codexEntries, err := c.ListSeriesCodexEntries(ctx, id)
		if err == nil {
			s.CodexEntries = codexEntries
		}
	}

	s.normalize()
	return &s, nil
}

// CreateSeries creates a new book series.
func (c *Client) CreateSeries(ctx context.Context, params CreateSeriesParams) (*Series, error) {
	name := params.Name
	if name == "" {
		name = params.Title
	}

	body := map[string]any{
		"name":  name,
		"title": name,
	}
	if params.Description != nil {
		body["description"] = *params.Description
	}

	req, err := c.NewRequest(ctx, http.MethodPost, "/api/series", body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/series failed: %w", err)
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
		Data Series `json:"data"`
	}
	var s Series
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		s = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &s); err != nil {
		return nil, fmt.Errorf("failed to decode create series response: %w", err)
	}

	s.normalize()
	return &s, nil
}

// UpdateSeries updates an existing series.
func (c *Client) UpdateSeries(ctx context.Context, id int, params UpdateSeriesParams) (*Series, error) {
	body := map[string]any{}
	if params.Name != nil {
		body["name"] = *params.Name
		body["title"] = *params.Name
	} else if params.Title != nil {
		body["name"] = *params.Title
		body["title"] = *params.Title
	}
	if params.Description != nil {
		body["description"] = *params.Description
	}

	path := fmt.Sprintf("/api/series/%d", id)
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
		Data Series `json:"data"`
	}
	var s Series
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		s = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &s); err != nil {
		return nil, fmt.Errorf("failed to decode update series response: %w", err)
	}

	s.normalize()
	return &s, nil
}

// DeleteSeries deletes a series by ID.
func (c *Client) DeleteSeries(ctx context.Context, id int) error {
	path := fmt.Sprintf("/api/series/%d", id)
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

// AttachSeriesBook attaches a book to a series.
func (c *Client) AttachSeriesBook(ctx context.Context, seriesID, bookID int) (*Book, error) {
	path := fmt.Sprintf("/api/series/%d/stories/%d", seriesID, bookID)
	req, err := c.NewRequest(ctx, http.MethodPut, path, nil)
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
		return nil, fmt.Errorf("failed to decode attach book response: %w", err)
	}

	book.normalize()
	return &book, nil
}

// DetachSeriesBook removes a book from a series.
func (c *Client) DetachSeriesBook(ctx context.Context, seriesID, bookID int) error {
	path := fmt.Sprintf("/api/series/%d/stories/%d", seriesID, bookID)
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
