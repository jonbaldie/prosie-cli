package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// failedTurnAPI answers conversation creation for book 5 with id 77 (or with
// createStatus when it is not 201) and fails every message request with 502.
type failedTurnAPI struct {
	createStatus int
	created      int
	messageSent  int
}

func (f *failedTurnAPI) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/api/stories/5/conversations" && r.Method == http.MethodPost:
		f.created++
		w.WriteHeader(f.createStatus)
		if f.createStatus != http.StatusCreated {
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Story not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 77}})
	case strings.HasPrefix(r.URL.Path, "/api/conversations/") && r.Method == http.MethodPost:
		f.messageSent++
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Upstream LLM error"})
	default:
		http.NotFound(w, r)
	}
}

func TestChatTurnFailures(t *testing.T) {
	for _, sub := range []string{"send", "stream"} {
		t.Run(sub+" reports created conversation when the message fails", func(t *testing.T) {
			api := &failedTurnAPI{createStatus: http.StatusCreated}
			_, cfgPath, httpClient := setupTestChatEnv(t, api.handle)

			cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
			code := cmd.Execute([]string{"chat", sub, "5", "Hello there", "--json"})
			if code != 1 {
				t.Fatalf("expected exit code 1, got %d", code)
			}
			if api.created != 1 || api.messageSent != 1 {
				t.Fatalf("expected one create and one message request, got created=%d sent=%d", api.created, api.messageSent)
			}
			if !strings.Contains(errOut.String(), "Upstream LLM error") {
				t.Fatalf("expected API error in stderr, got %q", errOut.String())
			}
			if !strings.Contains(errOut.String(), "conversation 77 was created; continue with --conversation 77") {
				t.Fatalf("expected conversation ID in stderr, got %q", errOut.String())
			}
			if out.Len() != 0 {
				t.Fatalf("expected empty stdout, got %q", out.String())
			}
		})

		t.Run(sub+" with --conversation does not create a conversation when the message fails", func(t *testing.T) {
			api := &failedTurnAPI{createStatus: http.StatusCreated}
			_, cfgPath, httpClient := setupTestChatEnv(t, api.handle)

			cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
			code := cmd.Execute([]string{"chat", sub, "--conversation", "42", "Hello there"})
			if code != 1 {
				t.Fatalf("expected exit code 1, got %d", code)
			}
			if api.created != 0 || api.messageSent != 1 {
				t.Fatalf("expected no create and one message request, got created=%d sent=%d", api.created, api.messageSent)
			}
			if !strings.Contains(errOut.String(), "Upstream LLM error") {
				t.Fatalf("expected API error in stderr, got %q", errOut.String())
			}
			if strings.Contains(errOut.String(), "was created") {
				t.Fatalf("expected no created-conversation notice, got %q", errOut.String())
			}
		})

		t.Run(sub+" sends no message when conversation creation fails", func(t *testing.T) {
			api := &failedTurnAPI{createStatus: http.StatusNotFound}
			_, cfgPath, httpClient := setupTestChatEnv(t, api.handle)

			cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
			code := cmd.Execute([]string{"chat", sub, "5", "Hello there"})
			if code != 1 {
				t.Fatalf("expected exit code 1, got %d", code)
			}
			if api.created != 1 || api.messageSent != 0 {
				t.Fatalf("expected one create and no message request, got created=%d sent=%d", api.created, api.messageSent)
			}
			if !strings.Contains(errOut.String(), "error creating conversation: ") || !strings.Contains(errOut.String(), "Story not found") {
				t.Fatalf("expected creation error in stderr, got %q", errOut.String())
			}
		})
	}
}

func TestChatStreamFailureAfterTokensEndsTheLine(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations/804/messages/stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: delta\ndata: {\"delta\":\"Partial\"}\n\n")
		_, _ = io.WriteString(w, "event: error\ndata: {\"message\":\"Upstream AI timeout.\"}\n\n")
	}
	_, cfgPath, httpClient := setupTestChatEnv(t, handler)

	cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
	code := cmd.Execute([]string{"chat", "stream", "--conversation", "804", "Go on"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if out.String() != "Partial\n" {
		t.Fatalf("expected partial tokens and a line end on stdout, got %q", out.String())
	}
	if !strings.HasPrefix(errOut.String(), "error sending message: ") || !strings.Contains(errOut.String(), "Upstream AI timeout.") {
		t.Fatalf("unexpected stderr: %q", errOut.String())
	}
}
