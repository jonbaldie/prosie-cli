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
	name string
	path string
	run  func(*Client, context.Context, func(string)) (string, error)
}

func streamTestOperations() []streamTestOperation {
	return []streamTestOperation{
		{
			name: "continue",
			path: "/api/scenes/101/continue/stream",
			run: func(cli *Client, ctx context.Context, onToken func(string)) (string, error) {
				result, err := cli.Generation().StreamContinue(ctx, "101", ContinueParams{}, onToken)
				if result == nil {
					return "", err
				}
				return result.Prose, err
			},
		},
		{
			name: "rewrite",
			path: "/api/scenes/101/rewrite/stream",
			run: func(cli *Client, ctx context.Context, onToken func(string)) (string, error) {
				result, err := cli.Generation().StreamRewrite(ctx, "101", RewriteParams{Selection: "original"}, onToken)
				if result == nil {
					return "", err
				}
				return result.Prose, err
			},
		},
		{
			name: "chat",
			path: "/api/conversations/44/messages/stream",
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
				_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":%q}", finalToken)
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
				_, _ = fmt.Fprint(w, "event: delta\ndata: {\"delta\":\"\"}\n\nevent: delta\ndata: {\"delta\":\"next\"}\n\n")
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
