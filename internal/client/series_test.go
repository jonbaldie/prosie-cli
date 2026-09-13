package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/api/series":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":          1,
						"name":        "The Solar Cycle",
						"description": "Epic space opera series",
						"story_ids":   []int{10, 20},
						"updated_at":  "2026-09-13T12:00:00Z",
					},
					{
						"id":          2,
						"name":        "Voidborne",
						"description": "Cyberpunk saga",
						"story_ids":   []int{},
						"updated_at":  "2026-09-13T13:00:00Z",
					},
				},
			})
		case "/api/stories":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         10,
						"title":      "Solar Drift",
						"word_count": 12500,
						"series_id":  1,
					},
					{
						"id":         20,
						"title":      "Solar Eclipse",
						"word_count": 23000,
						"series_id":  1,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	seriesList, err := cli.ListSeries(context.Background())
	if err != nil {
		t.Fatalf("ListSeries returned error: %v", err)
	}

	if len(seriesList) != 2 {
		t.Fatalf("expected 2 series, got %d", len(seriesList))
	}

	if seriesList[0].ID != 1 || seriesList[0].Title != "The Solar Cycle" {
		t.Fatalf("unexpected series[0]: %+v", seriesList[0])
	}
	if len(seriesList[0].Books) != 2 {
		t.Fatalf("expected 2 member books in series[0], got %d", len(seriesList[0].Books))
	}
	if seriesList[0].Books[0].Title != "Solar Drift" {
		t.Fatalf("expected member book title 'Solar Drift', got %q", seriesList[0].Books[0].Title)
	}
	if seriesList[1].ID != 2 || seriesList[1].Title != "Voidborne" {
		t.Fatalf("unexpected series[1]: %+v", seriesList[1])
	}
}

func TestGetSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/series/1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":          1,
					"name":        "The Solar Cycle",
					"description": "Epic space opera",
					"story_ids":   []int{10},
					"updated_at":  "2026-09-13T14:00:00Z",
				},
			})
		case "/api/stories":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         10,
						"title":      "Solar Drift",
						"word_count": 12500,
						"series_id":  1,
					},
				},
			})
		case "/api/series/1/codex-entries":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":       100,
						"name":     "Helios Drive",
						"category": "lore",
						"content":  "Faster-than-light jump system.",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	s, err := cli.GetSeries(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetSeries returned error: %v", err)
	}

	if s.ID != 1 || s.Title != "The Solar Cycle" {
		t.Fatalf("unexpected series: %+v", s)
	}
	if len(s.Books) != 1 || s.Books[0].Title != "Solar Drift" {
		t.Fatalf("expected 1 book 'Solar Drift', got %+v", s.Books)
	}
	if len(s.CodexEntries) != 1 || s.CodexEntries[0].Name != "Helios Drive" {
		t.Fatalf("expected 1 codex entry 'Helios Drive', got %+v", s.CodexEntries)
	}
}

func TestGetSeriesNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "Series not found",
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	_, err := cli.GetSeries(context.Background(), 999)
	if err == nil {
		t.Fatalf("expected error for non-existent series, got nil")
	}
	apiErr, ok := err.(*ApiError)
	if !ok || apiErr.StatusCode != 404 {
		t.Fatalf("expected 404 ApiError, got %v", err)
	}
}

func TestCreateSeries(t *testing.T) {
	desc := "A sweeping space fantasy"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/series" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["name"] != "New Series" && payload["title"] != "New Series" {
			t.Errorf("unexpected payload: %+v", payload)
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":          5,
				"name":        "New Series",
				"description": desc,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	s, err := cli.CreateSeries(context.Background(), CreateSeriesParams{
		Title:       "New Series",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateSeries returned error: %v", err)
	}
	if s.ID != 5 || s.Title != "New Series" {
		t.Fatalf("unexpected created series: %+v", s)
	}
}

func TestUpdateSeries(t *testing.T) {
	newTitle := "Updated Title"
	newDesc := "Updated Description"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/series/5" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":          5,
				"name":        newTitle,
				"description": newDesc,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	s, err := cli.UpdateSeries(context.Background(), 5, UpdateSeriesParams{
		Title:       &newTitle,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateSeries returned error: %v", err)
	}
	if s.ID != 5 || s.Title != newTitle || *s.Description != newDesc {
		t.Fatalf("unexpected updated series: %+v", s)
	}
}

func TestDeleteSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/series/5" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	err := cli.DeleteSeries(context.Background(), 5)
	if err != nil {
		t.Fatalf("DeleteSeries returned error: %v", err)
	}
}

func TestAttachSeriesBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/series/1/stories/10" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":        10,
				"title":     "Solar Drift",
				"series_id": 1,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	book, err := cli.AttachSeriesBook(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("AttachSeriesBook returned error: %v", err)
	}
	if book.ID != 10 || book.SeriesID == nil || *book.SeriesID != 1 {
		t.Fatalf("unexpected book: %+v", book)
	}
}

func TestDetachSeriesBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/series/1/stories/10" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	err := cli.DetachSeriesBook(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("DetachSeriesBook returned error: %v", err)
	}
}
