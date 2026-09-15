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

func TestCodexEditingPreservesNotesAndAliases(t *testing.T) {
	for _, tc := range []struct {
		name               string
		args               []string
		method, path, body string
	}{
		{"create character", []string{"codex", "create", "21", "--name", "Ada", "--type", "character", "--details", "Navigator", "--aliases", "Captain"}, "POST", "/api/stories/21/codex-entries", `{"name":"Ada","category":"character","type":"character","details":"Navigator","content":"Navigator","aliases":"Captain"}`},
		{"create through aliases", []string{"codex", "create", "21", "--name", "Ada", "--category", "character", "--content", "Navigator"}, "POST", "/api/stories/21/codex-entries", `{"name":"Ada","category":"character","type":"character","details":"Navigator","content":"Navigator"}`},
		{"default lore", []string{"codex", "create", "21", "--name", "Island", "--details", "Harbour", "--aliases", "  "}, "POST", "/api/stories/21/codex-entries", `{"name":"Island","category":"lore","type":"lore","details":"Harbour","content":"Harbour"}`},
		{"update character", []string{"codex", "update", "41", "--name", "Ada", "--type", "character", "--details", "Navigator", "--aliases", "Captain"}, "PATCH", "/api/codex-entries/41", `{"name":"Ada","category":"character","type":"character","details":"Navigator","content":"Navigator","aliases":"Captain"}`},
		{"update through aliases", []string{"codex", "update", "41", "--category", "character", "--content", "Navigator"}, "PATCH", "/api/codex-entries/41", `{"category":"character","type":"character","details":"Navigator","content":"Navigator"}`},
		{"clear primary fields", []string{"codex", "update", "41", "--name=", "--type=", "--category", "lore", "--details=", "--content", "Old note", "--aliases="}, "PATCH", "/api/codex-entries/41", `{"name":"","category":"","type":"","details":"","content":"","aliases":""}`},
		{"keep omitted fields", []string{"codex", "update", "41", "--name", "Ada"}, "PATCH", "/api/codex-entries/41", `{"name":"Ada"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var wantBody map[string]any
			if err := json.Unmarshal([]byte(tc.body), &wantBody); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request=%s %s; want=%s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				var got map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(got, wantBody) {
					t.Errorf("body=%#v; want=%#v", got, wantBody)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data":{"id":41,"name":"Ada","category":"character","content":"Navigator","aliases":"Captain","story_id":21}}`)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			if code := root.Execute(append(tc.args, "--json")); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			var got, want map[string]any
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(`{"id":41,"name":"Ada","category":"character","type":"character","content":"Navigator","details":"Navigator","aliases":"Captain","story_id":21}`), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("output=%#v; want=%#v", got, want)
			}
		})
	}
}
