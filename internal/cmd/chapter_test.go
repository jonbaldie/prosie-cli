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

func setupTestChapterEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-chapter-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-chapter-token")

	return server, configPath, server.Client()
}

func TestChapterHelp(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"chapter", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Manage chapters on Prosie.") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestChapterUnknownSubcommand(t *testing.T) {
	cmd, _, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"chapter", "unknown"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown chapter command: unknown") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestChapterList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/stories/1/scenes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         101,
						"story_id":   1,
						"order":      0,
						"name":       "Prologue",
						"summary":    "The origin of the signal.",
						"word_count": 1500,
					},
					{
						"id":         102,
						"story_id":   1,
						"order":      1,
						"name":       "The Signal",
						"summary":    nil,
						"word_count": 3200,
					},
				},
			})
		case "/api/stories/404/scenes":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Story not found."})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing book id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "list"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text tabular", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "list", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "INDEX") || !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "TITLE") || !strings.Contains(stdout, "WORDS") || !strings.Contains(stdout, "SUMMARY") {
			t.Fatalf("missing table headers in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "101") || !strings.Contains(stdout, "Prologue") || !strings.Contains(stdout, "1500") || !strings.Contains(stdout, "The origin of the signal.") {
			t.Fatalf("missing Prologue row: %s", stdout)
		}
		if !strings.Contains(stdout, "102") || !strings.Contains(stdout, "The Signal") || !strings.Contains(stdout, "3200") || !strings.Contains(stdout, "-") {
			t.Fatalf("missing The Signal row: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "list", "1", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var chapters []client.Chapter
		if err := json.Unmarshal(out.Bytes(), &chapters); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if len(chapters) != 2 {
			t.Fatalf("expected 2 chapters, got %d", len(chapters))
		}
		if chapters[0].ID != 101 || chapters[1].ID != 102 {
			t.Fatalf("unexpected chapters: %+v", chapters)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		emptyHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
		_, emptyCfg, emptyClient := setupTestChapterEnv(t, emptyHandler)

		cmd, out, errOut := newTestRootCmd(emptyCfg, emptyClient)
		code := cmd.Execute([]string{"chapter", "list", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No chapters found.") {
			t.Fatalf("expected 'No chapters found.', got %s", out.String())
		}
	})

	t.Run("404 story not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "list", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Story not found.") {
			t.Fatalf("expected 404 message in stderr: %s", errOut.String())
		}
	})
}

func TestChapterShow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/scenes/101":
			summary := "A brief summary."
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":         101,
					"story_id":   1,
					"order":      0,
					"name":       "Chapter 1",
					"content":    "The vessel floated silently through the debris field.",
					"summary":    summary,
					"word_count": 8,
				},
			})
		case "/api/scenes/102":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":         102,
					"story_id":   1,
					"order":      1,
					"name":       "Empty Chapter",
					"content":    "",
					"word_count": 0,
				},
			})
		case "/api/scenes/404":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "show"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text prose", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "show", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The vessel floated silently through the debris field.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("empty content shows message", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "show", "102"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No content.") {
			t.Fatalf("expected 'No content.', got %s", out.String())
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "show", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var ch client.Chapter
		if err := json.Unmarshal(out.Bytes(), &ch); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if ch.ID != 101 || ch.Content != "The vessel floated silently through the debris field." {
			t.Fatalf("unexpected chapter: %+v", ch)
		}
	})

	t.Run("404 not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "show", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Scene not found.") {
			t.Fatalf("expected 404 message: %s", errOut.String())
		}
	})

	t.Run("bare dash requests id dash", func(t *testing.T) {
		var capturedPath string
		dashHandler := func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":      1,
					"content": "Prose content for dash.",
				},
			})
		}
		_, dashCfg, dashClient := setupTestChapterEnv(t, dashHandler)

		cmd, out, errOut := newTestRootCmd(dashCfg, dashClient)
		code := cmd.Execute([]string{"chapter", "show", "-"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if capturedPath != "/api/scenes/-" {
			t.Fatalf("expected path /api/scenes/-, got %s", capturedPath)
		}
		if !strings.Contains(out.String(), "Prose content for dash.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})
}

func TestChapterCreate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/scenes" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		name, _ := body["name"].(string)
		if name == "" {
			name = "Chapter 1"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         105,
				"story_id":   1,
				"order":      0,
				"name":       name,
				"content":    body["content"],
				"summary":    body["summary"],
				"word_count": 15,
			},
		})
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing book id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "create"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("create with flags plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "create", "1",
			"--title", "Arrival at Station",
			"--content", "Docking clamps engaged with a dull clang.",
			"--summary", "Docking successful.",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Created chapter \"Arrival at Station\" (ID: 105).") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("create with --file flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "chapter.txt")
		if err := os.WriteFile(filePath, []byte("Content from file."), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "create", "1",
			"--title", "From File",
			"--file", filePath,
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Created chapter \"From File\" (ID: 105).") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("create with missing file", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "create", "1",
			"--file", "/path/to/nonexistent/file.txt",
		})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "error reading file") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("create with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "create", "1",
			"--title", "JSON Chapter",
			"--json",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var ch client.Chapter
		if err := json.Unmarshal(out.Bytes(), &ch); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if ch.ID != 105 || ch.Display().Title != "JSON Chapter" {
			t.Fatalf("unexpected chapter: %+v", ch)
		}
	})
}

func TestChapterUpdate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
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
				"word_count": 25,
			},
		})
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "update"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful update plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "update", "101",
			"--title", "Revised Scene Title",
			"--summary", "A new summary.",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Updated chapter 101 (\"Revised Scene Title\").") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("update with --file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "revised.txt")
		if err := os.WriteFile(filePath, []byte("Fresh prose from file."), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "update", "101",
			"--title", "File Update",
			"--file", filePath,
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Updated chapter 101 (\"File Update\").") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("update with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"chapter", "update", "101",
			"--title", "JSON Title",
			"--json",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var ch client.Chapter
		if err := json.Unmarshal(out.Bytes(), &ch); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if ch.ID != 101 || ch.Display().Title != "JSON Title" {
			t.Fatalf("unexpected chapter: %+v", ch)
		}
	})
}

func TestChapterReorder(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/scenes/reorder" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 103, "order": 0, "name": "Chapter 3"},
				{"id": 101, "order": 1, "name": "Chapter 1"},
				{"id": 102, "order": 2, "name": "Chapter 2"},
			},
		})
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing arguments", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "reorder"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID and chapter order are required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("reorder comma-separated", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "reorder", "1", "103,101,102"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Reordered 3 chapters for book 1.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("reorder space-separated", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "reorder", "1", "103", "101", "102"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Reordered 3 chapters for book 1.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("reorder --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "reorder", "1", "103,101,102", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var chapters []client.Chapter
		if err := json.Unmarshal(out.Bytes(), &chapters); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if len(chapters) != 3 || chapters[0].ID != 103 {
			t.Fatalf("unexpected chapters: %+v", chapters)
		}
	})

	t.Run("bare dash preserves book id and chapter order", func(t *testing.T) {
		var capturedPath string
		var capturedBody map[string]any

		dashHandler := func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 1, "order": 0, "name": "Chapter 1"},
					{"id": 2, "order": 1, "name": "Chapter 2"},
				},
			})
		}
		_, dashCfg, dashClient := setupTestChapterEnv(t, dashHandler)

		cmd, _, errOut := newTestRootCmd(dashCfg, dashClient)
		code := cmd.Execute([]string{"chapter", "reorder", "7", "-", "1,2"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if capturedPath != "/api/stories/7/scenes/reorder" {
			t.Fatalf("expected path /api/stories/7/scenes/reorder, got %s", capturedPath)
		}
		sceneIDs, ok := capturedBody["scene_ids"].([]any)
		if !ok {
			t.Fatalf("expected scene_ids in body: %v", capturedBody)
		}
		var idx1, idx2 int = -1, -1
		for i, id := range sceneIDs {
			if id == "--" {
				t.Fatalf("invented '--' found in scene_ids: %v", sceneIDs)
			}
			if id == float64(7) || id == "7" {
				t.Fatalf("book ID 7 leaked into scene_ids: %v", sceneIDs)
			}
			if id == float64(1) || id == "1" {
				idx1 = i
			}
			if id == float64(2) || id == "2" {
				idx2 = i
			}
		}
		if idx1 == -1 || idx2 == -1 || idx1 >= idx2 {
			t.Fatalf("expected chapter IDs 1 and 2 in order, got scene_ids: %v", sceneIDs)
		}
	})
}

func TestChapterDelete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/scenes/101" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/api/scenes/999" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "A story must retain at least one chapter.",
			})
			return
		}
		http.NotFound(w, r)
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "delete"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("delete cancelled via prompt", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("n\n")
		code := cmd.Execute([]string{"chapter", "delete", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d", code)
		}
		if out.Len() != 0 {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
		if !strings.Contains(errOut.String(), "Deletion cancelled.") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("delete confirmed via prompt", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("y\n")
		code := cmd.Execute([]string{"chapter", "delete", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted chapter 101.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete chapter 101? [y/N]: ") {
			t.Fatalf("expected prompt on stderr, got: %s", errOut.String())
		}
	})

	t.Run("delete with --yes flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "delete", "101", "--yes"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted chapter 101.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with -y flag and --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "delete", "101", "-y", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(101) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
	})

	t.Run("delete confirmed via prompt with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("y\n")
		code := cmd.Execute([]string{"chapter", "delete", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(101) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete chapter 101? [y/N]: ") {
			t.Fatalf("expected prompt on stderr, got: %s", errOut.String())
		}
	})

	t.Run("delete cancelled via prompt with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("n\n")
		code := cmd.Execute([]string{"chapter", "delete", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if out.Len() != 0 {
			t.Fatalf("expected empty stdout on cancellation, got: %s", out.String())
		}
		if !strings.Contains(errOut.String(), "Deletion cancelled.") {
			t.Fatalf("expected cancellation on stderr, got: %s", errOut.String())
		}
	})

	t.Run("error deleting last chapter", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "delete", "999", "--yes"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "A story must retain at least one chapter.") {
			t.Fatalf("unexpected error message: %s", errOut.String())
		}
	})
}

func TestChapterExport(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/scenes/101/export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte("# Chapter 1 Prose\n\nThe stars were silent."))
	}
	_, cfgPath, httpClient := setupTestChapterEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "export"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("export to stdout", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "export", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "# Chapter 1 Prose") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("export to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "exported_scene.md")

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "export", "101", "-o", outFile})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Exported chapter 101 to "+outFile) {
			t.Fatalf("unexpected stdout: %s", out.String())
		}

		data, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("failed to read exported file: %v", err)
		}
		if !strings.Contains(string(data), "# Chapter 1 Prose") {
			t.Fatalf("unexpected file content: %s", string(data))
		}
	})

	t.Run("export to file --json", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "exported_scene.md")

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chapter", "export", "101", "--output", outFile, "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(101) || res["output"] != outFile {
			t.Fatalf("unexpected json: %+v", res)
		}
	})
}
