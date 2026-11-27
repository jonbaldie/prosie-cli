package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

// An update with no field flag changes nothing. It must fail and send no request.
func TestEmptyUpdateIsRejected(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"book no flags", []string{"book", "update", "1"}},
		{"book json only", []string{"book", "update", "1", "--json"}},
		{"chapter no flags", []string{"chapter", "update", "41"}},
		{"chapter json only", []string{"chapter", "update", "41", "--json"}},
		{"chapter empty file", []string{"chapter", "update", "41", "--file", ""}},
		{"series no flags", []string{"series", "update", "1"}},
		{"series json only", []string{"series", "update", "1", "--json"}},
		{"codex no flags", []string{"codex", "update", "1"}},
		{"codex json only", []string{"codex", "update", "1", "--json"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data":{"id":1,"title":"Book One","name":"Chapter One"}}`)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.HTTPClient, root.ConfigPath = out, errOut, server.Client(), filepath.Join(t.TempDir(), "config.json")
			code := root.Execute(tc.args)
			if code != 1 {
				t.Fatalf("code=%d; want 1. stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if !strings.Contains(errOut.String(), "at least one field flag is required") {
				t.Fatalf("stderr=%q; want a message that says at least one field flag is required", errOut.String())
			}
			if requests != 0 {
				t.Fatalf("requests=%d; want 0", requests)
			}
		})
	}
}

// One field flag counts as a field, also with an explicit empty or zero value.
func TestUpdateWithOneExplicitField(t *testing.T) {
	tests := []struct {
		name string
		args []string
		path string
		body string
		resp string
	}{
		{"book empty title", []string{"book", "update", "1", "--title="}, "/api/stories/1", `{"title":""}`, `{"data":{"id":1,"title":"","word_count":0}}`},
		{"book premise", []string{"book", "update", "1", "--premise", "Voyage"}, "/api/stories/1", `{"story_so_far":"Voyage"}`, `{"data":{"id":1,"title":"Book One","word_count":0}}`},
		{"book lore", []string{"book", "update", "1", "--lore", "Island"}, "/api/stories/1", `{"lore":"Island"}`, `{"data":{"id":1,"title":"Book One","word_count":0}}`},
		{"book characters", []string{"book", "update", "1", "--characters", "Ada"}, "/api/stories/1", `{"characters":"Ada"}`, `{"data":{"id":1,"title":"Book One","word_count":0}}`},
		{"book zero target words", []string{"book", "update", "1", "--target-words", "0"}, "/api/stories/1", `{"target_word_count":0}`, `{"data":{"id":1,"title":"Book One","word_count":0}}`},
		{"chapter title", []string{"chapter", "update", "41", "--title", "Arrival"}, "/api/scenes/41", `{"name":"Arrival"}`, `{"data":{"id":41,"story_id":21,"order":0,"name":"Arrival"}}`},
		{"chapter content", []string{"chapter", "update", "41", "--content", "Prose"}, "/api/scenes/41", `{"content":"Prose"}`, `{"data":{"id":41,"story_id":21,"order":0,"name":"Chapter One"}}`},
		{"chapter summary", []string{"chapter", "update", "41", "--summary", "Docking"}, "/api/scenes/41", `{"summary":"Docking"}`, `{"data":{"id":41,"story_id":21,"order":0,"name":"Chapter One"}}`},
		{"chapter pov", []string{"chapter", "update", "41", "--pov", "Ada"}, "/api/scenes/41", `{"custom_pov":"Ada"}`, `{"data":{"id":41,"story_id":21,"order":0,"name":"Chapter One"}}`},
		{"chapter character", []string{"chapter", "update", "41", "--character", "Ada"}, "/api/scenes/41", `{"custom_character":"Ada"}`, `{"data":{"id":41,"story_id":21,"order":0,"name":"Chapter One"}}`},
		{"chapter zero order", []string{"chapter", "update", "41", "--order", "0"}, "/api/scenes/41", `{"order":0}`, `{"data":{"id":41,"story_id":21,"order":0,"name":"Chapter One"}}`},
		{"series title", []string{"series", "update", "1", "--title", "Voyages"}, "/api/series/1", `{"name":"Voyages","title":"Voyages"}`, `{"data":{"id":1,"name":"Voyages","title":"Voyages"}}`},
		{"series name", []string{"series", "update", "1", "--name", "Voyages"}, "/api/series/1", `{"name":"Voyages","title":"Voyages"}`, `{"data":{"id":1,"name":"Voyages","title":"Voyages"}}`},
		{"series empty description", []string{"series", "update", "1", "--description", ""}, "/api/series/1", `{"description":""}`, `{"data":{"id":1,"name":"Series One","description":""}}`},
		{"codex empty name", []string{"codex", "update", "1", "--name="}, "/api/codex-entries/1", `{"name":""}`, `{"data":{"id":1,"name":""}}`},
		{"codex type", []string{"codex", "update", "1", "--type", "character"}, "/api/codex-entries/1", `{"category":"character"}`, `{"data":{"id":1,"name":"Ada","category":"character"}}`},
		{"codex category", []string{"codex", "update", "1", "--category", "character"}, "/api/codex-entries/1", `{"category":"character"}`, `{"data":{"id":1,"name":"Ada","category":"character"}}`},
		{"codex details", []string{"codex", "update", "1", "--details", "Navigator"}, "/api/codex-entries/1", `{"content":"Navigator"}`, `{"data":{"id":1,"name":"Ada","content":"Navigator"}}`},
		{"codex content", []string{"codex", "update", "1", "--content", "Navigator"}, "/api/codex-entries/1", `{"content":"Navigator"}`, `{"data":{"id":1,"name":"Ada","content":"Navigator"}}`},
		{"codex aliases", []string{"codex", "update", "1", "--aliases", "Captain"}, "/api/codex-entries/1", `{"aliases":"Captain"}`, `{"data":{"id":1,"name":"Ada","aliases":"Captain"}}`},
		{"codex zero order", []string{"codex", "update", "1", "--order", "0"}, "/api/codex-entries/1", `{"order":0}`, `{"data":{"id":1,"name":"Ada"}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != "PATCH" || r.URL.Path != tc.path {
					t.Errorf("request=%s %s; want PATCH %s", r.Method, r.URL.Path, tc.path)
				}
				var got, want map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("request body: got %#v; want %#v", got, want)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.resp)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.HTTPClient, root.ConfigPath = out, errOut, server.Client(), filepath.Join(t.TempDir(), "config.json")
			code := root.Execute(tc.args)
			if code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d; want 0 and empty stderr. stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if requests != 1 {
				t.Fatalf("requests=%d; want 1", requests)
			}
		})
	}
}
