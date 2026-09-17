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

func setupTestCodexEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-codex-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-codex-token")

	return server, configPath, server.Client()
}

func TestCodexHelp(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"codex", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Manage story codex entries") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestCodexUnknownSubcommand(t *testing.T) {
	cmd, _, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"codex", "unknown"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown codex command: unknown") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCodexList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/codex-entries" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":         10,
					"story_id":   1,
					"name":       "Marcus Vance",
					"category":   "character",
					"content":    "Chief pilot of the colony vessel.",
					"aliases":    "Old Vance",
					"updated_at": "2026-09-13T10:00:00Z",
				},
				{
					"id":         20,
					"story_id":   1,
					"name":       "Vanguard Station",
					"category":   "lore",
					"content":    "Orbital outpost situated near Mars.",
					"updated_at": "2026-09-13T11:00:00Z",
				},
			},
		})
	}
	_, cfgPath, httpClient := setupTestCodexEnv(t, handler)

	t.Run("missing book id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "list"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text tabular", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "list", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "TYPE") || !strings.Contains(stdout, "NAME") || !strings.Contains(stdout, "DETAILS") || !strings.Contains(stdout, "UPDATED") {
			t.Fatalf("missing table headers: %s", stdout)
		}
		if !strings.Contains(stdout, "Marcus Vance") || !strings.Contains(stdout, "character") {
			t.Fatalf("missing Marcus Vance entry: %s", stdout)
		}
		if !strings.Contains(stdout, "Vanguard Station") || !strings.Contains(stdout, "lore") {
			t.Fatalf("missing Vanguard Station entry: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "list", "1", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var list []client.CodexEntry
		if err := json.Unmarshal(out.Bytes(), &list); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if len(list) != 2 || list[0].Name != "Marcus Vance" {
			t.Fatalf("unexpected list: %+v", list)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		emptyHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
		_, emptyCfg, emptyHTTP := setupTestCodexEnv(t, emptyHandler)

		cmd, out, errOut := newTestRootCmd(emptyCfg, emptyHTTP)
		code := cmd.Execute([]string{"codex", "list", "2"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No codex entries found for book 2.") {
			t.Fatalf("expected empty message, got: %s", out.String())
		}
	})
}

func TestCodexShow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/codex-entries/10" {
			http.NotFound(w, r)
			return
		}
		storyID := 1
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         10,
				"story_id":   storyID,
				"name":       "Marcus Vance",
				"category":   "character",
				"content":    "Chief pilot of the colony vessel.",
				"aliases":    "Old Vance",
				"updated_at": "2026-09-13T10:00:00Z",
			},
		})
	}
	_, cfgPath, httpClient := setupTestCodexEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "show"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "codex entry ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "show", "bad"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "invalid codex entry ID") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "show", "10"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "Marcus Vance") || !strings.Contains(stdout, "character") || !strings.Contains(stdout, "Old Vance") {
			t.Fatalf("missing fields in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "Chief pilot of the colony vessel.") {
			t.Fatalf("missing details in stdout: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "show", "10", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var entry client.CodexEntry
		if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if entry.ID != 10 || entry.Name != "Marcus Vance" || entry.Display().Type != "character" {
			t.Fatalf("unexpected entry json: %+v", entry)
		}
	})
}

func TestCodexCreate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/stories/1/codex-entries" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		storyID := 1
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       100,
				"story_id": storyID,
				"name":     payload["name"],
				"category": payload["category"],
				"content":  payload["content"],
				"aliases":  payload["aliases"],
			},
		})
	}
	_, cfgPath, httpClient := setupTestCodexEnv(t, handler)

	t.Run("missing book id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "create"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("missing name flag", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "create", "1", "--details", "Some details"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--name is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("missing details flag", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "create", "1", "--name", "Test Name"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--details is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful create plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "create", "1", "--name", "Dr. Sarah Paulson", "--type", "character", "--details", "Chief medical officer."})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Created codex entry \"Dr. Sarah Paulson\" (ID: 100, type: character).") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("successful create --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "create", "1", "--name", "Jump Gate Protocol", "--type", "lore", "--details", "FTL mechanics.", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var entry client.CodexEntry
		if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if entry.ID != 100 || entry.Name != "Jump Gate Protocol" || entry.Display().Type != "lore" {
			t.Fatalf("unexpected entry json: %+v", entry)
		}
	})
}

func TestCodexUpdate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/codex-entries/100" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       100,
				"name":     payload["name"],
				"category": payload["category"],
				"content":  payload["content"],
			},
		})
	}
	_, cfgPath, httpClient := setupTestCodexEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "update"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "codex entry ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("successful update plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "update", "100", "--name", "Senior Officer Paulson"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Updated codex entry 100 (\"Senior Officer Paulson\").") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("successful update --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "update", "100", "--name", "Senior Officer Paulson", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var entry client.CodexEntry
		if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if entry.ID != 100 || entry.Name != "Senior Officer Paulson" {
			t.Fatalf("unexpected entry json: %+v", entry)
		}
	})
}

func TestCodexDelete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/codex-entries/100" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	_, cfgPath, httpClient := setupTestCodexEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "delete"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "codex entry ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("cancelled via prompt", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("n\n")
		code := cmd.Execute([]string{"codex", "delete", "100"})
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
		code := cmd.Execute([]string{"codex", "delete", "100"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted codex entry 100.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete codex entry 100? [y/N]: ") {
			t.Fatalf("expected prompt on stderr, got: %s", errOut.String())
		}
	})

	t.Run("delete with --yes flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "delete", "100", "--yes"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Deleted codex entry 100.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with -y and --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"codex", "delete", "100", "-y", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(100) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
	})

	t.Run("delete confirmed via prompt with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("y\n")
		code := cmd.Execute([]string{"codex", "delete", "100", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != float64(100) || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete codex entry 100? [y/N]: ") {
			t.Fatalf("expected prompt on stderr, got: %s", errOut.String())
		}
	})

	t.Run("delete cancelled via prompt with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = strings.NewReader("n\n")
		code := cmd.Execute([]string{"codex", "delete", "100", "--json"})
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
