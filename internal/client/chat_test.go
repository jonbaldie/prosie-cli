package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListConversations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/1/conversations" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":              101,
					"story_id":        1,
					"title":           "Plot discussion",
					"fidelity":        "summary",
					"expires_in_days": 30,
					"updated_at":      "2026-09-13T12:00:00Z",
				},
				{
					"id":              102,
					"story_id":        1,
					"title":           nil,
					"fidelity":        "verbatim",
					"expires_in_days": 15,
					"updated_at":      "2026-09-13T13:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "test-token", server.Client())
	convs, err := cli.ListConversations(context.Background(), "1")
	if err != nil {
		t.Fatalf("ListConversations returned error: %v", err)
	}

	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}

	if convs[0].ID != 101 || *convs[0].Title != "Plot discussion" {
		t.Fatalf("unexpected conversation 0: %+v", convs[0])
	}
	if convs[0].DisplayTitle() != "Plot discussion" {
		t.Fatalf("unexpected DisplayTitle: %q", convs[0].DisplayTitle())
	}
	if convs[1].DisplayTitle() != "Conversation 102" {
		t.Fatalf("unexpected DisplayTitle for nil title: %q", convs[1].DisplayTitle())
	}
}

func TestListConversations_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.ListConversations(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "book ID is required") {
		t.Fatalf("expected book ID is required, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Story not found."})
	}))
	defer server.Close()

	cli = New(server.URL, "token", server.Client())
	_, err = cli.ListConversations(context.Background(), "999")
	if err == nil || !strings.Contains(err.Error(), "Story not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestGetConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations/42" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":              42,
				"story_id":        7,
				"title":           "Character Arc Discussion",
				"fidelity":        "summary",
				"model":           "openai/gpt-5.6-luna",
				"expires_in_days": 28,
				"messages": []map[string]any{
					{
						"id":              1,
						"conversation_id": 42,
						"role":            "user",
						"content":         "What motivates Dax?",
					},
					{
						"id":              2,
						"conversation_id": 42,
						"role":            "assistant",
						"content":         "Dax is seeking revenge for his lost crew.",
					},
				},
				"created_at": "2026-09-13T10:00:00Z",
				"updated_at": "2026-09-13T10:05:00Z",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	conv, err := cli.GetConversation(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetConversation returned error: %v", err)
	}

	if conv.ID != 42 || conv.StoryID != 7 {
		t.Fatalf("unexpected conversation: %+v", conv)
	}
	if len(conv.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Content != "What motivates Dax?" {
		t.Fatalf("unexpected message 0 content: %q", conv.Messages[0].Content)
	}
	if conv.Messages[1].Role != "assistant" {
		t.Fatalf("unexpected message 1 role: %q", conv.Messages[1].Role)
	}
}

func TestGetConversation_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.GetConversation(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("expected conversation ID is required, got %v", err)
	}
}

func TestCreateConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/5/conversations" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		title, _ := body["title"].(string)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":         88,
				"story_id":   5,
				"title":      title,
				"fidelity":   "summary",
				"created_at": "2026-09-13T15:00:00Z",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	conv, err := cli.CreateConversation(context.Background(), "5", "Worldbuilding lore")
	if err != nil {
		t.Fatalf("CreateConversation returned error: %v", err)
	}

	if conv.ID != 88 || conv.StoryID != 5 || *conv.Title != "Worldbuilding lore" {
		t.Fatalf("unexpected created conversation: %+v", conv)
	}
}

func TestCreateConversation_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.CreateConversation(context.Background(), "", "Title")
	if err == nil || !strings.Contains(err.Error(), "book ID is required") {
		t.Fatalf("expected book ID is required, got %v", err)
	}
}

func TestDeleteConversation(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/conversations/55" && r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	err := cli.DeleteConversation(context.Background(), "55")
	if err != nil {
		t.Fatalf("DeleteConversation returned error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected conversation to be deleted")
	}
}

func TestDeleteConversation_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	err := cli.DeleteConversation(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("expected conversation ID is required, got %v", err)
	}
}

func TestExportConversation(t *testing.T) {
	expectedJSON := `{"version":1,"title":"Exported Discussion","fidelity":"summary","messages":[{"role":"user","content":"Hello"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations/12/export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, expectedJSON)
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	data, err := cli.ExportConversation(context.Background(), "12")
	if err != nil {
		t.Fatalf("ExportConversation returned error: %v", err)
	}
	if string(data) != expectedJSON {
		t.Fatalf("unexpected export data: %s", string(data))
	}
}

func TestExportConversation_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.ExportConversation(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("expected conversation ID is required, got %v", err)
	}
}

func TestImportConversation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "chat-thread.json")
	exportContent := `{"version":1,"title":"Imported Thread","fidelity":"summary","messages":[{"role":"user","content":"Past question"}]}`
	if err := os.WriteFile(filePath, []byte(exportContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stories/9/conversations/import" || r.Method != http.MethodPost {
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
				"id":         150,
				"story_id":   9,
				"title":      "Imported Thread",
				"fidelity":   "summary",
				"created_at": "2026-09-13T16:00:00Z",
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	conv, err := cli.ImportConversation(context.Background(), "9", filePath)
	if err != nil {
		t.Fatalf("ImportConversation returned error: %v", err)
	}

	if conv.ID != 150 || *conv.Title != "Imported Thread" {
		t.Fatalf("unexpected imported conversation: %+v", conv)
	}
}

func TestImportConversation_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.ImportConversation(context.Background(), "", "path/to/file")
	if err == nil || !strings.Contains(err.Error(), "book ID is required") {
		t.Fatalf("expected book ID is required, got %v", err)
	}

	_, err = cli.ImportConversation(context.Background(), "1", "nonexistent/file.json")
	if err == nil || !strings.Contains(err.Error(), "failed to open file") {
		t.Fatalf("expected failed to open file error, got %v", err)
	}
}

func TestSendChatMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations/33/messages" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["content"] != "How does the engine work?" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"message": map[string]any{
					"id":              201,
					"conversation_id": 33,
					"role":            "assistant",
					"content":         "The engine runs on dark matter siphon coils.",
				},
				"model": "openai/gpt-5.6-luna",
				"usage": map[string]any{
					"prompt_tokens":     80,
					"completion_tokens": 15,
					"total_tokens":      95,
				},
			},
		})
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	resp, err := cli.SendChatMessage(context.Background(), "33", "How does the engine work?")
	if err != nil {
		t.Fatalf("SendChatMessage returned error: %v", err)
	}

	if resp.Message == nil || resp.Message.Content != "The engine runs on dark matter siphon coils." {
		t.Fatalf("unexpected message content: %+v", resp.Message)
	}
	if resp.Model != "openai/gpt-5.6-luna" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 95 {
		t.Fatalf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}
}

func TestSendChatMessage_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.SendChatMessage(context.Background(), "", "Hello")
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("expected conversation ID is required, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "LLM provider error."})
	}))
	defer server.Close()

	cli = New(server.URL, "token", server.Client())
	_, err = cli.SendChatMessage(context.Background(), "33", "Hello")
	if err == nil || !strings.Contains(err.Error(), "LLM provider error") {
		t.Fatalf("expected LLM provider error, got %v", err)
	}
}

func TestStreamChatMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations/44/messages/stream" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("expected ResponseWriter to be Flusher")
		}

		_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"The \"}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"stars \"}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: delta\ndata: {\"delta\":\"align.\"}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: done\ndata: {\"message\":{\"id\":301,\"role\":\"assistant\",\"content\":\"The stars align.\"},\"model\":\"openai/gpt-5.6-luna\",\"usage\":{\"prompt_tokens\":50,\"completion_tokens\":4,\"total_tokens\":54}}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	cli := New(server.URL, "token", server.Client())
	var receivedTokens []string
	resp, err := cli.StreamChatMessage(context.Background(), "44", "Look up", func(token string) {
		receivedTokens = append(receivedTokens, token)
	})
	if err != nil {
		t.Fatalf("StreamChatMessage returned error: %v", err)
	}

	joined := strings.Join(receivedTokens, "")
	if joined != "The stars align." {
		t.Fatalf("expected streamed text 'The stars align.', got %q", joined)
	}
	if resp.Message == nil || resp.Message.Content != "The stars align." {
		t.Fatalf("unexpected done message: %+v", resp.Message)
	}
	if resp.Usage.TotalTokens != 54 {
		t.Fatalf("unexpected tokens: %d", resp.Usage.TotalTokens)
	}
}

func TestStreamChatMessage_Errors(t *testing.T) {
	cli := New("http://localhost:1", "token", nil)
	_, err := cli.StreamChatMessage(context.Background(), "", "Hello", nil)
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("expected conversation ID is required, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "event: error\ndata: {\"message\":\"Rate limit exceeded.\"}\n\n")
	}))
	defer server.Close()

	cli = New(server.URL, "token", server.Client())
	_, err = cli.StreamChatMessage(context.Background(), "44", "Hello", nil)
	if err == nil || !strings.Contains(err.Error(), "Rate limit exceeded") {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}
