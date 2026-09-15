package cmd_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

// An HTTP success status does not make an unreadable API result a successful command.
func TestCommandsRejectMalformedAPIResponses(t *testing.T) {
	for _, args := range [][]string{
		{"book", "list"}, {"book", "show", "41"}, {"book", "create", "--title", "Voyage"}, {"book", "update", "41", "--title", "Voyage"}, {"book", "duplicate", "41"},
		{"chapter", "list", "21"}, {"chapter", "show", "41"}, {"chapter", "create", "21", "--title", "Arrival"}, {"chapter", "update", "41", "--content", "Prose"}, {"chapter", "reorder", "21", "41,42"},
		{"codex", "list", "21"}, {"codex", "show", "41"}, {"codex", "create", "21", "--name", "Ada", "--details", "Navigator"}, {"codex", "update", "41", "--name", "Ada"},
		{"series", "list"}, {"series", "show", "41"}, {"series", "create", "--title", "Voyages"}, {"series", "update", "41", "--title", "Voyages"}, {"series", "attach", "21", "41"},
		{"chat", "list", "21"}, {"chat", "show", "41"}, {"chat", "send", "--conversation", "41", "Hello"},
		{"generate", "continue", "41", "--no-stream"}, {"generate", "rewrite", "41", "--selection", "Prose", "--prompt", "Tighten"}, {"generate", "summarize", "41"}, {"generate", "reject", "41"},
	} {
		t.Run(strings.Join(args[:2], "/"), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, "broken JSON")
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			if code := root.Execute(append(args, "--json")); code != 1 {
				t.Fatalf("code=%d; want=1 stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if out.Len() != 0 || !strings.Contains(errOut.String(), "failed to decode") {
				t.Fatalf("stdout=%q stderr=%q", out.String(), errOut.String())
			}
		})
	}
}
