package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCodexEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/api/stories/1/codex-entries" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":         1,
					"story_id":   1,
					"name":       "Marcus Kane",
					"category":   "character",
					"content":    "Chief Navigator aboard the Astraea.",
					"aliases":    "The Navigator",
					"updated_at": "2026-09-13T12:00:00Z",
				},
				{
					"id":         2,
					"story_id":   1,
					"name":       "Astraea Station",
					"category":   "lore",
					"content":    "Orbital habitat above Jupiter.",
					"updated_at": "2026-09-13T13:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	entries, err := cli.ListCodexEntries(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListCodexEntries returned error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].ID != 1 || entries[0].Name != "Marcus Kane" || entries[0].DisplayType() != "character" {
		t.Fatalf("unexpected entry[0]: %+v", entries[0])
	}
	if entries[0].DisplayDetails() != "Chief Navigator aboard the Astraea." {
		t.Fatalf("unexpected details: %q", entries[0].DisplayDetails())
	}
	if entries[0].DisplayAliases() != "The Navigator" {
		t.Fatalf("unexpected aliases: %q", entries[0].DisplayAliases())
	}
	if entries[1].ID != 2 || entries[1].Name != "Astraea Station" || entries[1].DisplayType() != "lore" {
		t.Fatalf("unexpected entry[1]: %+v", entries[1])
	}
}

func TestListSeriesCodexEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/series/5/codex-entries" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":        50,
					"series_id": 5,
					"name":      "Jump Gate Protocol",
					"category":  "lore",
					"content":   "Standard orbital rendezvous rules.",
				},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	entries, err := cli.ListSeriesCodexEntries(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListSeriesCodexEntries returned error: %v", err)
	}

	if len(entries) != 1 || entries[0].ID != 50 || entries[0].Name != "Jump Gate Protocol" {
		t.Fatalf("unexpected series codex entries: %+v", entries)
	}
}

func TestGetCodexEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/codex-entries/1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         1,
				"story_id":   1,
				"name":       "Marcus Kane",
				"category":   "character",
				"content":    "Chief Navigator aboard the Astraea.",
				"aliases":    "The Navigator",
				"updated_at": "2026-09-13T12:00:00Z",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	entry, err := cli.GetCodexEntry(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetCodexEntry returned error: %v", err)
	}

	if entry.ID != 1 || entry.Name != "Marcus Kane" || entry.DisplayType() != "character" {
		t.Fatalf("unexpected codex entry: %+v", entry)
	}
}

func TestCreateCodexEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/stories/1/codex-entries" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["name"] != "Lyra Vance" {
			t.Errorf("unexpected payload name: %v", payload["name"])
		}
		if payload["category"] != "character" {
			t.Errorf("unexpected category: %v", payload["category"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       15,
				"story_id": 1,
				"name":     "Lyra Vance",
				"category": "character",
				"content":  "Lead engineer and pilot.",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	entry, err := cli.CreateCodexEntry(context.Background(), 1, CreateCodexParams{
		Name:    "Lyra Vance",
		Type:    "character",
		Details: "Lead engineer and pilot.",
	})
	if err != nil {
		t.Fatalf("CreateCodexEntry returned error: %v", err)
	}
	if entry.ID != 15 || entry.Name != "Lyra Vance" || entry.DisplayType() != "character" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestUpdateCodexEntry(t *testing.T) {
	newName := "Commander Lyra Vance"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/codex-entries/15" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       15,
				"story_id": 1,
				"name":     newName,
				"category": "character",
				"content":  "Promoted lead engineer.",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	entry, err := cli.UpdateCodexEntry(context.Background(), 15, UpdateCodexParams{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateCodexEntry returned error: %v", err)
	}
	if entry.ID != 15 || entry.Name != newName {
		t.Fatalf("unexpected updated entry: %+v", entry)
	}
}

func TestDeleteCodexEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/codex-entries/15" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	err := cli.DeleteCodexEntry(context.Background(), 15)
	if err != nil {
		t.Fatalf("DeleteCodexEntry returned error: %v", err)
	}
}
