package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func setupTestBookEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-book-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-book-token")

	return server, configPath, server.Client()
}

func TestBookHelp(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"book", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Manage books on Prosie.") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestBookUnknownSubcommand(t *testing.T) {
	cmd, _, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"book", "nonexistent"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown book command: nonexistent") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestBookList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":                1,
					"title":             "Solar Drift",
					"word_count":        12500,
					"target_word_count": 80000,
					"updated_at":        "2026-09-13T10:00:00Z",
				},
				{
					"id":         2,
					"title":      "Echoes of Stone",
					"word_count": 300,
					"updated_at": "2026-09-12T15:30:00Z",
				},
			},
		})
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("plain text tabular", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "list"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "TITLE") || !strings.Contains(stdout, "WORDS") || !strings.Contains(stdout, "TARGET") || !strings.Contains(stdout, "UPDATED") {
			t.Fatalf("missing table headers in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "Solar Drift") || !strings.Contains(stdout, "12500") || !strings.Contains(stdout, "80000") {
			t.Fatalf("missing Solar Drift row: %s", stdout)
		}
		if !strings.Contains(stdout, "Echoes of Stone") || !strings.Contains(stdout, "300") {
			t.Fatalf("missing Echoes of Stone row: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "list", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var books []client.Book
		if err := json.Unmarshal(out.Bytes(), &books); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if len(books) != 2 {
			t.Fatalf("expected 2 books, got %d", len(books))
		}
		if books[0].Title != "Solar Drift" || books[1].Title != "Echoes of Stone" {
			t.Fatalf("unexpected books json: %+v", books)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		emptyHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
		_, emptyCfg, emptyClient := setupTestBookEnv(t, emptyHandler)

		cmd, out, errOut := newTestRootCmd(emptyCfg, emptyClient)
		code := cmd.Execute([]string{"book", "list"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No books found.") {
			t.Fatalf("expected 'No books found.', got %s", out.String())
		}
	})
}

func TestBookShow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/stories/1":
			target := 80000
			percent := 15
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":                      1,
					"title":                   "The Starlight Gate",
					"story_so_far":            "A crew discovers a ruined jump gate.",
					"lore":                    "Built by ancient precursors.",
					"characters":              "Captain Vance, Pilot Kira",
					"word_count":              12000,
					"target_word_count":       target,
					"target_progress_percent": percent,
					"updated_at":              "2026-09-13T14:00:00Z",
				},
			})
		case "/api/stories/1/scenes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 101, "name": "Chapter 1: The Signal", "order": 0, "word_count": 5000},
					{"id": 102, "name": "Chapter 2: The Derelict", "order": 1, "word_count": 7000},
				},
			})
		case "/api/stories/404":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Book not found."})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "show"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "show", "abc"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "must be an integer") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "show", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "Title:       The Starlight Gate") {
			t.Fatalf("missing title: %s", stdout)
		}
		if !strings.Contains(stdout, "Words:       12000 / 80000 (15%)") {
			t.Fatalf("missing words: %s", stdout)
		}
		if !strings.Contains(stdout, "Premise:     A crew discovers a ruined jump gate.") {
			t.Fatalf("missing premise: %s", stdout)
		}
		if !strings.Contains(stdout, "Lore:        Built by ancient precursors.") {
			t.Fatalf("missing lore: %s", stdout)
		}
		if !strings.Contains(stdout, "Characters:  Captain Vance, Pilot Kira") {
			t.Fatalf("missing characters: %s", stdout)
		}
		if !strings.Contains(stdout, "Chapters (2):") || !strings.Contains(stdout, "Chapter 1: The Signal (5000 words)") {
			t.Fatalf("missing chapter list: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "show", "1", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var b map[string]any
		if err := json.Unmarshal(out.Bytes(), &b); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if b["title"] != "The Starlight Gate" || b["premise"] != "A crew discovers a ruined jump gate." {
			t.Fatalf("unexpected json content: %+v", b)
		}
		chapters, ok := b["chapters"].([]any)
		if !ok || len(chapters) != 2 {
			t.Fatalf("expected 2 chapters in json, got %+v", b["chapters"])
		}
	})

	t.Run("404 not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "show", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Book not found.") {
			t.Fatalf("expected error message in stderr, got: %s", errOut.String())
		}
	})
}

func TestBookCreate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
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
				"errors":  map[string]any{"title": []string{"The title field is required."}},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"filter_using_story_so_far": true,
				"chapters":                  []map[string]any{{"id": 101, "order": 0, "name": "Chapter 1"}},
				"id":                        42,
				"title":                     title,
				"story_so_far":              body["story_so_far"],
				"lore":                      body["lore"],
				"characters":                body["characters"],
				"target_word_count":         body["target_word_count"],
			},
		})
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing title flag", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "create"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--title is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful create plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"book", "create",
			"--title", "The Iron Core",
			"--premise", "Underground miners awaken something.",
			"--lore", "The crust has layers of forgotten tech.",
			"--characters", "Dax, Cora",
			"--target-words", "90000",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Created book \"The Iron Core\" (ID: 42).") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("all supplied notes survive creation", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "create", "--title", "The Iron Core", "--premise", "Underground miners awaken something.", "--lore", "The crust has layers of forgotten tech.", "--characters", "Dax, Cora", "--target-words", "90000", "--json"})
		if code != 0 || errOut.Len() != 0 {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
		var book map[string]any
		if err := json.Unmarshal(out.Bytes(), &book); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{"title": "The Iron Core", "premise": "Underground miners awaken something.", "story_so_far": "Underground miners awaken something.", "lore": "The crust has layers of forgotten tech.", "characters": "Dax, Cora", "target_word_count": float64(90000)}
		for field, want := range expected {
			if book[field] != want {
				t.Errorf("%s=%v; want=%v", field, book[field], want)
			}
		}
	})

	t.Run("successful create --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"book", "create",
			"--title", "The Iron Core",
			"--json",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var b map[string]any
		if err := json.Unmarshal(out.Bytes(), &b); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		chapters, ok := b["chapters"].([]any)
		if b["filter_using_story_so_far"] != true || !ok || len(chapters) != 1 {
			t.Fatalf("create must retain saved defaults and chapters: %+v", b)
		}
		if chapter, ok := chapters[0].(map[string]any); !ok || chapter["name"] != "Chapter 1" {
			t.Fatalf("missing initial chapter: %+v", chapters)
		}
		if b["id"] != float64(42) || b["title"] != "The Iron Core" {
			t.Fatalf("unexpected created book: %+v", b)
		}
	})
}

func TestBookUpdate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/7" || r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                7,
				"title":             body["title"],
				"story_so_far":      body["story_so_far"],
				"target_word_count": body["target_word_count"],
			},
		})
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "update"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful update plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"book", "update", "7",
			"--title", "Revised Edition",
			"--premise", "New exciting premise.",
			"--target-words", "85000",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Updated book 7 (\"Revised Edition\").") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("successful update --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"book", "update", "7",
			"--title", "Revised Edition",
			"--json",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var b map[string]any
		if err := json.Unmarshal(out.Bytes(), &b); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if b["id"] != float64(7) || b["title"] != "Revised Edition" {
			t.Fatalf("unexpected json: %+v", b)
		}
	})
}

func TestBookDuplicate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/12/duplicate" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":    13,
				"title": "Copy of Original Book",
			},
		})
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "duplicate"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("duplicate plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "duplicate", "12"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Duplicated book 12 to \"Copy of Original Book\" (ID: 13).") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("duplicate --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "duplicate", "12", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var b map[string]any
		if err := json.Unmarshal(out.Bytes(), &b); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if b["id"] != float64(13) || b["title"] != "Copy of Original Book" {
			t.Fatalf("unexpected json: %+v", b)
		}
	})
}

func TestBookDelete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stories/8" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "delete"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("delete cancelled via prompt", func(t *testing.T) {
		cmd, out, _ := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("n\n")
		code := cmd.Execute([]string{"book", "delete", "8"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d", code)
		}
		if !strings.Contains(out.String(), "Deletion cancelled.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete confirmed via prompt", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("y\n")
		code := cmd.Execute([]string{"book", "delete", "8"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted book 8.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with --yes flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "delete", "8", "--yes"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted book 8.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with -y flag and --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "delete", "8", "-y", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(8) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
	})
}

func TestBookExport(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/5/export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte("# Full Manuscript\n\nChapter 1 text.\n\nChapter 2 text."))
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "export"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("export to stdout", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "export", "5"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "# Full Manuscript") {
			t.Fatalf("missing exported manuscript: %s", out.String())
		}
	})

	t.Run("export to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "exported.md")

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "export", "5", "-o", outFile})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Exported book 5 to "+outFile) {
			t.Fatalf("unexpected output: %s", out.String())
		}

		data, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("failed to read exported file: %v", err)
		}
		if !strings.Contains(string(data), "# Full Manuscript") {
			t.Fatalf("file does not contain expected manuscript: %s", string(data))
		}
	})

	t.Run("export to file --json", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "exported.md")

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "export", "5", "--output", outFile, "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(5) || res["output"] != outFile {
			t.Fatalf("unexpected json: %+v", res)
		}
	})
}

func TestBookImport(t *testing.T) {
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "my-novel.docx")
	if err := os.WriteFile(docxPath, []byte("dummy-docx-content"), 0644); err != nil {
		t.Fatalf("failed to create test docx: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/import-docx" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseMultipartForm(10 << 20)
		title := r.FormValue("title")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         99,
				"title":      title,
				"word_count": 5000,
			},
			"scenes": []map[string]any{
				{"id": 1, "name": "Chapter 1", "order": 0, "word_count": 2500},
				{"id": 2, "name": "Chapter 2", "order": 1, "word_count": 2500},
			},
		})
	}
	_, cfgPath, httpClient := setupTestBookEnv(t, handler)

	t.Run("missing file arg", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "import"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "DOCX file path is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("file not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "import", "/path/to/missing.docx"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "file not found") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("import plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "import", docxPath, "--title", "Imported Novel"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Imported book \"Imported Novel\" (ID: 99) with 2 chapters.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("import --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"book", "import", docxPath, "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var b map[string]any
		if err := json.Unmarshal(out.Bytes(), &b); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if b["id"] != float64(99) || b["title"] != "my-novel" {
			t.Fatalf("unexpected json: %+v", b)
		}
	})
}
