package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCommandsUseRequestContext(t *testing.T) {
	for _, command := range []struct {
		name     string
		args     []string
		requests int32
		output   string
	}{
		{"book list", []string{"book", "list", "--json"}, 1, `"title": "Orbit"`},
		{"chat send new conversation", []string{"chat", "send", "1", "Hello", "--json"}, 2, `"content": "Ready."`},
		{"chat send existing conversation", []string{"chat", "send", "--conversation", "7", "Hello", "--json"}, 1, `"content": "Ready."`},
		{"chat stream", []string{"chat", "stream", "--conversation", "7", "Hello"}, 1, "Ready.\n"},
	} {
		for _, state := range []string{"nil", "active", "cancelled", "expired deadline"} {
			t.Run(command.name+"/"+state, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					w.Header().Set("Content-Type", "application/json")
					switch r.Method + " " + r.URL.Path {
					case "GET /api/stories":
						fmt.Fprint(w, `{"data":[{"id":1,"title":"Orbit"}]}`)
					case "POST /api/stories/1/conversations":
						w.WriteHeader(http.StatusCreated)
						fmt.Fprint(w, `{"data":{"id":7,"story_id":1}}`)
					case "POST /api/conversations/7/messages":
						fmt.Fprint(w, `{"data":{"message":{"id":1,"role":"assistant","content":"Ready."}}}`)
					case "POST /api/conversations/7/messages/stream":
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprint(w, "event: delta\ndata: {\"delta\":\"Ready.\"}\n\nevent: done\ndata: {\"message\":{\"role\":\"assistant\",\"content\":\"Ready.\"}}\n\n")
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				t.Setenv("PROSIE_API_URL", server.URL)
				t.Setenv("PROSIE_API_TOKEN", "test-token")

				cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), server.Client())
				wantError := ""
				switch state {
				case "active":
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					cmd.Context = ctx
				case "cancelled":
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					cmd.Context = ctx
					wantError = "context canceled"
				case "expired deadline":
					ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					defer cancel()
					cmd.Context = ctx
					wantError = "context deadline exceeded"
				}

				code := cmd.Execute(command.args)
				if wantError != "" {
					if code != 1 || !strings.Contains(errOut.String(), wantError) || out.Len() != 0 || requests.Load() != 0 {
						t.Fatalf("code=%d stdout=%q stderr=%q requests=%d; want code=1, %q on stderr, no stdout or requests", code, out.String(), errOut.String(), requests.Load(), wantError)
					}
					return
				}
				if code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), command.output) || requests.Load() != command.requests {
					t.Fatalf("code=%d stdout=%q stderr=%q requests=%d; want code=0, output containing %q, no stderr, requests=%d", code, out.String(), errOut.String(), requests.Load(), command.output, command.requests)
				}
			})
		}
	}
}

func TestCommandsCancelActiveRequest(t *testing.T) {
	for _, command := range []struct {
		name     string
		args     []string
		requests int32
		hint     string
	}{
		{"book list", []string{"book", "list", "--json"}, 1, "error listing books:"},
		{"chat send after conversation creation", []string{"chat", "send", "1", "Hello", "--json"}, 2, "conversation 7 was created; continue with --conversation 7"},
	} {
		t.Run(command.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					t.Error(err)
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/stories/1/conversations" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"data":{"id":7,"story_id":1}}`)
					return
				}
				cancel()
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
					t.Error("the request did not stop after caller cancellation")
				}
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "test-token")
			cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), server.Client())
			cmd.Context = ctx
			code := cmd.Execute(command.args)
			if code != 1 || out.Len() != 0 || !strings.Contains(errOut.String(), "context canceled") || !strings.Contains(errOut.String(), command.hint) || requests.Load() != command.requests {
				t.Fatalf("code=%d stdout=%q stderr=%q requests=%d; want code=1, cancellation and %q on stderr, no stdout, requests=%d", code, out.String(), errOut.String(), requests.Load(), command.hint, command.requests)
			}
		})
	}
}
