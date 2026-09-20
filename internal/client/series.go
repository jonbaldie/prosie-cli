package client

import (
	"context"
	"encoding/json"
	"fmt"
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

func displaySeriesTitle(s *Series) string {
	if s.Title != "" {
		return s.Title
	}
	if s.Name != "" {
		return s.Name
	}
	return fmt.Sprintf("Series %d", s.ID)
}

func displaySeriesDescription(s *Series) string {
	if s.Description != nil && strings.TrimSpace(*s.Description) != "" {
		return *s.Description
	}
	return "-"
}

// Unloaded relations stay nil so that omitempty leaves them out of JSON output.
func (s *Series) normalize() {
	normalizeSeriesTitles(s)
	normalizeSeriesBooks(s)
}

func normalizeSeriesTitles(s *Series) {
	if s.Title == "" {
		s.Title = s.Name
	}
	if s.Name == "" {
		s.Name = s.Title
	}
}

func normalizeSeriesBooks(s *Series) {
	if len(s.Books) == 0 && len(s.Stories) > 0 {
		s.Books = s.Stories
	}
	if len(s.Stories) == 0 && len(s.Books) > 0 {
		s.Stories = s.Books
	}
	for i := range s.Books {
		s.Books[i].normalize()
	}
	for i := range s.Stories {
		s.Stories[i].normalize()
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
func (c *SeriesCollection) ListSeries(ctx context.Context) ([]Series, error) {
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, "/api/series", nil)
	if err != nil {
		return nil, err
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

	c.populateSeriesBooks(ctx, seriesList)

	return seriesList, nil
}

// GetSeries fetches a single series by ID with member books and shared codex entries.
func (c *SeriesCollection) GetSeries(ctx context.Context, id int) (*Series, error) {
	path := fmt.Sprintf("/api/series/%d", id)
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var s Series
	s, err = decodeSeries(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode series response: %w", err)
	}

	s.normalize()

	if len(s.Books) == 0 {
		books, err := c.transport.Books().ListBooks(ctx)
		if err == nil {
			s.Books = seriesBooks(s, books)
		}
	}

	// Resolve shared codex entries if not populated
	if len(s.CodexEntries) == 0 {
		codexEntries, err := c.transport.Codex().ListSeriesCodexEntries(ctx, id)
		if err == nil {
			s.CodexEntries = codexEntries
		}
	}

	s.normalize()
	return &s, nil
}

// CreateSeries creates a new book series.
func (c *SeriesCollection) CreateSeries(ctx context.Context, params CreateSeriesParams) (*Series, error) {
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

	bodyBytes, err := c.transport.request(ctx, http.MethodPost, "/api/series", body)
	if err != nil {
		return nil, err
	}

	var s Series
	s, err = decodeSeries(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode create series response: %w", err)
	}

	s.normalize()
	return &s, nil
}

// UpdateSeries updates an existing series.
func (c *SeriesCollection) UpdateSeries(ctx context.Context, id int, params UpdateSeriesParams) (*Series, error) {
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
	bodyBytes, err := c.transport.request(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var s Series
	s, err = decodeSeries(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode update series response: %w", err)
	}

	s.normalize()
	return &s, nil
}

// DeleteSeries deletes a series by ID.
func (c *SeriesCollection) DeleteSeries(ctx context.Context, id int) error {
	path := fmt.Sprintf("/api/series/%d", id)
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

// AttachSeriesBook attaches a book to a series.
func (c *SeriesCollection) AttachSeriesBook(ctx context.Context, seriesID, bookID int) (*Book, error) {
	path := fmt.Sprintf("/api/series/%d/stories/%d", seriesID, bookID)
	bodyBytes, err := c.transport.request(ctx, http.MethodPut, path, nil)
	if err != nil {
		return nil, err
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
func (c *SeriesCollection) DetachSeriesBook(ctx context.Context, seriesID, bookID int) error {
	path := fmt.Sprintf("/api/series/%d/stories/%d", seriesID, bookID)
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

func decodeSeries(data []byte) (Series, error) {
	var envelope struct {
		Data Series `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Data.ID != 0 {
		return envelope.Data, nil
	}
	var resource Series
	err := json.Unmarshal(data, &resource)
	return resource, err
}

func (c *SeriesCollection) populateSeriesBooks(ctx context.Context, series []Series) {
	if !needsSeriesBooks(series) {
		return
	}
	books, err := c.transport.Books().ListBooks(ctx)
	if err != nil || len(books) == 0 {
		return
	}
	for i := range series {
		if len(series[i].Books) == 0 {
			series[i].Books = seriesBooks(series[i], books)
		}
		series[i].normalize()
	}
}

func needsSeriesBooks(series []Series) bool {
	for _, s := range series {
		if len(s.Books) == 0 {
			return true
		}
	}
	return false
}

// Explicit story IDs determine order; otherwise membership comes from each book.
func seriesBooks(series Series, books []Book) []Book {
	members := series.Books
	if len(series.StoryIDs) == 0 {
		for _, book := range books {
			if book.SeriesID != nil && *book.SeriesID == series.ID {
				members = append(members, book)
			}
		}
		return members
	}
	byID := make(map[int]Book)
	for _, book := range books {
		byID[book.ID] = book
	}
	for _, id := range series.StoryIDs {
		if book, ok := byID[id]; ok {
			members = append(members, book)
		}
	}
	return members
}

// SeriesCollection owns series operations over the shared authenticated transport.
type SeriesCollection struct{ transport *Client }

// Series returns the series module for this client.
func (c *Client) Series() *SeriesCollection { return &SeriesCollection{transport: c} }

// SeriesDisplay contains the text used to display a series.
type SeriesDisplay struct{ Title, Description string }

// Display returns the text fields with their user-facing fallbacks.
func (s *Series) Display() SeriesDisplay {
	return SeriesDisplay{Title: displaySeriesTitle(s), Description: displaySeriesDescription(s)}
}
