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

func TestContinuationStreamsCompleteTextAndReportsErrors(t *testing.T) {
	for _, tc := range []struct {
		name, events, output, diagnostic string
		code                             int
	}{
		{"tokens", "event: delta\ndata: {\"delta\":\"Café \"}\n\nevent: delta\ndata: {\"delta\":\"opened.\"}\n\n", "Café opened.\n", "", 0},
		{"existing newline", "event: delta\ndata: {\"delta\":\"Prose.\\n\"}\n\n", "Prose.\n", "", 0},
		{"empty stream", "", "", "", 0},
		{"comments and CRLF", ": keepalive\r\nevent: delta\r\ndata: {\"delta\":\"Prose.\"}\r\n\r\n", "Prose.\n", "", 0},
		{"multiline data", "event: delta\ndata: {\ndata: \"delta\":\"Prose.\"}\n\n", "Prose.\n", "", 0},
		{"bad token ignored", "event: delta\ndata: broken\n\nevent: delta\ndata: {\"delta\":\"Prose.\"}\n\n", "Prose.\n", "", 0},
		{"structured failure", "event: error\ndata: {\"message\":\"Capacity reached\"}\n\n", "", "error generating continuation: stream error: Capacity reached\n", 1},
		{"text failure", "event: error\ndata: Capacity reached\n\n", "", "error generating continuation: stream error: Capacity reached\n", 1},
		{"partial failure", "event: delta\ndata: {\"delta\":\"Partial\"}\n\nevent: error\ndata: Capacity reached\n\n", "Partial", "error generating continuation: stream error: Capacity reached\n", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/41/continue/stream" || r.Header.Get("Accept") != "text/event-stream" {
					t.Errorf("unexpected request: %s %s accept=%q", r.Method, r.URL.Path, r.Header.Get("Accept"))
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, tc.events)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			code := root.Execute([]string{"generate", "continue", "41"})
			if code != tc.code || out.String() != tc.output || errOut.String() != tc.diagnostic {
				t.Fatalf("code=%d stdout=%q stderr=%q; want %d %q %q", code, out.String(), errOut.String(), tc.code, tc.output, tc.diagnostic)
			}
		})
	}
}
