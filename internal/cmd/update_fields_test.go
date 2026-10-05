package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

// Updating notes must distinguish clearing a field from leaving it unchanged.
func TestBookUpdatePreservesExplicitAndOmittedFields(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
		body  string
	}{
		{"replace notes", []string{"--title", "Voyage", "--premise", "A rescue", "--lore", "Island", "--characters", "Ada", "--target-words", "60000"}, `{"title":"Voyage","story_so_far":"A rescue","lore":"Island","characters":"Ada","target_word_count":60000}`},
		{"clear notes", []string{"--title=", "--premise=", "--lore=", "--characters=", "--target-words=0"}, `{"title":"","story_so_far":"","lore":"","characters":"","target_word_count":0}`},
		{"keep other notes", []string{"--title", "Voyage"}, `{"title":"Voyage"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != "PATCH" || r.URL.Path != "/api/stories/41" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer fixture-token" {
					t.Error("missing credentials")
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
				fmt.Fprint(w, `{"data":{"id":41,"title":"Voyage","word_count":13,"filter_using_story_so_far":true}}`)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.HTTPClient, root.ConfigPath = out, errOut, server.Client(), filepath.Join(t.TempDir(), "config.json")
			args := append([]string{"book", "update", "41", "--json"}, tc.flags...)
			if code := root.Execute(args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			if requests != 1 {
				t.Fatalf("expected one update, got %d", requests)
			}
			var got, want map[string]any
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(`{"id":41,"title":"Voyage","word_count":13,"filter_using_story_so_far":true}`), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("output: got %#v; want %#v", got, want)
			}
		})
	}
}

// The scene responder validates name and not title, so --title is sent as name only.
func TestChapterTitleIsSentAsName(t *testing.T) {
	tests := []struct {
		name, method, path string
		args               []string
		body               string
	}{
		{"create", "POST", "/api/stories/21/scenes", []string{"chapter", "create", "21", "--title", "Arrival", "--summary", "Docking"}, `{"name":"Arrival","summary":"Docking"}`},
		{"update", "PATCH", "/api/scenes/41", []string{"chapter", "update", "41", "--title", "Arrival", "--content", "Prose"}, `{"name":"Arrival","content":"Prose"}`},
		{"clear title", "PATCH", "/api/scenes/41", []string{"chapter", "update", "41", "--title="}, `{"name":""}`},
		{"keep omitted fields", "PATCH", "/api/scenes/41", []string{"chapter", "update", "41", "--summary", "Docking"}, `{"summary":"Docking"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request=%s %s; want=%s %s", r.Method, r.URL.Path, tc.method, tc.path)
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
				fmt.Fprint(w, `{"data":{"id":41,"story_id":21,"order":0,"name":"Arrival"}}`)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.HTTPClient, root.ConfigPath = out, errOut, server.Client(), filepath.Join(t.TempDir(), "config.json")
			if code := root.Execute(append(tc.args, "--json")); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			if requests != 1 {
				t.Fatalf("expected one request, got %d", requests)
			}
		})
	}
}

// The story responder validates story_so_far and not premise, so --premise is sent as story_so_far only.
func TestBookCreateSendsPremiseAsStorySoFar(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
		body  string
	}{
		{"premise", []string{"--title", "Voyage", "--premise", "A rescue", "--target-words", "60000"}, `{"title":"Voyage","story_so_far":"A rescue","target_word_count":60000}`},
		{"title only", []string{"--title", "Voyage"}, `{"title":"Voyage"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != "POST" || r.URL.Path != "/api/stories" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
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
				fmt.Fprint(w, `{"data":{"id":41,"title":"Voyage","word_count":0,"filter_using_story_so_far":true}}`)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.HTTPClient, root.ConfigPath = out, errOut, server.Client(), filepath.Join(t.TempDir(), "config.json")
			args := append([]string{"book", "create", "--json"}, tc.flags...)
			if code := root.Execute(args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			if requests != 1 {
				t.Fatalf("expected one create, got %d", requests)
			}
		})
	}
}
