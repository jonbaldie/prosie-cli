package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func setupTestSeriesEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-series-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-series-token")

	return server, configPath, server.Client()
}

func TestSeriesHelp(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"series", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Manage book series") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestSeriesUnknownSubcommand(t *testing.T) {
	cmd, _, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"series", "nonexistent"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown series command: nonexistent") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestSeriesList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/series":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":          1,
						"name":        "The Solar Cycle",
						"description": "Space opera trilogy",
						"story_ids":   []int{10, 20},
						"updated_at":  "2026-09-13T10:00:00Z",
					},
					{
						"id":          2,
						"name":        "Voidborne",
						"description": "Standalone universe",
						"story_ids":   []int{},
						"updated_at":  "2026-09-13T11:00:00Z",
					},
				},
			})
		case "/api/stories":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 10, "title": "Solar Drift", "series_id": 1},
					{"id": 20, "title": "Solar Eclipse", "series_id": 1},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("plain text tabular", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "list"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "TITLE") || !strings.Contains(stdout, "BOOKS") || !strings.Contains(stdout, "UPDATED") {
			t.Fatalf("missing table headers: %s", stdout)
		}
		if !strings.Contains(stdout, "The Solar Cycle") || !strings.Contains(stdout, "Solar Drift, Solar Eclipse") {
			t.Fatalf("missing series details: %s", stdout)
		}
		if !strings.Contains(stdout, "Voidborne") {
			t.Fatalf("missing Voidborne series: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "list", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var list []client.Series
		if err := json.Unmarshal(out.Bytes(), &list); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if len(list) != 2 || list[0].Title != "The Solar Cycle" {
			t.Fatalf("unexpected json list: %+v", list)
		}
		if strings.Contains(out.String(), `"chapters": []`) || strings.Contains(out.String(), `"chapters":[]`) {
			t.Fatalf("expected unloaded chapters to not be emitted as empty array, got: %s", out.String())
		}
	})

	t.Run("empty list", func(t *testing.T) {
		emptyHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
		_, emptyCfg, emptyHTTP := setupTestSeriesEnv(t, emptyHandler)

		cmd, out, errOut := newTestRootCmd(emptyCfg, emptyHTTP)
		code := cmd.Execute([]string{"series", "list"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No series found.") {
			t.Fatalf("expected 'No series found.', got: %s", out.String())
		}
	})
}

func TestSeriesShow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/series/1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":          1,
					"name":        "The Solar Cycle",
					"description": "Epic three-part space saga",
					"story_ids":   []int{10},
					"updated_at":  "2026-09-13T10:00:00Z",
				},
			})
		case "/api/stories":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 10, "title": "Solar Drift", "word_count": 15000, "series_id": 1},
				},
			})
		case "/api/series/1/codex-entries":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":       101,
						"name":     "Helios Drive",
						"category": "lore",
						"content":  "Jump drive mechanics.",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "show"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "series ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "show", "not-a-number"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "invalid series ID") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "show", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "The Solar Cycle") || !strings.Contains(stdout, "Epic three-part space saga") {
			t.Fatalf("missing details in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "Solar Drift") {
			t.Fatalf("missing book in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "Helios Drive") {
			t.Fatalf("missing shared codex entry in stdout: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "show", "1", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var s client.Series
		if err := json.Unmarshal(out.Bytes(), &s); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if s.ID != 1 || s.Title != "The Solar Cycle" || len(s.Books) != 1 || len(s.CodexEntries) != 1 {
			t.Fatalf("unexpected series json: %+v", s)
		}
		if strings.Contains(out.String(), `"chapters": []`) || strings.Contains(out.String(), `"chapters":[]`) {
			t.Fatalf("expected unloaded chapters to not be emitted as empty array, got: %s", out.String())
		}
	})

	t.Run("404 not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "show", "999"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "error fetching series") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})
}

func TestSeriesCreate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/series" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":          3,
				"name":        payload["name"],
				"description": payload["description"],
			},
		})
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("missing title flag", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "create"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--title is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful create plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "create", "--title", "Chronicles of Mars", "--description", "Mars colony story"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Created series \"Chronicles of Mars\" (ID: 3).") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("successful create --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "create", "--title", "Chronicles of Mars", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var s client.Series
		if err := json.Unmarshal(out.Bytes(), &s); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if s.ID != 3 || s.Title != "Chronicles of Mars" {
			t.Fatalf("unexpected json: %+v", s)
		}
	})
}

func TestSeriesUpdate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/series/3" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":          3,
				"name":        payload["name"],
				"description": payload["description"],
			},
		})
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "update"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "series ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful update plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "update", "3", "--title", "Updated Mars"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Updated series 3 (\"Updated Mars\").") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("successful update --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "update", "3", "--title", "Updated Mars", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var s client.Series
		if err := json.Unmarshal(out.Bytes(), &s); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if s.ID != 3 || s.Title != "Updated Mars" {
			t.Fatalf("unexpected json: %+v", s)
		}
	})
}

func TestSeriesDelete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/series/3" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "delete"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "series ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("cancelled via prompt", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("n\n")
		code := cmd.Execute([]string{"series", "delete", "3"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if out.Len() != 0 {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
		if !strings.Contains(errOut.String(), "Deletion cancelled.") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("confirmed via prompt", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("y\n")
		code := cmd.Execute([]string{"series", "delete", "3"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted series 3.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete series 3? [y/N]: ") {
			t.Fatalf("expected prompt on stderr, got: %s", errOut.String())
		}
	})

	t.Run("delete with --yes flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "delete", "3", "--yes"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted series 3.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with -y and --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "delete", "3", "-y", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(3) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
	})

	t.Run("delete confirmed via prompt with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("y\n")
		code := cmd.Execute([]string{"series", "delete", "3", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(3) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete series 3? [y/N]: ") {
			t.Fatalf("expected prompt on stderr, got: %s", errOut.String())
		}
	})

	t.Run("delete cancelled via prompt with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("n\n")
		code := cmd.Execute([]string{"series", "delete", "3", "--json"})
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
}

func TestSeriesAttach(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
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
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("missing arguments", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "attach", "1"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "series ID and book ID are required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("attach plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "attach", "1", "10"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Attached book 10 (\"Solar Drift\") to series 1.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("attach --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "attach", "1", "10", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var b client.Book
		if err := json.Unmarshal(out.Bytes(), &b); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if b.ID != 10 || b.SeriesID == nil || *b.SeriesID != 1 {
			t.Fatalf("unexpected book json: %+v", b)
		}
		if strings.Contains(out.String(), `"chapters": []`) || strings.Contains(out.String(), `"chapters":[]`) {
			t.Fatalf("expected unloaded chapters to not be emitted as empty array, got: %s", out.String())
		}
	})
}

func TestSeriesDetach(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/series/1/stories/10" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	_, cfgPath, httpClient := setupTestSeriesEnv(t, handler)

	t.Run("missing arguments", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "detach", "1"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "series ID and book ID are required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("detach plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "detach", "1", "10"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Detached book 10 from series 1.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("detach --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"series", "detach", "1", "10", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["series_id"] != float64(1) || res["book_id"] != float64(10) || res["detached"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
	})
}
