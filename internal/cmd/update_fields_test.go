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
		{"replace notes", []string{"--title", "Voyage", "--premise", "A rescue", "--lore", "Island", "--characters", "Ada", "--target-words", "60000"}, `{"title":"Voyage","premise":"A rescue","story_so_far":"A rescue","lore":"Island","characters":"Ada","target_word_count":60000}`},
		{"clear notes", []string{"--title=", "--premise=", "--lore=", "--characters=", "--target-words=0"}, `{"title":"","premise":"","story_so_far":"","lore":"","characters":"","target_word_count":0}`},
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
