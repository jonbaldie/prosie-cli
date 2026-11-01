package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"regexp"
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

// assertNoUnloadedSeriesRelations fails if a JSON payload reports unloaded relations as empty arrays.
func assertNoUnloadedSeriesRelations(t *testing.T, out string) {
	t.Helper()
	for _, field := range []string{"books", "stories", "codex_entries", "chapters"} {
		for _, form := range []string{`"` + field + `": []`, `"` + field + `":[]`} {
			if strings.Contains(out, form) {
				t.Fatalf("expected unloaded %s to not be emitted as empty array, got: %s", field, out)
			}
		}
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
		assertNoUnloadedSeriesRelations(t, out.String())
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
		assertNoUnloadedSeriesRelations(t, out.String())
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
		assertNoUnloadedSeriesRelations(t, out.String())
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

func TestSeriesListMembership(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			requests := map[string]int{}
			handler := func(w http.ResponseWriter, r *http.Request) {
				requests[r.URL.Path]++
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/series":
					_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"Explicit","story_ids":[20,99,10,20]},{"id":2,"name":"Fallback"},{"id":3,"name":"Empty"},{"id":4,"name":"Embedded books","books":[{"id":40,"title":"Inside"}]},{"id":5,"name":"Embedded stories","stories":[{"id":50,"title":"Within"}]},{"id":6,"name":"Missing","story_ids":[99]}]}`))
				case "/api/stories":
					_, _ = w.Write([]byte(`{"data":[{"id":20,"title":"Old","series_id":2},{"id":10,"title":"First","series_id":2},{"id":20,"title":"Last","series_id":2},{"id":60,"title":"Other","series_id":4}]}`))
				default:
					http.NotFound(w, r)
				}
			}
			_, cfg, httpClient := setupTestSeriesEnv(t, handler)
			cmd, out, errOut := newTestRootCmd(cfg, httpClient)
			args := []string{"series", "list"}
			if format == "json" {
				args = append(args, "--json")
			}
			if code := cmd.Execute(args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut.String())
			}
			if requests["/api/series"] != 1 || requests["/api/stories"] != 1 || len(requests) != 2 {
				t.Fatalf("requests: %v", requests)
			}
			if format == "text" {
				assertSeriesTextRows(t, out.String(), [][]string{
					{"1", "Explicit", "Last, First, Last", "-"},
					{"2", "Fallback", "Old, First, Last", "-"},
					{"3", "Empty", "-", "-"},
					{"4", "Embedded books", "Inside", "-"},
					{"5", "Embedded stories", "Within", "-"},
					{"6", "Missing", "Book 99", "-"},
				})
				return
			}
			var got []struct {
				Books []struct {
					ID    int    `json:"id"`
					Title string `json:"title"`
				} `json:"books"`
				Stories []struct {
					ID    int    `json:"id"`
					Title string `json:"title"`
				} `json:"stories"`
			}
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != 6 {
				t.Fatalf("output: %s", out.String())
			}
			want := [][]string{{"Last", "First", "Last"}, {"Old", "First", "Last"}, nil, {"Inside"}, {"Within"}, nil}
			for i, s := range got {
				var books, stories []string
				for _, b := range s.Books {
					books = append(books, b.Title)
				}
				for _, b := range s.Stories {
					stories = append(stories, b.Title)
				}
				if !reflect.DeepEqual(books, want[i]) || !reflect.DeepEqual(stories, want[i]) {
					t.Fatalf("series %d: books %v, stories %v", i, books, stories)
				}
			}
			assertNoUnloadedSeriesRelations(t, out.String())
		})
	}
}

func TestSeriesListBookFetchCases(t *testing.T) {
	for _, tc := range []struct {
		name, series, books string
		status, requests    int
	}{
		{"failed", `[{"id":1,"name":"Unloaded"},{"id":2,"name":"Kept","stories":[{"id":7,"title":"Inside"}]}]`, `failure`, 500, 1},
		{"empty books", `[{"id":1,"name":"Unloaded"},{"id":2,"name":"Kept","stories":[{"id":7,"title":"Inside"}]}]`, `{"data":[]}`, 200, 1},
		{"all embedded", `[{"id":2,"name":"Kept","stories":[{"id":7,"title":"Inside"}]}]`, ``, 200, 0},
		{"empty series", `[]`, ``, 200, 0},
	} {
		for _, format := range []string{"text", "json"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				counts := map[string]int{}
				_, cfg, httpClient := setupTestSeriesEnv(t, func(w http.ResponseWriter, r *http.Request) {
					counts[r.URL.Path]++
					switch r.URL.Path {
					case "/api/series":
						_, _ = w.Write([]byte(`{"data":` + tc.series + `}`))
					case "/api/stories":
						w.WriteHeader(tc.status)
						_, _ = w.Write([]byte(tc.books))
					default:
						http.NotFound(w, r)
					}
				})
				cmd, out, errOut := newTestRootCmd(cfg, httpClient)
				args := []string{"series", "list"}
				if format == "json" {
					args = append(args, "--json")
				}
				if code := cmd.Execute(args); code != 0 || errOut.Len() != 0 {
					t.Fatalf("exit %d, stderr %q", code, errOut.String())
				}
				if counts["/api/series"] != 1 || counts["/api/stories"] != tc.requests || len(counts) != 1+tc.requests {
					t.Fatalf("requests: %v", counts)
				}
				if tc.name == "empty series" {
					want := "No series found."
					if format == "json" {
						want = "[]"
					}
					if strings.TrimSpace(out.String()) != want {
						t.Fatalf("output: %s", out.String())
					}
					return
				}
				if format == "text" {
					want := [][]string{{"2", "Kept", "Inside", "-"}}
					if tc.requests == 1 {
						want = append([][]string{{"1", "Unloaded", "-", "-"}}, want...)
					}
					assertSeriesTextRows(t, out.String(), want)
				}
				if format == "json" {
					if !strings.Contains(out.String(), "Inside") {
						t.Fatalf("output: %s", out.String())
					}
					var got []map[string]json.RawMessage
					if err := json.Unmarshal(out.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if tc.requests == 1 {
						if _, ok := got[0]["books"]; ok {
							t.Fatalf("unexpected books: %s", out.String())
						}
						if _, ok := got[0]["stories"]; ok {
							t.Fatalf("unexpected stories: %s", out.String())
						}
					}
					assertNoUnloadedSeriesRelations(t, out.String())
				}
			})
		}
	}
}

func assertSeriesTextRows(t *testing.T, output string, want [][]string) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != len(want)+1 {
		t.Fatalf("output: %q", output)
	}
	separator := regexp.MustCompile(` {2,}`)
	for i, line := range lines[1:] {
		got := separator.Split(strings.TrimSpace(line), -1)
		if !reflect.DeepEqual(got, want[i]) {
			t.Fatalf("row %d: got %q, want %q", i+1, got, want[i])
		}
	}
}

func TestSeriesShowMembership(t *testing.T) {
	for _, tc := range []struct {
		name, series string
		want         []string
	}{
		{"explicit", `{"id":1,"name":"Shown","story_ids":[20,99,10,20]}`, []string{"Last", "First", "Last"}},
		{"fallback", `{"id":1,"name":"Shown"}`, []string{"Old", "First", "Last"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			counts := map[string]int{}
			_, cfg, httpClient := setupTestSeriesEnv(t, func(w http.ResponseWriter, r *http.Request) {
				counts[r.URL.Path]++
				switch r.URL.Path {
				case "/api/series/1":
					_, _ = w.Write([]byte(tc.series))
				case "/api/stories":
					_, _ = w.Write([]byte(`{"data":[{"id":20,"title":"Old","series_id":1},{"id":10,"title":"First","series_id":1},{"id":20,"title":"Last","series_id":1},{"id":30,"title":"Other","series_id":2}]}`))
				case "/api/series/1/codex-entries":
					_, _ = w.Write([]byte(`{"data":[]}`))
				default:
					http.NotFound(w, r)
				}
			})
			cmd, out, errOut := newTestRootCmd(cfg, httpClient)
			if code := cmd.Execute([]string{"series", "show", "1", "--json"}); code != 0 || errOut.Len() != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut.String())
			}
			var got client.Series
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			var titles []string
			for _, book := range got.Books {
				titles = append(titles, book.Title)
			}
			if !reflect.DeepEqual(titles, tc.want) {
				t.Fatalf("books: got %q, want %q", titles, tc.want)
			}
			if !reflect.DeepEqual(counts, map[string]int{"/api/series/1": 1, "/api/stories": 1, "/api/series/1/codex-entries": 1}) {
				t.Fatalf("requests: %v", counts)
			}
		})
	}
}
