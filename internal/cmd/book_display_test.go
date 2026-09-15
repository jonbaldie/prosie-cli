package cmd_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestBookDetailsUseAvailableNotesAndChapterNames(t *testing.T) {
	for _, tc := range []struct{ name, book, want string }{
		{"complete manuscript", `{"id":41,"title":"Voyage","premise":"Rescue","story_so_far":"Older summary","subtitle":"Older subtitle","lore":"Island","characters":"Ada","word_count":3,"target_word_count":100,"target_progress_percent":3,"updated_at":"2026-09-13T14:05:06Z","chapters":[{"id":71,"title":"Arrival","name":"Old name","order":0,"word_count":1},{"id":72,"name":"Departure","order":1,"word_count":2}]}`,
			"Title:       Voyage\nID:          41\nWords:       3 / 100 (3%)\nPremise:     Rescue\nLore:        Island\nCharacters:  Ada\nUpdated:     2026-09-13 14:05\n\nChapters (2):\n  1. Arrival (1 word)\n  2. Departure (2 words)\n"},
		{"summary fallback", `{"id":41,"title":"Voyage","premise":"","story_so_far":"Rescue","word_count":3,"target_word_count":100,"chapters":[{"id":71,"order":2,"word_count":3}]}`,
			"Title:       Voyage\nID:          41\nWords:       3 / 100\nPremise:     Rescue\nLore:        -\nCharacters:  -\nUpdated:     -\n\nChapters (1):\n  1. Chapter 3 (3 words)\n"},
		{"subtitle fallback", `{"id":41,"title":"Voyage","premise":"","story_so_far":"","subtitle":"Rescue","lore":"","characters":"","word_count":0,"chapters":[]}`,
			"Title:       Voyage\nID:          41\nWords:       0\nPremise:     Rescue\nLore:        -\nCharacters:  -\nUpdated:     -\n\nChapters: (none)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/stories/41":
					fmt.Fprint(w, tc.book)
				case "/api/stories/41/scenes":
					fmt.Fprint(w, `{"data":[]}`)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			if code := root.Execute([]string{"book", "show", "41"}); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			if out.String() != tc.want {
				t.Fatalf("output=%q; want=%q", out.String(), tc.want)
			}
		})
	}
}
