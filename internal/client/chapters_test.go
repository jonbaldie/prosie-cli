package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListChapters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/scenes" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":         101,
					"story_id":   1,
					"order":      0,
					"name":       "Chapter 1: The Departure",
					"summary":    "The crew leaves Earth orbit.",
					"word_count": 3500,
					"created_at": "2026-09-13T10:00:00Z",
					"updated_at": "2026-09-13T10:30:00Z",
				},
				{
					"id":         102,
					"story_id":   1,
					"order":      1,
					"name":       "Chapter 2: The Void",
					"summary":    nil,
					"word_count": 4200,
					"created_at": "2026-09-13T11:00:00Z",
					"updated_at": "2026-09-13T11:45:00Z",
				},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	chapters, err := cli.Chapters().ListChapters(context.Background(), "1")
	if err != nil {
		t.Fatalf("ListChapters returned error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	if chapters[0].ID != 101 || chapters[0].Display().Title != "Chapter 1: The Departure" {
		t.Fatalf("unexpected chapter 0: %+v", chapters[0])
	}
	if chapters[0].Display().Summary != "The crew leaves Earth orbit." {
		t.Fatalf("expected summary, got %q", chapters[0].Display().Summary)
	}
	if chapters[1].Display().Summary != "-" {
		t.Fatalf("expected dash for nil summary, got %q", chapters[1].Display().Summary)
	}
}

func TestGetChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/scenes/101":
			summary := "The crew leaves Earth orbit."
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":         101,
					"story_id":   1,
					"order":      0,
					"name":       "Chapter 1: The Departure",
					"content":    "<p>The engines roared as the starship broke free.</p>",
					"summary":    summary,
					"word_count": 8,
				},
			})
		case "/api/scenes/999":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())

	t.Run("success", func(t *testing.T) {
		ch, err := cli.Chapters().GetChapter(context.Background(), "101")
		if err != nil {
			t.Fatalf("GetChapter returned error: %v", err)
		}
		if ch.ID != 101 || ch.Display().Title != "Chapter 1: The Departure" {
			t.Fatalf("unexpected chapter: %+v", ch)
		}
		if ch.Content != "<p>The engines roared as the starship broke free.</p>" {
			t.Fatalf("unexpected content: %s", ch.Content)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := cli.Chapters().GetChapter(context.Background(), "999")
		if err == nil {
			t.Fatal("expected 404 error, got nil")
		}
		apiErr, ok := err.(*ApiError)
		if !ok || apiErr.StatusCode != 404 {
			t.Fatalf("expected 404 ApiError, got %v", err)
		}
	})
}

func TestCreateChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/scenes" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		name, _ := body["name"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         105,
				"story_id":   1,
				"order":      2,
				"name":       name,
				"content":    body["content"],
				"summary":    body["summary"],
				"word_count": 10,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	title := "Chapter 3: First Contact"
	content := "A strange signal was detected."
	summary := "Signal received."

	ch, err := cli.Chapters().CreateChapter(context.Background(), "1", CreateChapterParams{
		Title:   &title,
		Content: &content,
		Summary: &summary,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch.ID != 105 || ch.Display().Title != "Chapter 3: First Contact" {
		t.Fatalf("unexpected chapter: %+v", ch)
	}
	if ch.Content != "A strange signal was detected." {
		t.Fatalf("unexpected content: %q", ch.Content)
	}
}

func TestUpdateChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/scenes/101" || r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         101,
				"story_id":   1,
				"order":      0,
				"name":       body["name"],
				"content":    body["content"],
				"summary":    body["summary"],
				"word_count": 12,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	newTitle := "Revised Title"
	newContent := "Updated prose content."

	ch, err := cli.Chapters().UpdateChapter(context.Background(), "101", UpdateChapterParams{
		Title:   &newTitle,
		Content: &newContent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch.Display().Title != "Revised Title" {
		t.Fatalf("expected Revised Title, got %s", ch.Display().Title)
	}
	if ch.Content != "Updated prose content." {
		t.Fatalf("expected updated prose, got %s", ch.Content)
	}
}

func TestReorderChapters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/scenes/reorder" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		rawIDs, ok := body["scene_ids"].([]any)
		if !ok || len(rawIDs) != 2 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Invalid scene_ids"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 102, "story_id": 1, "order": 0, "name": "Chapter 2"},
				{"id": 101, "story_id": 1, "order": 1, "name": "Chapter 1"},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	chapters, err := cli.Chapters().ReorderChapters(context.Background(), "1", []string{"102", "101"})
	if err != nil {
		t.Fatalf("unexpected error reordering chapters: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].ID != 102 || chapters[1].ID != 101 {
		t.Fatalf("unexpected order: %+v", chapters)
	}
}

func TestDeleteChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/scenes/101" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/api/scenes/999" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "A story must retain at least one chapter.",
				"errors": map[string][]string{
					"scene": {"A story must retain at least one chapter."},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())

	t.Run("successful delete", func(t *testing.T) {
		if err := cli.Chapters().DeleteChapter(context.Background(), "101"); err != nil {
			t.Fatalf("unexpected error deleting chapter: %v", err)
		}
	})

	t.Run("last scene error", func(t *testing.T) {
		err := cli.Chapters().DeleteChapter(context.Background(), "999")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "A story must retain at least one chapter.") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestExportChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/scenes/101/export" {
			http.NotFound(w, r)
			return
		}
		format := r.URL.Query().Get("format")
		if format == "markdown" || format == "" {
			w.Header().Set("Content-Type", "text/markdown")
			_, _ = w.Write([]byte("# Chapter 1\n\nIt was a dark and stormy night."))
			return
		}
		http.Error(w, "invalid format", http.StatusBadRequest)
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	data, err := cli.Chapters().ExportChapter(context.Background(), "101", "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "# Chapter 1\n\nIt was a dark and stormy night." {
		t.Fatalf("unexpected exported data: %s", string(data))
	}
}
