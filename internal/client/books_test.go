package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListBooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories" {
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
					"id":                1,
					"title":             "The First Story",
					"story_so_far":      "Once upon a time in space.",
					"word_count":        5000,
					"target_word_count": 80000,
					"updated_at":        "2026-09-13T12:00:00Z",
				},
				{
					"id":         2,
					"title":      "Second Story",
					"word_count": 1200,
					"updated_at": "2026-09-13T13:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	books, err := cli.Books().ListBooks(context.Background())
	if err != nil {
		t.Fatalf("ListBooks returned error: %v", err)
	}

	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}

	if books[0].ID != 1 || books[0].Title != "The First Story" {
		t.Fatalf("unexpected book 0: %+v", books[0])
	}
	if books[0].Display().Premise != "Once upon a time in space." {
		t.Fatalf("expected premise normalized, got %q", books[0].Display().Premise)
	}
	if books[1].Display().Premise != "-" {
		t.Fatalf("expected dash for empty premise, got %q", books[1].Display().Premise)
	}
}

func TestGetBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/stories/1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":                1,
					"title":             "Book One",
					"lore":              "A secret galaxy.",
					"characters":        "Hero and Villain.",
					"story_so_far":      "The journey begins.",
					"word_count":        10000,
					"target_word_count": 50000,
				},
			})
		case "/api/stories/1/scenes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 10, "story_id": 1, "order": 0, "name": "Chapter 1: The Call", "word_count": 4000},
					{"id": 11, "story_id": 1, "order": 1, "name": "Chapter 2: The Crossing", "word_count": 6000},
				},
			})
		case "/api/stories/99":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Story not found."})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())

	t.Run("success with populated chapters", func(t *testing.T) {
		book, err := cli.Books().GetBook(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetBook returned error: %v", err)
		}
		if book.ID != 1 || book.Title != "Book One" {
			t.Fatalf("unexpected book: %+v", book)
		}
		if book.Display().Lore != "A secret galaxy." {
			t.Fatalf("unexpected lore: %q", book.Display().Lore)
		}
		if book.Display().Characters != "Hero and Villain." {
			t.Fatalf("unexpected characters: %q", book.Display().Characters)
		}
		if book.Display().Premise != "The journey begins." {
			t.Fatalf("unexpected premise: %q", book.Display().Premise)
		}
		if len(book.Chapters) != 2 {
			t.Fatalf("expected 2 chapters, got %d", len(book.Chapters))
		}
		if book.Chapters[0].Display().Title != "Chapter 1: The Call" {
			t.Fatalf("unexpected chapter title: %s", book.Chapters[0].Display().Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := cli.Books().GetBook(context.Background(), 99)
		if err == nil {
			t.Fatal("expected 404 error, got nil")
		}
		apiErr, ok := err.(*ApiError)
		if !ok || apiErr.StatusCode != 404 {
			t.Fatalf("expected ApiError with 404, got %v", err)
		}
	})
}

func TestCreateBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		title, _ := body["title"].(string)
		if title == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "The title field is required.",
				"errors": map[string]any{
					"title": []string{"The title field is required."},
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                5,
				"title":             title,
				"lore":              body["lore"],
				"characters":        body["characters"],
				"story_so_far":      body["story_so_far"],
				"target_word_count": body["target_word_count"],
				"word_count":        0,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())

	t.Run("successful create", func(t *testing.T) {
		premise := "Epic space adventure."
		lore := "Deep space outpost."
		chars := "Captain John."
		target := 75000

		book, err := cli.Books().CreateBook(context.Background(), CreateBookParams{
			Title:           "Starlight",
			Premise:         &premise,
			Lore:            &lore,
			Characters:      &chars,
			TargetWordCount: &target,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if book.ID != 5 || book.Title != "Starlight" {
			t.Fatalf("unexpected book: %+v", book)
		}
		if book.Display().Premise != premise {
			t.Fatalf("unexpected premise: %q", book.Display().Premise)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		_, err := cli.Books().CreateBook(context.Background(), CreateBookParams{
			Title: "",
		})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		apiErr, ok := err.(*ApiError)
		if !ok || apiErr.StatusCode != 422 {
			t.Fatalf("expected 422 ApiError, got %v", err)
		}
		if !strings.Contains(err.Error(), "title: The title field is required.") {
			t.Fatalf("expected error message to include field validation details, got: %s", err.Error())
		}
	})
}

func TestUpdateBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1" || r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                1,
				"title":             body["title"],
				"story_so_far":      body["story_so_far"],
				"target_word_count": body["target_word_count"],
				"word_count":        1000,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	newTitle := "Updated Title"
	newPremise := "Updated Premise"
	target := 60000

	book, err := cli.Books().UpdateBook(context.Background(), 1, UpdateBookParams{
		Title:           &newTitle,
		Premise:         &newPremise,
		TargetWordCount: &target,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Title != "Updated Title" {
		t.Fatalf("expected Updated Title, got %s", book.Title)
	}
	if book.Display().Premise != "Updated Premise" {
		t.Fatalf("expected Updated Premise, got %s", book.Display().Premise)
	}
}

func TestDeleteBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stories/1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	if err := cli.Books().DeleteBook(context.Background(), 1); err != nil {
		t.Fatalf("expected successful delete, got error: %v", err)
	}

	if err := cli.Books().DeleteBook(context.Background(), 2); err == nil {
		t.Fatal("expected error deleting non-existent book, got nil")
	}
}

func TestDuplicateBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/duplicate" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":    2,
				"title": "Copy of Original",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	copyBook, err := cli.Books().DuplicateBook(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if copyBook.ID != 2 || copyBook.Title != "Copy of Original" {
		t.Fatalf("unexpected duplicate book: %+v", copyBook)
	}
	if copyBook.Chapters != nil {
		t.Fatalf("expected nil chapters on duplicated book, got: %+v", copyBook.Chapters)
	}
	data, err := json.Marshal(copyBook)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if strings.Contains(string(data), `"chapters"`) {
		t.Fatalf("expected no chapters field in duplicate book JSON, got: %s", string(data))
	}
}

func TestBookChaptersJSONSerialization(t *testing.T) {
	t.Run("omits chapters when unloaded", func(t *testing.T) {
		b := Book{ID: 1, Title: "Test"}
		b.normalize()
		if b.Chapters != nil {
			t.Fatalf("expected nil Chapters, got %+v", b.Chapters)
		}
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		if strings.Contains(string(data), `"chapters"`) {
			t.Fatalf("expected chapters to be omitted from JSON, got: %s", string(data))
		}
	})

	t.Run("includes chapters when loaded", func(t *testing.T) {
		b := Book{
			ID:       1,
			Title:    "Test",
			Chapters: []Chapter{{ID: 10, Title: "Chapter 1"}},
		}
		b.normalize()
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		if !strings.Contains(string(data), `"chapters"`) {
			t.Fatalf("expected chapters to be present in JSON, got: %s", string(data))
		}
	})
}

func TestExportStory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/export" {
			http.NotFound(w, r)
			return
		}
		format := r.URL.Query().Get("format")
		if format == "markdown" {
			w.Header().Set("Content-Type", "text/markdown")
			_, _ = w.Write([]byte("# Novel Title\n\nChapter prose here."))
			return
		}
		http.Error(w, "invalid format", http.StatusBadRequest)
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	content, err := cli.Books().ExportStory(context.Background(), 1, "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "# Novel Title\n\nChapter prose here." {
		t.Fatalf("unexpected content: %s", string(content))
	}
}

func TestImportDocx(t *testing.T) {
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "novel.docx")
	if err := os.WriteFile(docxPath, []byte("fake-docx-binary-data"), 0644); err != nil {
		t.Fatalf("failed to write test docx: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/import-docx" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		file, header, err := r.FormFile("document")
		if err != nil {
			t.Errorf("missing document file in multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileBytes, _ := io.ReadAll(file)
		if string(fileBytes) != "fake-docx-binary-data" {
			t.Errorf("unexpected file content: %s", string(fileBytes))
		}
		if header.Filename != "novel.docx" {
			t.Errorf("unexpected filename: %s", header.Filename)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         10,
				"title":      title,
				"word_count": 2500,
			},
			"scenes": []map[string]any{
				{"id": 101, "name": "Chapter 1", "order": 0, "word_count": 1200},
				{"id": 102, "name": "Chapter 2", "order": 1, "word_count": 1300},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	book, err := cli.Books().ImportDocx(context.Background(), docxPath, "Custom Title")
	if err != nil {
		t.Fatalf("unexpected error importing docx: %v", err)
	}

	if book.ID != 10 || book.Title != "Custom Title" {
		t.Fatalf("unexpected book: %+v", book)
	}
	if len(book.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(book.Chapters))
	}
}
