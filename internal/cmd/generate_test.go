package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func setupTestGenerateEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-generate-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-generate-token")

	return server, configPath, server.Client()
}

func TestGenerateHelp(t *testing.T) {
	cmd, out, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"generate", "--help"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Manage AI prose generation on Prosie.") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
	if !strings.Contains(out.String(), "undo        Undo latest AI rewrite on a chapter") {
		t.Fatalf("help output does not list undo: %s", out.String())
	}
}

func TestGenerateUnknownSubcommand(t *testing.T) {
	cmd, _, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"generate", "invalid"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown generate command: invalid") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestGenerateContinue(t *testing.T) {
	var cancelCalled bool
	var mu sync.Mutex

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scenes/101/continue/stream" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "flusher error", 500)
				return
			}

			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"The stars \"}\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"burned brightly.\"}\n\n"))
			flusher.Flush()
			doneJSON, _ := json.Marshal(map[string]any{
				"prose":     "The stars burned brightly.",
				"persisted": body["persist"] == true,
				"model":     "test-model",
			})
			_, _ = w.Write([]byte("event: done\ndata: " + string(doneJSON) + "\n\n"))
			flusher.Flush()

		case r.URL.Path == "/api/scenes/101/continue" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"prose":     "The stars burned brightly.",
					"persisted": body["persist"] == true,
					"model":     "test-model",
					"usage": map[string]int{
						"total_tokens": 42,
					},
				},
			})

		case r.URL.Path == "/api/scenes/101/continue/cancel" && r.Method == http.MethodPost:
			mu.Lock()
			cancelCalled = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)

		case r.URL.Path == "/api/scenes/404/continue/stream" || r.URL.Path == "/api/scenes/404/continue":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})

		case r.URL.Path == "/api/scenes/500/continue/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("event: error\ndata: {\"message\":\"LLM capacity limit reached\"}\n\n"))

		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestGenerateEnv(t, handler)

	t.Run("missing chapter id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "--help"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Generate AI prose continuation for a chapter.") {
			t.Fatalf("unexpected help text: %s", out.String())
		}
	})

	t.Run("streaming default", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The stars burned brightly.") {
			t.Fatalf("unexpected stream output: %s", out.String())
		}
	})

	t.Run("streaming with --no-persist", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "101", "--no-persist"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The stars burned brightly.") {
			t.Fatalf("unexpected stream output: %s", out.String())
		}
	})

	t.Run("non-streaming with --no-stream", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "101", "--no-stream"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The stars burned brightly.") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})

	t.Run("non-streaming with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res client.ContinueResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res.Prose != "The stars burned brightly." || !res.Persisted {
			t.Fatalf("unexpected json result: %+v", res)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Scene not found.") {
			t.Fatalf("expected Scene not found error, got: %s", errOut.String())
		}
	})

	t.Run("sse error event", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "continue", "500"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "LLM capacity limit reached") {
			t.Fatalf("expected capacity limit error, got: %s", errOut.String())
		}
	})

	t.Run("cancellation cleanly cancels on server", func(t *testing.T) {
		slowHandler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/scenes/101/continue/stream" {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				if f, ok := w.(http.Flusher); ok {
					_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"Initial token...\"}\n\n"))
					f.Flush()
				}
				// Hang until client terminates
				select {
				case <-r.Context().Done():
					return
				case <-time.After(2 * time.Second):
					return
				}
			}
			if r.URL.Path == "/api/scenes/101/continue/cancel" {
				mu.Lock()
				cancelCalled = true
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.NotFound(w, r)
		}
		_, slowCfg, slowClient := setupTestGenerateEnv(t, slowHandler)

		ctx, cancel := context.WithCancel(context.Background())
		cmd, _, errOut := newTestRootCmd(slowCfg, slowClient)
		cmd.Context = ctx

		mu.Lock()
		cancelCalled = false
		mu.Unlock()

		done := make(chan int)
		go func() {
			done <- cmd.Execute([]string{"generate", "continue", "101"})
		}()

		// Give the stream a moment to connect and send initial token, then cancel context
		time.Sleep(50 * time.Millisecond)
		cancel()

		code := <-done
		if code != 1 {
			t.Fatalf("expected code 1 for cancelled stream, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Generation cancelled.") {
			t.Fatalf("expected 'Generation cancelled.' in stderr, got: %s", errOut.String())
		}

		mu.Lock()
		called := cancelCalled
		mu.Unlock()
		if !called {
			t.Fatal("expected CancelContinue to be invoked on server")
		}
	})
}

func TestGenerateContinueGuidanceFlags(t *testing.T) {
	var mu sync.Mutex
	var bodies []map[string]any

	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()

		switch r.URL.Path {
		case "/api/scenes/101/continue/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"The end.\"}\n\nevent: done\ndata: {\"prose\":\"The end.\",\"persisted\":true}\n\n"))
		case "/api/scenes/101/continue":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"prose": "The end.", "persisted": true}})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestGenerateEnv(t, handler)

	lastBody := func(t *testing.T) map[string]any {
		mu.Lock()
		defer mu.Unlock()
		if len(bodies) == 0 {
			t.Fatal("expected a request body to be captured")
		}
		return bodies[len(bodies)-1]
	}

	t.Run("default sends no guidance fields", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		if code := cmd.Execute([]string{"generate", "continue", "101", "--no-stream"}); code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		body := lastBody(t)
		for _, key := range []string{"instruction", "word_target", "line_limit"} {
			if _, present := body[key]; present {
				t.Fatalf("expected %q to be absent by default, body: %v", key, body)
			}
		}
	})

	t.Run("non-streaming forwards instruction, words and lines", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "continue", "101", "--no-stream",
			"--instruction", "Write the final scene. Resolve every open thread.",
			"--words", "900",
			"--lines", "40",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The end.") {
			t.Fatalf("unexpected output: %s", out.String())
		}
		body := lastBody(t)
		if body["instruction"] != "Write the final scene. Resolve every open thread." {
			t.Fatalf("instruction not forwarded, body: %v", body)
		}
		if body["word_target"] != float64(900) {
			t.Fatalf("word_target not forwarded, body: %v", body)
		}
		if body["line_limit"] != float64(40) {
			t.Fatalf("line_limit not forwarded, body: %v", body)
		}
		if body["persist"] != true {
			t.Fatalf("persist default lost, body: %v", body)
		}
	})

	t.Run("streaming forwards instruction and words", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "continue", "101",
			"--instruction", "Bring the story to a close.",
			"--words", "1000",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The end.") {
			t.Fatalf("unexpected stream output: %s", out.String())
		}
		body := lastBody(t)
		if body["instruction"] != "Bring the story to a close." {
			t.Fatalf("instruction not forwarded on stream, body: %v", body)
		}
		if body["word_target"] != float64(1000) {
			t.Fatalf("word_target not forwarded on stream, body: %v", body)
		}
		if _, present := body["line_limit"]; present {
			t.Fatalf("line_limit should be absent when not set, body: %v", body)
		}
	})

	t.Run("help documents the guidance flags", func(t *testing.T) {
		cmd, out, _ := newTestRootCmd(cfgPath, httpClient)
		if code := cmd.Execute([]string{"generate", "continue", "--help"}); code != 0 {
			t.Fatalf("expected code 0, got %d", code)
		}
		for _, flag := range []string{"--instruction", "--words", "--lines"} {
			if !strings.Contains(out.String(), flag) {
				t.Fatalf("help missing %s: %s", flag, out.String())
			}
		}
	})

	t.Run("rejects non-positive words", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		if code := cmd.Execute([]string{"generate", "continue", "101", "--words", "0"}); code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--words must be a positive integer") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})
}

func TestGenerateReject(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scenes/101/reject-continuation" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":      101,
					"name":    "Chapter One",
					"content": "Original user text.",
				},
			})
		case r.URL.Path == "/api/scenes/404/reject-continuation":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestGenerateEnv(t, handler)

	t.Run("missing chapter id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "reject"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "reject", "--help"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Revert the latest AI continuation on a chapter.") {
			t.Fatalf("unexpected help text: %s", out.String())
		}
	})

	t.Run("plain text output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "reject", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Reverted latest continuation for chapter 101.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "reject", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var ch client.Chapter
		if err := json.Unmarshal(out.Bytes(), &ch); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if ch.ID != 101 || ch.Content != "Original user text." {
			t.Fatalf("unexpected chapter json: %+v", ch)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "reject", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Scene not found.") {
			t.Fatalf("expected 404 message: %s", errOut.String())
		}
	})
}

func TestGenerateUndo(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scenes/101/rewrite/undo" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":      101,
					"name":    "Chapter One",
					"content": "Text before the rewrite.",
				},
			})
		case r.URL.Path == "/api/scenes/202/rewrite/undo" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "No rewrite undo baseline found for this scene.",
				"errors": map[string][]string{
					"scene": {"No rewrite undo baseline found for this scene."},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestGenerateEnv(t, handler)

	t.Run("missing chapter id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "undo"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "undo", "--help"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Undo the latest AI rewrite on a chapter.") {
			t.Fatalf("unexpected help text: %s", out.String())
		}
	})

	t.Run("plain text output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "undo", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Undid latest rewrite for chapter 101.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "undo", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var ch client.Chapter
		if err := json.Unmarshal(out.Bytes(), &ch); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if ch.ID != 101 || ch.Content != "Text before the rewrite." {
			t.Fatalf("unexpected chapter json: %+v", ch)
		}
	})

	t.Run("no undo baseline", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "undo", "202"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "No rewrite undo baseline found for this scene.") {
			t.Fatalf("expected baseline message on stderr: %s", errOut.String())
		}
		if out.Len() != 0 {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})
}

func TestGenerateRewrite(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scenes/101/rewrite/stream" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "flusher error", 500)
				return
			}
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"She \"}\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"whispered.\"}\n\n"))
			flusher.Flush()
			_, _ = w.Write([]byte("event: done\ndata: {\"prose\":\"She whispered.\",\"persisted\":false}\n\n"))
			flusher.Flush()

		case r.URL.Path == "/api/scenes/101/rewrite" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"prose":           "She whispered.",
					"persisted":       body["persist"] == true,
					"model":           "test-model",
					"applied_content": "<p>She whispered.</p>",
				},
			})

		case r.URL.Path == "/api/scenes/404/rewrite":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})

		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestGenerateEnv(t, handler)

	t.Run("missing chapter id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "rewrite"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("missing selection", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "rewrite", "101", "--prompt", "tighten"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--selection or --selection-file is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("missing prompt and action", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "rewrite", "101", "--selection", "She said quietly."})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "--prompt or --action is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "rewrite", "--help"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Rewrite selected chapter prose using a prompt or preset action.") {
			t.Fatalf("unexpected help text: %s", out.String())
		}
	})

	t.Run("help cites real preset action keys", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		if code := cmd.Execute([]string{"generate", "rewrite", "--help"}); code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if strings.Contains(out.String(), "show-not-tell") {
			t.Fatalf("help cites nonexistent preset key show-not-tell: %s", out.String())
		}
		for _, key := range []string{"show", "tighten", "voice", "user-<id>"} {
			if !strings.Contains(out.String(), key) {
				t.Fatalf("help missing preset key %s: %s", key, out.String())
			}
		}
	})

	t.Run("rewrite with prompt plain text", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "101",
			"--selection", "She said quietly.",
			"--prompt", "Use stronger verbs",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "She whispered.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("rewrite with action preset", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "101",
			"--selection", "She said quietly.",
			"--action", "show",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "She whispered.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("rewrite with selection file", func(t *testing.T) {
		tmpDir := t.TempDir()
		selFile := filepath.Join(tmpDir, "selection.txt")
		_ = os.WriteFile(selFile, []byte("She said quietly from file."), 0644)

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "101",
			"--selection-file", selFile,
			"--action", "tighten",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "She whispered.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("rewrite with missing selection file", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "101",
			"--selection-file", "/nonexistent/path/sel.txt",
			"--action", "tighten",
		})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "error reading selection file") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("rewrite with --stream", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "101",
			"--selection", "She said quietly.",
			"--prompt", "Whisper",
			"--stream",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "She whispered.") {
			t.Fatalf("unexpected stream output: %s", out.String())
		}
	})

	t.Run("rewrite with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "101",
			"--selection", "She said quietly.",
			"--action", "tighten",
			"--persist",
			"--json",
		})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res client.RewriteResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res.Prose != "She whispered." || !res.Persisted {
			t.Fatalf("unexpected rewrite json: %+v", res)
		}
	})

	t.Run("server error", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{
			"generate", "rewrite", "404",
			"--selection", "Missing chapter.",
			"--action", "tighten",
		})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Scene not found.") {
			t.Fatalf("expected 404 error: %s", errOut.String())
		}
	})
}

func TestGenerateSummarize(t *testing.T) {
	var lastBody map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scenes/101/summarize" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&lastBody)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"summary":   "The investigation begins in the engine room.",
					"persisted": lastBody["persist"] == true,
					"model":     "test-model",
				},
			})
		case r.URL.Path == "/api/scenes/404/summarize":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scene not found."})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestGenerateEnv(t, handler)

	t.Run("missing chapter id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "chapter ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize", "--help"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Generate or update a one-line chapter summary.") {
			t.Fatalf("unexpected help text: %s", out.String())
		}
	})

	t.Run("plain text output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "The investigation begins in the engine room.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize", "101", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res client.SummaryResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res.Summary != "The investigation begins in the engine room." || !res.Persisted {
			t.Fatalf("unexpected summary json: %+v", res)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Scene not found.") {
			t.Fatalf("expected 404 error: %s", errOut.String())
		}
	})

	t.Run("defaults to persist true", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize", "101"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if lastBody["persist"] != true {
			t.Fatalf("expected persist:true in request body, got %+v", lastBody)
		}
	})

	t.Run("--no-persist sends persist false", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"generate", "summarize", "101", "--no-persist"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if lastBody["persist"] != false {
			t.Fatalf("expected persist:false in request body, got %+v", lastBody)
		}
	})
}
