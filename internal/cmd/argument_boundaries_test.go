package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestArgumentCountErrorsPrecedeSideEffects(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"book", "delete"}, "book ID is required. Usage: prosie book delete <id> [flags]"},
		{[]string{"series", "attach", "1"}, "series ID and book ID are required. Usage: prosie series attach <series-id> <book-id> [flags]"},
		{[]string{"chat", "import", "1"}, "book ID and file path are required. Usage: prosie chat import <book-id> <file.json> [flags]"},
		{[]string{"chapter", "reorder", "1"}, "book ID and chapter order are required. Usage: prosie chapter reorder <book-id> <id1,id2,...> [flags]"},
		{[]string{"chat", "send"}, "book ID and message are required. Usage: prosie chat send <book-id> <message> [flags]"},
		{[]string{"chat", "stream", "1"}, "message is required. Usage: prosie chat stream <book-id> <message> [flags]"},
		{[]string{"chat", "send", "--conversation", "1"}, "message is required. Usage: prosie chat send <message> [flags]"},
		{[]string{"chat", "stream", "--conversation", "1"}, "message is required. Usage: prosie chat stream <message> [flags]"},
		{[]string{"book", "delete", "1", "2"}, "unexpected argument \"2\". Usage: prosie book delete <id> [flags]"},
		{[]string{"book", "show", "1", "--", "--json"}, "unexpected argument \"--json\". Usage: prosie book show <id> [flags]"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "test-token")
			cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), server.Client())
			cmd.In = failOnRead{t: t}
			code := cmd.Execute(tc.args)
			if code != 1 || out.Len() != 0 || errOut.String() != "error: "+tc.want+"\n" {
				t.Errorf("exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if got := requests.Load(); got != 0 {
				t.Errorf("sent %d API requests, want none", got)
			}
		})
	}
}

type failOnRead struct{ t *testing.T }

func (r failOnRead) Read([]byte) (int, error) {
	r.t.Error("command read standard input before rejecting argument count")
	return 0, io.EOF
}

func TestHelpPrecedesArgumentCounts(t *testing.T) {
	for _, args := range [][]string{
		{"book", "delete", "--help"},
		{"book", "delete", "1", "2", "--help"},
		{"series", "list", "extra", "--help"},
		{"llm", "show", "extra", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), nil)
			if code := cmd.Execute(args); code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "Usage:") {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
		})
	}
}

func TestChatAcceptsVariableArgumentCounts(t *testing.T) {
	for _, operation := range []string{"send", "stream"} {
		for _, existing := range []bool{false, true} {
			for _, jsonMode := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/existing=%t/json=%t", operation, existing, jsonMode), func(t *testing.T) {
					var content string
					var creates int
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						switch r.URL.Path {
						case "/api/stories/42/conversations":
							creates++
							w.WriteHeader(http.StatusCreated)
							_, _ = io.WriteString(w, `{"data":{"id":777,"story_id":42}}`)
						case "/api/conversations/777/messages", "/api/conversations/777/messages/stream":
							var body struct{ Content string }
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Errorf("invalid request body: %v", err)
							}
							content = body.Content
							if operation == "stream" {
								w.Header().Set("Content-Type", "text/event-stream")
								_, _ = io.WriteString(w, "event: delta\ndata: {\"delta\":\"Reply\"}\n\nevent: done\ndata: {\"message\":{\"id\":1,\"role\":\"assistant\",\"content\":\"Reply\"}}\n\n")
								return
							}
							_, _ = io.WriteString(w, `{"data":{"message":{"id":1,"role":"assistant","content":"Reply"}}}`)
						default:
							http.NotFound(w, r)
						}
					}))
					defer server.Close()
					t.Setenv("PROSIE_API_URL", server.URL)
					t.Setenv("PROSIE_API_TOKEN", "test-token")
					args := []string{"chat", operation, "42"}
					if existing {
						args = []string{"chat", operation, "--conversation", "777"}
					}
					args = append(args, "hello", "from", "the", "book")
					if jsonMode {
						args = append(args, "--json")
					}
					cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), server.Client())
					code := cmd.Execute(args)
					if code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "Reply") {
						t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
					}
					if content != "hello from the book" {
						t.Errorf("API received content %q", content)
					}
					wantCreates := 1
					if existing {
						wantCreates = 0
					}
					if creates != wantCreates {
						t.Errorf("created %d conversations, want %d", creates, wantCreates)
					}
				})
			}
		}
	}
}
