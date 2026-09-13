package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func setupTestChatEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string, *http.Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = config.Save(configPath, &config.Config{
		ApiURL: server.URL,
		Token:  "test-chat-token",
	})
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "test-chat-token")

	return server, configPath, server.Client()
}

func TestChatHelp(t *testing.T) {
	t.Run("with --help", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd("", nil)
		code := cmd.Execute([]string{"chat", "--help"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Manage novel chat conversations grounded in your manuscript.") {
			t.Fatalf("unexpected help output: %s", out.String())
		}
		if !strings.Contains(out.String(), "send") || !strings.Contains(out.String(), "stream") {
			t.Fatalf("missing chat subcommands in help: %s", out.String())
		}
	})

	t.Run("without args", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd("", nil)
		code := cmd.Execute([]string{"chat"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Manage novel chat conversations") {
			t.Fatalf("unexpected help output: %s", out.String())
		}
	})
}

func TestChatUnknownSubcommand(t *testing.T) {
	cmd, _, errOut := newTestRootCmd("", nil)
	code := cmd.Execute([]string{"chat", "unknown-cmd"})
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown chat command: unknown-cmd") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestChatList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/conversations" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":              10,
					"story_id":        1,
					"title":           "Plot Twist Discussion",
					"fidelity":        "summary",
					"expires_in_days": 30,
					"updated_at":      "2026-09-13T12:00:00Z",
				},
				{
					"id":              11,
					"story_id":        1,
					"title":           nil,
					"fidelity":        "verbatim",
					"expires_in_days": 14,
					"updated_at":      "2026-09-13T14:30:00Z",
				},
			},
		})
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing book id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "list"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text tabular", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "list", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "TITLE") || !strings.Contains(stdout, "FIDELITY") || !strings.Contains(stdout, "EXPIRES") {
			t.Fatalf("missing table headers in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "Plot Twist Discussion") || !strings.Contains(stdout, "30d") {
			t.Fatalf("missing Plot Twist Discussion row: %s", stdout)
		}
		if !strings.Contains(stdout, "Conversation 11") || !strings.Contains(stdout, "14d") {
			t.Fatalf("missing Conversation 11 row: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "list", "1", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var convs []client.Conversation
		if err := json.Unmarshal(out.Bytes(), &convs); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if len(convs) != 2 {
			t.Fatalf("expected 2 conversations, got %d", len(convs))
		}
		if convs[0].ID != 10 || *convs[0].Title != "Plot Twist Discussion" {
			t.Fatalf("unexpected conversation: %+v", convs[0])
		}
	})

	t.Run("empty list", func(t *testing.T) {
		emptyHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
		_, emptyCfg, emptyClient := setupTestChatEnv(t, emptyHandler)

		cmd, out, errOut := newTestRootCmd(emptyCfg, emptyClient)
		code := cmd.Execute([]string{"chat", "list", "1"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No conversations found.") {
			t.Fatalf("expected 'No conversations found.', got %s", out.String())
		}
	})
}

func TestChatShow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/conversations/42":
			model := "openai/gpt-5.6-luna"
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":              42,
					"story_id":        5,
					"title":           "Worldbuilding Query",
					"fidelity":        "verbatim",
					"model":           model,
					"expires_in_days": 25,
					"updated_at":      "2026-09-13T16:00:00Z",
					"messages": []map[string]any{
						{
							"id":              1,
							"conversation_id": 42,
							"role":            "user",
							"content":         "What is the capital of the outer colony?",
						},
						{
							"id":              2,
							"conversation_id": 42,
							"role":            "assistant",
							"content":         "The capital is Port Nova, situated on the western ridge.",
						},
					},
				},
			})
		case "/api/conversations/43":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":         43,
					"story_id":   5,
					"title":      "Empty Thread",
					"fidelity":   "summary",
					"messages":   []any{},
					"updated_at": "2026-09-13T16:00:00Z",
				},
			})
		case "/api/conversations/404":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Conversation not found."})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "show"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "conversation ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text show", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "show", "42"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		stdout := out.String()
		if !strings.Contains(stdout, "Title:       Worldbuilding Query") {
			t.Fatalf("missing title: %s", stdout)
		}
		if !strings.Contains(stdout, "Fidelity:    verbatim") {
			t.Fatalf("missing fidelity: %s", stdout)
		}
		if !strings.Contains(stdout, "Model:       openai/gpt-5.6-luna") {
			t.Fatalf("missing model: %s", stdout)
		}
		if !strings.Contains(stdout, "Book ID:     5") {
			t.Fatalf("missing book id: %s", stdout)
		}
		if !strings.Contains(stdout, "[You]") || !strings.Contains(stdout, "What is the capital of the outer colony?") {
			t.Fatalf("missing user turn: %s", stdout)
		}
		if !strings.Contains(stdout, "[Assistant]") || !strings.Contains(stdout, "The capital is Port Nova") {
			t.Fatalf("missing assistant turn: %s", stdout)
		}
	})

	t.Run("plain text show empty thread", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "show", "43"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No messages found.") {
			t.Fatalf("expected 'No messages found.', got %s", out.String())
		}
	})

	t.Run("json show", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "show", "42", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var conv client.Conversation
		if err := json.Unmarshal(out.Bytes(), &conv); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if conv.ID != 42 || len(conv.Messages) != 2 {
			t.Fatalf("unexpected json conversation: %+v", conv)
		}
	})

	t.Run("not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "show", "404"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Conversation not found") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})
}

func TestChatSend(t *testing.T) {
	createdThread := false
	sentMessage := false

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/stories/1/conversations" && r.Method == http.MethodPost:
			createdThread = true
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			title, _ := body["title"].(string)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":       501,
					"story_id": 1,
					"title":    title,
					"fidelity": "summary",
				},
			})
		case r.URL.Path == "/api/conversations/501/messages" && r.Method == http.MethodPost:
			sentMessage = true
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			content, _ := body["content"].(string)

			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"message": map[string]any{
						"id":              1,
						"conversation_id": 501,
						"role":            "assistant",
						"content":         fmt.Sprintf("Echo: %s", content),
					},
					"model": "openai/gpt-5.6-luna",
					"usage": map[string]any{
						"prompt_tokens":     40,
						"completion_tokens": 10,
						"total_tokens":      50,
					},
				},
			})
		case r.URL.Path == "/api/conversations/777/messages" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			content, _ := body["content"].(string)

			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"message": map[string]any{
						"id":              2,
						"conversation_id": 777,
						"role":            "assistant",
						"content":         fmt.Sprintf("Reply from 777: %s", content),
					},
					"model": "openai/gpt-5.6-luna",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing args", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "send"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID and message are required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("missing message only", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "send", "1"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "message is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("send creates thread and prints assistant reply", func(t *testing.T) {
		createdThread = false
		sentMessage = false

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "send", "1", "Who is the antagonist?", "--title", "Antagonist Thread"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !createdThread || !sentMessage {
			t.Fatalf("expected thread creation and message sending, got created=%v sent=%v", createdThread, sentMessage)
		}
		if !strings.Contains(out.String(), "Echo: Who is the antagonist?") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("send with --conversation flag", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "send", "--conversation", "777", "Continuing old query"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Reply from 777: Continuing old query") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("send with --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "send", "1", "Testing JSON flag", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("invalid json output: %v, raw: %s", err, out.String())
		}
		if payload["conversation_id"] != float64(501) {
			t.Fatalf("expected conversation_id 501, got %+v", payload["conversation_id"])
		}
		msg, ok := payload["message"].(map[string]any)
		if !ok || msg["content"] != "Echo: Testing JSON flag" {
			t.Fatalf("unexpected message in json: %+v", payload)
		}
	})
}

func TestChatStream(t *testing.T) {
	createdThread := false

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/stories/2/conversations" && r.Method == http.MethodPost:
			createdThread = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":       801,
					"story_id": 2,
					"title":    "Streaming Thread",
				},
			})
		case r.URL.Path == "/api/conversations/801/messages/stream" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatalf("expected flusher")
			}
			_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"Streaming \"}\n\n")
			flusher.Flush()
			_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"is \"}\n\n")
			flusher.Flush()
			_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"active.\"}\n\n")
			flusher.Flush()
			_, _ = fmt.Fprintf(w, "event: done\ndata: {\"message\":{\"id\":10,\"role\":\"assistant\",\"content\":\"Streaming is active.\"}}\n\n")
			flusher.Flush()
		case r.URL.Path == "/api/conversations/802/messages/stream" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "event: error\ndata: {\"message\":\"Upstream AI timeout.\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing args", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "stream"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID and message are required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("stream creates conversation and streams tokens", func(t *testing.T) {
		createdThread = false
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "stream", "2", "Stream me this story beat"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !createdThread {
			t.Fatalf("expected conversation thread to be created")
		}
		if !strings.Contains(out.String(), "Streaming is active.") {
			t.Fatalf("unexpected streamed stdout: %s", out.String())
		}
	})

	t.Run("stream error handling", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "stream", "--conversation", "802", "Trigger failure"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "Upstream AI timeout") {
			t.Fatalf("expected error in stderr, got: %s", errOut.String())
		}
	})
}

func TestChatExport(t *testing.T) {
	exportData := `{"version":1,"title":"Full Chat History","fidelity":"summary","messages":[{"role":"user","content":"Hi"}]}`
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations/100/export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, exportData)
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "export"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "conversation ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("export to stdout", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "export", "100"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Full Chat History") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("export to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "exported-chat.json")

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "export", "100", "-o", outFile})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Exported conversation 100 to "+outFile) {
			t.Fatalf("unexpected stdout: %s", out.String())
		}

		bytesRead, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("failed to read exported file: %v", err)
		}
		if string(bytesRead) != exportData {
			t.Fatalf("file content mismatch: %s", string(bytesRead))
		}
	})

	t.Run("export to file --json", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "exported-chat.json")

		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "export", "100", "--output", outFile, "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != "100" || res["output"] != outFile {
			t.Fatalf("unexpected json: %+v", res)
		}
	})
}

func TestChatImport(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "import-chat.json")
	chatContent := `{"version":1,"title":"Imported Discussion","fidelity":"summary","messages":[{"role":"user","content":"Question"}]}`
	if err := os.WriteFile(jsonPath, []byte(chatContent), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/3/conversations/import" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseMultipartForm(10 << 20)
		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       601,
				"story_id": 3,
				"title":    "Imported Discussion",
				"fidelity": "summary",
			},
		})
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing args", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "import"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "book ID and file path are required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("file not found", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "import", "3", "/path/to/missing.json"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "file not found") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("plain text import", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "import", "3", jsonPath})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Imported conversation 601 (\"Imported Discussion\") into book 3.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("json import", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "import", "3", jsonPath, "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var conv client.Conversation
		if err := json.Unmarshal(out.Bytes(), &conv); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if conv.ID != 601 || *conv.Title != "Imported Discussion" {
			t.Fatalf("unexpected json: %+v", conv)
		}
	})
}

func TestChatDelete(t *testing.T) {
	deleted := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/conversations/901" && r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	t.Run("missing id", func(t *testing.T) {
		cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "delete"})
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "conversation ID is required") {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	})

	t.Run("cancelled via prompt", func(t *testing.T) {
		cmd, out, _ := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("n\n")
		code := cmd.Execute([]string{"chat", "delete", "901"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d", code)
		}
		if !strings.Contains(out.String(), "Deletion cancelled.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("confirmed via prompt", func(t *testing.T) {
		deleted = false
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		cmd.In = bytes.NewBufferString("y\n")
		code := cmd.Execute([]string{"chat", "delete", "901"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !deleted {
			t.Fatalf("expected conversation to be deleted")
		}
		if !strings.Contains(out.String(), "Deleted conversation 901.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with --yes flag", func(t *testing.T) {
		deleted = false
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "delete", "901", "--yes"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		if !deleted {
			t.Fatalf("expected conversation to be deleted")
		}
		if !strings.Contains(out.String(), "Deleted conversation 901.") {
			t.Fatalf("unexpected stdout: %s", out.String())
		}
	})

	t.Run("delete with -y and --json", func(t *testing.T) {
		cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
		code := cmd.Execute([]string{"chat", "delete", "901", "-y", "--json"})
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
		}
		var res map[string]any
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("invalid json: %v, raw: %s", err, out.String())
		}
		if res["id"] != "901" || res["deleted"] != true {
			t.Fatalf("unexpected json: %+v", res)
		}
	})
}
