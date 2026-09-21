package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestContinue(t *testing.T) {
	t.Run("persist true", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/101/continue" {
				http.NotFound(w, r)
				return
			}

			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["persist"] != true {
				t.Errorf("expected persist: true, got %v", body["persist"])
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"prose":     "The engine hummed to life.",
					"html":      "<p>The engine hummed to life.</p>",
					"persisted": true,
					"model":     "test-model",
					"usage": map[string]int{
						"prompt_tokens":     15,
						"completion_tokens": 6,
						"total_tokens":      21,
					},
					"estimated_prompt_tokens": 15,
				},
			})
		}))
		defer server.Close()

		cli := New(server.URL, "test-token", server.Client())
		res, err := cli.Generation().Continue(context.Background(), "101", ContinueParams{Persist: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Prose != "The engine hummed to life." {
			t.Errorf("unexpected prose: %q", res.Prose)
		}
		if !res.Persisted {
			t.Errorf("expected persisted true")
		}
		if res.Usage == nil || res.Usage.TotalTokens != 21 {
			t.Errorf("unexpected usage: %+v", res.Usage)
		}
	})

	t.Run("persist false", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["persist"] != false {
				t.Errorf("expected persist: false, got %v", body["persist"])
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"prose":     "Draft prose without saving.",
					"persisted": false,
				},
			})
		}))
		defer server.Close()

		cli := New(server.URL, "test-token", server.Client())
		res, err := cli.Generation().Continue(context.Background(), "101", ContinueParams{Persist: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Prose != "Draft prose without saving." {
			t.Errorf("unexpected prose: %q", res.Prose)
		}
		if res.Persisted {
			t.Errorf("expected persisted false")
		}
	})

	t.Run("guidance fields are forwarded and zero values omitted", func(t *testing.T) {
		var got map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"prose": "Fin.", "persisted": true}})
		}))
		defer server.Close()

		cli := New(server.URL, "test-token", server.Client())
		_, err := cli.Generation().Continue(context.Background(), "101", ContinueParams{
			Persist:     true,
			Instruction: "Close the story.",
			WordTarget:  750,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["instruction"] != "Close the story." {
			t.Errorf("expected instruction forwarded, got %v", got)
		}
		if got["word_target"] != float64(750) {
			t.Errorf("expected word_target 750, got %v", got)
		}
		if _, present := got["line_limit"]; present {
			t.Errorf("expected line_limit omitted when zero, got %v", got)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})
		}))
		defer server.Close()

		cli := New(server.URL, "test-token", server.Client())
		_, err := cli.Generation().Continue(context.Background(), "999", ContinueParams{Persist: true})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Scene not found.") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestStreamContinue(t *testing.T) {
	t.Run("successful stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/scenes/101/continue/stream" {
				http.NotFound(w, r)
				return
			}
			if r.Header.Get("Accept") != "text/event-stream" {
				t.Errorf("expected Accept text/event-stream, got %q", r.Header.Get("Accept"))
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected http.Flusher")
			}

			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"The reactor \"}\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"stabilized.\"}\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("event: done\ndata: {\"prose\":\"The reactor stabilized.\",\"persisted\":true,\"model\":\"gpt-4\"}\n\n"))
			flusher.Flush()
		}))
		defer server.Close()

		cli := New(server.URL, "test-token", server.Client())
		var tokens []string
		res, err := cli.Generation().StreamContinue(context.Background(), "101", ContinueParams{Persist: true}, func(tok string) {
			tokens = append(tokens, tok)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Join(tokens, "") != "The reactor stabilized." {
			t.Errorf("unexpected collected tokens: %v", tokens)
		}
		if res.Prose != "The reactor stabilized." || !res.Persisted {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("stream error event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("event: error\ndata: {\"message\":\"Quota exceeded\"}\n\n"))
		}))
		defer server.Close()

		cli := New(server.URL, "test-token", server.Client())
		_, err := cli.Generation().StreamContinue(context.Background(), "101", ContinueParams{Persist: true}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Quota exceeded") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("stream context cancellation", func(t *testing.T) {
		releaseServer := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"Start...\"}\n\n"))
				f.Flush()
			}
			<-releaseServer
		}))
		defer func() {
			close(releaseServer)
			server.Close()
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cli := New(server.URL, "test-token", server.Client())

		var wg sync.WaitGroup
		wg.Add(1)
		var streamErr error

		go func() {
			defer wg.Done()
			_, streamErr = cli.Generation().StreamContinue(ctx, "101", ContinueParams{Persist: true}, func(tok string) {
				cancel() // Cancel when first token arrives
			})
		}()

		wg.Wait()
		if streamErr == nil {
			t.Fatal("expected error upon context cancellation, got nil")
		}
		if streamErr != context.Canceled && !strings.Contains(streamErr.Error(), "context canceled") {
			t.Errorf("expected context canceled error, got: %v", streamErr)
		}
	})
}

func TestCancelContinue(t *testing.T) {
	cancelled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/scenes/101/continue/cancel" {
			cancelled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	err := cli.Generation().CancelContinue(context.Background(), "101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cancelled {
		t.Error("expected server to receive cancel request")
	}
}

func TestRejectContinuation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/101/reject-continuation" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         101,
				"story_id":   1,
				"order":      0,
				"name":       "Restored Chapter",
				"content":    "Only the author prose remains.",
				"word_count": 5,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	chapter, err := cli.Generation().RejectContinuation(context.Background(), "101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chapter.ID != 101 || chapter.Content != "Only the author prose remains." {
		t.Errorf("unexpected restored chapter: %+v", chapter)
	}
}

func TestUndoRewrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/101/rewrite/undo" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         101,
				"story_id":   1,
				"order":      0,
				"name":       "Restored Chapter",
				"content":    "Text before the rewrite.",
				"word_count": 4,
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	chapter, err := cli.Generation().UndoRewrite(context.Background(), "101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chapter.ID != 101 || chapter.Content != "Text before the rewrite." {
		t.Errorf("unexpected restored chapter: %+v", chapter)
	}
}

func TestRewrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/101/rewrite" {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["selection"] != "He ran quickly." {
			t.Errorf("unexpected selection: %v", body["selection"])
		}
		if body["action"] != "tighten" {
			t.Errorf("unexpected action: %v", body["action"])
		}
		if body["persist"] != true {
			t.Errorf("unexpected persist: %v", body["persist"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"prose":           "He sprinted.",
				"persisted":       true,
				"applied_content": "<p>He sprinted.</p>",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	params := RewriteParams{
		Selection: "He ran quickly.",
		Action:    "tighten",
		Persist:   true,
	}
	res, err := cli.Generation().Rewrite(context.Background(), "101", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Prose != "He sprinted." || !res.Persisted {
		t.Errorf("unexpected rewrite result: %+v", res)
	}
}

func TestStreamRewrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/scenes/101/rewrite/stream" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if f, ok := w.(http.Flusher); ok {
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"He \"}\n\n"))
			f.Flush()
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"dashed.\"}\n\n"))
			f.Flush()
			_, _ = w.Write([]byte("event: done\ndata: {\"prose\":\"He dashed.\",\"persisted\":false}\n\n"))
			f.Flush()
		}
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	var tokens []string
	params := RewriteParams{
		Selection:   "He walked away fast.",
		Instruction: "Make it more active",
		Persist:     false,
	}
	res, err := cli.Generation().StreamRewrite(context.Background(), "101", params, func(tok string) {
		tokens = append(tokens, tok)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(tokens, "") != "He dashed." {
		t.Errorf("unexpected tokens: %v", tokens)
	}
	if res.Prose != "He dashed." || res.Persisted {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestSummarize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/101/summarize" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"summary":   "The crew arrives at Station Alpha.",
				"persisted": true,
				"model":     "test-model",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	res, err := cli.Generation().Summarize(context.Background(), "101", SummarizeParams{Persist: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Summary != "The crew arrives at Station Alpha." || !res.Persisted {
		t.Errorf("unexpected summary result: %+v", res)
	}
}
