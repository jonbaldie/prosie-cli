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

func TestIncompleteStreamsExitWithAnError(t *testing.T) {
	const partial = "event: delta\ndata: {\"delta\":\"Partial prose\"}\n\n"
	const wantErr = "stream error: stream ended before completion"
	for _, tc := range []struct {
		name, path, events, args, stdout, stderrPrefix string
	}{
		{
			name:         "continue",
			path:         "/api/scenes/103/continue/stream",
			events:       partial,
			args:         "generate continue 103",
			stdout:       "Partial prose",
			stderrPrefix: "error generating continuation: " + wantErr + "\n",
		},
		{
			name:         "rewrite",
			path:         "/api/scenes/103/rewrite/stream",
			events:       partial,
			args:         "generate rewrite 103 --selection Original --prompt Tense. --stream --persist",
			stdout:       "Partial prose",
			stderrPrefix: "error rewriting text: " + wantErr + "\n",
		},
		{
			name:         "chat",
			path:         "/api/conversations/44/messages/stream",
			events:       partial,
			args:         "chat stream --conversation 44 Continue",
			stdout:       "Partial prose\n",
			stderrPrefix: "error sending message: " + wantErr + "\n",
		},
		{
			name:         "chat json",
			path:         "/api/conversations/44/messages/stream",
			events:       partial,
			args:         "chat stream --conversation 44 Continue --json",
			stdout:       "",
			stderrPrefix: "error sending message: " + wantErr + "\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					http.NotFound(w, r)
					return
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
			code := root.Execute(splitArgs(tc.args))
			if code != 1 || out.String() != tc.stdout || errOut.String() != tc.stderrPrefix {
				t.Fatalf("code=%d stdout=%q stderr=%q; want 1 %q %q", code, out.String(), errOut.String(), tc.stdout, tc.stderrPrefix)
			}
		})
	}
}

func splitArgs(line string) []string {
	return strings.Fields(line)
}

func TestContinuationStreamsCompleteTextAndReportsErrors(t *testing.T) {
	for _, tc := range []struct {
		name, events, output, diagnostic string
		code                             int
	}{
		{"tokens", "event: delta\ndata: {\"delta\":\"Café \"}\n\nevent: delta\ndata: {\"delta\":\"opened.\"}\n\nevent: done\ndata: {\"prose\":\"Café opened.\"}\n\n", "Café opened.\n", "", 0},
		{"existing newline", "event: delta\ndata: {\"delta\":\"Prose.\\n\"}\n\nevent: done\ndata: {\"prose\":\"Prose.\\n\"}\n\n", "Prose.\n", "", 0},
		{"empty stream", "", "", "error generating continuation: stream error: stream ended before completion\n", 1},
		{"comments and CRLF", ": keepalive\r\nevent: delta\r\ndata: {\"delta\":\"Prose.\"}\r\n\r\nevent: done\r\ndata: {\"prose\":\"Prose.\"}\r\n\r\n", "Prose.\n", "", 0},
		{"multiline data", "event: delta\ndata: {\ndata: \"delta\":\"Prose.\"}\n\nevent: done\ndata: {\"prose\":\"Prose.\"}\n\n", "Prose.\n", "", 0},
		{"bad token ignored", "event: delta\ndata: broken\n\nevent: delta\ndata: {\"delta\":\"Prose.\"}\n\nevent: done\ndata: {\"prose\":\"Prose.\"}\n\n", "Prose.\n", "", 0},
		{"done without newline", "event: delta\ndata: {\"delta\":\"Prose.\"}\n\nevent: done\ndata: {\"prose\":\"Prose.\"}", "Prose.\n", "", 0},
		{"empty prose done", "event: done\ndata: {\"prose\":\"\"}\n\n", "", "", 0},
		{"invalid done", "event: delta\ndata: {\"delta\":\"Partial\"}\n\nevent: done\ndata: {not-json}\n\n", "Partial", "error generating continuation: stream error: stream ended before completion\n", 1},
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
