package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type streamTestOperation struct {
	name       string
	path       string
	completion func(string) string
	run        func(*Client, context.Context, func(string)) (string, error)
}

func proseCompletion(text string) string {
	return fmt.Sprintf("{\"prose\":%q}", text)
}

func chatCompletion(text string) string {
	return fmt.Sprintf("{\"message\":{\"role\":\"assistant\",\"content\":%q}}", text)
}

func streamTestOperations() []streamTestOperation {
	return []streamTestOperation{
		{
			name:       "continue",
			path:       "/api/scenes/101/continue/stream",
			completion: proseCompletion,
			run: func(cli *Client, ctx context.Context, onToken func(string)) (string, error) {
				result, err := cli.Generation().StreamContinue(ctx, "101", ContinueParams{}, onToken)
				if result == nil {
					return "", err
				}
				return result.Prose, err
			},
		},
		{
			name:       "rewrite",
			path:       "/api/scenes/101/rewrite/stream",
			completion: proseCompletion,
			run: func(cli *Client, ctx context.Context, onToken func(string)) (string, error) {
				result, err := cli.Generation().StreamRewrite(ctx, "101", RewriteParams{Selection: "original"}, onToken)
				if result == nil {
					return "", err
				}
				return result.Prose, err
			},
		},
		{
			name:       "chat",
			path:       "/api/conversations/44/messages/stream",
			completion: chatCompletion,
			run: func(cli *Client, ctx context.Context, onToken func(string)) (string, error) {
				result, err := cli.Conversations().StreamChatMessage(ctx, "44", "hello", onToken)
				if result == nil || result.Message == nil {
					return "", err
				}
				return result.Message.Content, err
			},
		},
	}
}

func TestStreamEndpointsFlushFinalEventAtEOF(t *testing.T) {
	const finalToken = "final token"
	for _, operation := range streamTestOperations() {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != operation.path {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":%q}\n\nevent: done\ndata: %s", finalToken, operation.completion(finalToken))
			}))
			defer server.Close()

			cli := New(server.URL, "test-token", server.Client())
			var tokens []string
			result, err := operation.run(cli, context.Background(), func(token string) {
				tokens = append(tokens, token)
			})
			if err != nil {
				t.Fatalf("stream returned error: %v", err)
			}
			if got := strings.Join(tokens, ""); got != finalToken {
				t.Errorf("received tokens %q, want %q", got, finalToken)
			}
			if result != finalToken {
				t.Errorf("result text %q, want %q", result, finalToken)
			}
		})
	}
}

func TestStreamEndpointsIgnoreEmptyDeltas(t *testing.T) {
	for _, operation := range streamTestOperations() {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != operation.path {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"\"}\n\nevent: delta\ndata: {\"delta\":\"next\"}\n\nevent: done\ndata: %s\n\n", operation.completion("next"))
			}))
			defer server.Close()

			cli := New(server.URL, "test-token", server.Client())
			var tokens []string
			_, err := operation.run(cli, context.Background(), func(token string) {
				tokens = append(tokens, token)
			})
			if err != nil {
				t.Fatalf("stream returned error: %v", err)
			}
			if len(tokens) != 1 || tokens[0] != "next" {
				t.Errorf("received tokens %q, want [\"next\"]", tokens)
			}
		})
	}
}

func TestStreamEndpointsUseTheSameStreamError(t *testing.T) {
	payloads := []struct {
		name string
		data string
		want string
	}{
		{name: "message payload", data: `{"message":"Quota exceeded"}`, want: "stream error: Quota exceeded"},
		{name: "raw payload", data: "Rate limit exceeded", want: "stream error: Rate limit exceeded"},
	}
	for _, operation := range streamTestOperations() {
		operation := operation
		for _, payload := range payloads {
			payload := payload
			t.Run(operation.name+"/"+payload.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.URL.Path != operation.path {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload.data)
				}))
				defer server.Close()

				cli := New(server.URL, "test-token", server.Client())
				_, err := operation.run(cli, context.Background(), nil)
				if err == nil {
					t.Fatal("stream returned no error")
				}
				if err.Error() != payload.want {
					t.Errorf("stream error %q, want %q", err, payload.want)
				}
			})
		}
	}
}

func TestStreamEndpointsReturnContextCancellation(t *testing.T) {
	for _, operation := range streamTestOperations() {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != operation.path {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				flusher, ok := w.(http.Flusher)
				if !ok {
					t.Error("response writer does not support flushing")
					return
				}
				_, _ = fmt.Fprint(w, "event: delta\ndata: {\"delta\":\"start\"}\n\n")
				flusher.Flush()
				<-r.Context().Done()
			}))
			defer server.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cli := New(server.URL, "test-token", server.Client())
			errCh := make(chan error, 1)
			go func() {
				_, err := operation.run(cli, ctx, func(string) { cancel() })
				errCh <- err
			}()

			select {
			case err := <-errCh:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("stream error %v, want context.Canceled", err)
				}
			case <-time.After(time.Second):
				t.Fatal("stream did not stop after context cancellation")
			}
		})
	}
}

func TestStreamEndpointsRejectIncompleteResponses(t *testing.T) {
	bodies := []struct {
		name   string
		body   string
		tokens []string
	}{
		{name: "delta only", body: "event: delta\ndata: {\"delta\":\"Partial\"}\n\n", tokens: []string{"Partial"}},
		{name: "empty", body: ""},
		{name: "invalid done", body: "event: delta\ndata: {\"delta\":\"Partial\"}\n\nevent: done\ndata: {not-json}\n\n", tokens: []string{"Partial"}},
	}
	const want = "stream error: stream ended before completion"
	for _, operation := range streamTestOperations() {
		operation := operation
		for _, body := range bodies {
			body := body
			t.Run(operation.name+"/"+body.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != operation.path {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, body.body)
				}))
				defer server.Close()

				cli := New(server.URL, "test-token", server.Client())
				var tokens []string
				text, err := operation.run(cli, context.Background(), func(token string) {
					tokens = append(tokens, token)
				})
				if err == nil || err.Error() != want {
					t.Fatalf("stream error %v, want %q", err, want)
				}
				if text != "" {
					t.Fatalf("success text %q, want no result", text)
				}
				if strings.Join(tokens, "") != strings.Join(body.tokens, "") {
					t.Fatalf("tokens %q, want %q", tokens, body.tokens)
				}
			})
		}
	}
}

func TestIncompleteStreamDoesNotInferPersistence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: delta\ndata: {\"delta\":\"Partial\"}\n\n")
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	continued, err := cli.Generation().StreamContinue(context.Background(), "101", ContinueParams{Persist: true}, nil)
	if err == nil || continued != nil {
		t.Fatalf("continue result=%+v err=%v, want nil result and an error", continued, err)
	}
	rewritten, err := cli.Generation().StreamRewrite(context.Background(), "101", RewriteParams{Selection: "original", Persist: true}, nil)
	if err == nil || rewritten != nil {
		t.Fatalf("rewrite result=%+v err=%v, want nil result and an error", rewritten, err)
	}
}

func TestCompletedStreamKeepsServerResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		switch r.URL.Path {
		case "/api/scenes/101/continue/stream":
			fmt.Fprint(w, "event: done\ndata: {\"prose\":\"\",\"persisted\":false}\n\n")
		case "/api/scenes/101/rewrite/stream":
			fmt.Fprint(w, "event: done\ndata: {\"prose\":\"Kept.\",\"persisted\":true}")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	continued, err := cli.Generation().StreamContinue(context.Background(), "101", ContinueParams{Persist: true}, nil)
	if err != nil {
		t.Fatalf("empty prose completion returned error: %v", err)
	}
	if continued == nil || continued.Prose != "" || continued.Persisted {
		t.Fatalf("continue result %+v, want empty prose and server persisted false", continued)
	}

	rewritten, err := cli.Generation().StreamRewrite(context.Background(), "101", RewriteParams{Selection: "original", Persist: false}, nil)
	if err != nil {
		t.Fatalf("rewrite completion returned error: %v", err)
	}
	if rewritten == nil || rewritten.Prose != "Kept." || !rewritten.Persisted {
		t.Fatalf("rewrite result %+v, want server prose and persisted true", rewritten)
	}
}
