package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type outputFunc func([]byte) (int, error)

func (f outputFunc) Write(p []byte) (int, error) { return f(p) }

func TestCommandsReportOutputFailures(t *testing.T) {
	for _, tc := range []struct {
		args     string
		method   string
		path     string
		response string
	}{
		{"book list", "GET", "/api/stories", `{"data":[{"id":1,"title":"Book"}]}`},
		{"book show 1", "GET", "/api/stories/1", `{"data":{"id":1,"title":"Book","chapters":[{"id":1,"title":"Chapter"}]}}`},
		{"book delete 1 -y", "DELETE", "/api/stories/1", ""},
		{"chapter delete 1 -y", "DELETE", "/api/scenes/1", ""},
		{"chat delete 1 -y", "DELETE", "/api/conversations/1", ""},
		{"codex delete 1 -y", "DELETE", "/api/codex-entries/1", ""},
		{"series delete 1 -y", "DELETE", "/api/series/1", ""},
		{"chapter show 1", "GET", "/api/scenes/1", `{"data":{"id":1,"content":"Prose."}}`},
		{"chapter export 1", "GET", "/api/scenes/1/export", "Prose."},
		{"chat list 1", "GET", "/api/stories/1/conversations", `{"data":[{"id":1,"title":"Chat"}]}`},
		{"codex list 1", "GET", "/api/stories/1/codex-entries", `{"data":[{"id":1,"name":"Character"}]}`},
		{"series list", "GET", "/api/series", `{"data":[{"id":1,"name":"Series","books":[{"id":1,"title":"Book"}]}]}`},
		{"generate summarize 1", "POST", "/api/scenes/1/summarize", `{"data":{"summary":"Summary."}}`},
		{"auth status", "GET", "/api/user", `{"id":1,"name":"Writer","email":"writer@example.com"}`},
		{"version", "", "", ""},
		{"llm show", "GET", "/api/llm-config", `{"data":{"provider":"openai","model":"test-model"}}`},
		{"llm models", "GET", "/api/llm-models", `{"data":{"providers":{"openai":{"test-model":"Test"}},"configured":["openai"]}}`},
		{"llm update --provider openai", "PATCH", "/api/llm-config", `{"data":{"provider":"openai","model":"test-model"}}`},
	} {
		for _, mode := range []string{"human", "json"} {
			t.Run(tc.args+"/"+mode, func(t *testing.T) {
				configPath, httpClient := setupTestLlmEnv(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != tc.method || r.URL.Path != tc.path {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, tc.response)
				})
				root, _, errOut := newTestRootCmd(configPath, httpClient)
				root.Out = outputFunc(func([]byte) (int, error) {
					return 0, errors.New("output unavailable")
				})
				args := strings.Fields(tc.args)
				if mode == "json" {
					args = append(args, "--json")
				}
				code := root.Execute(args)
				if code != 1 || errOut.String() != "error writing output: output unavailable\n" {
					t.Fatalf("code=%d stderr=%q; want 1 and the output error", code, errOut.String())
				}
			})
		}
	}
}

func TestHelpReportsOutputFailures(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"--json"}, {"book", "--help"}, {"book", "list", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root, _, errOut := newTestRootCmd("", nil)
			root.Out = outputFunc(func([]byte) (int, error) { return 0, io.ErrClosedPipe })
			code := root.Execute(args)
			if code != 1 || errOut.String() != "error writing output: io: read/write on closed pipe\n" {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
		})
	}
}

func TestOutputFailuresPreserveHandlerErrors(t *testing.T) {
	configPath, httpClient := setupTestLlmEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/conversations/44/messages/stream" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: delta\ndata: {\"delta\":\"Partial prose\"}\n\nevent: error\ndata: Capacity reached\n\n")
	})
	root, _, errOut := newTestRootCmd(configPath, httpClient)
	root.Out = outputFunc(func([]byte) (int, error) { return 0, io.ErrClosedPipe })
	code := root.Execute([]string{"chat", "stream", "--conversation", "44", "Continue"})
	if code != 1 || errOut.String() != "error sending message: stream error: Capacity reached\n" {
		t.Fatalf("code=%d stderr=%q; want the handler error only", code, errOut.String())
	}
}

func TestOutputFailureStateIsPerExecution(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"failed write", io.ErrClosedPipe, "error writing output: io: read/write on closed pipe\n"},
		{"short write", nil, "error writing output: short write\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, out, errOut := newTestRootCmd("", nil)
			fail := true
			root.Out = outputFunc(func(p []byte) (int, error) {
				if fail {
					return len(p) / 2, tc.err
				}
				return out.Write(p)
			})
			code := root.Execute([]string{"version"})
			if code != 1 || errOut.String() != tc.want {
				t.Fatalf("code=%d stderr=%q; want 1 %q", code, errOut.String(), tc.want)
			}
			fail = false
			errOut.Reset()
			code = root.Execute([]string{"version"})
			if code != 0 || out.String() != "prosie version 0.0.0-dev\n" || errOut.Len() != 0 {
				t.Fatalf("second execution: code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
		})
	}
}

func TestOutputFailureKeepsTheFirstError(t *testing.T) {
	configPath, httpClient := setupTestLlmEnv(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":1,"title":"Book","chapters":[{"id":1,"title":"Chapter"}]}}`)
	})
	root, _, errOut := newTestRootCmd(configPath, httpClient)
	calls := 0
	root.Out = outputFunc(func([]byte) (int, error) {
		calls++
		if calls == 1 {
			return 0, errors.New("first failure")
		}
		return 0, errors.New("later failure")
	})
	code := root.Execute([]string{"book", "show", "1"})
	if code != 1 || errOut.String() != "error writing output: first failure\n" {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}
}
