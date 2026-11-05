package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestChatRetry(t *testing.T) {
	_, cfg, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/conversations/42/retry" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"data":{"message":{"id":9,"role":"assistant","content":"Try again"},"model":"test-model","usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}}`)
	})
	for _, tc := range []struct {
		name     string
		args     []string
		wantJSON bool
	}{
		{"text", []string{"chat", "retry", "42"}, false},
		{"json", []string{"chat", "retry", "42", "--json"}, true},
		{"global json", []string{"--json", "chat", "retry", "42"}, true},
		{"trimmed id", []string{"chat", "retry", " 42 ", "--json"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, out, errOut := newTestRootCmd(cfg, httpClient)
			if code := cmd.Execute(tc.args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code %d, stderr %q", code, errOut.String())
			}
			if !tc.wantJSON {
				if out.String() != "Try again\n" {
					t.Fatalf("stdout %q", out.String())
				}
				return
			}
			var got struct {
				ConversationID int `json:"conversation_id"`
				Message        struct {
					Content string `json:"content"`
				} `json:"message"`
				Model string `json:"model"`
				Usage struct {
					TotalTokens int `json:"total_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.ConversationID != 42 || got.Message.Content != "Try again" || got.Model != "test-model" || got.Usage.TotalTokens != 5 {
				t.Fatalf("unexpected response %+v", got)
			}
		})
	}
}

func TestChatRetryErrorsAndHelp(t *testing.T) {
	_, cfg, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message":"No turn to retry"}`)
	})
	for _, tc := range []struct {
		args []string
		code int
		text string
		help bool
	}{
		{[]string{"chat", "retry", "42"}, 1, "No turn to retry", false},
		{[]string{"chat", "retry"}, 1, "conversation ID is required", false},
		{[]string{"chat", "retry", "--help"}, 0, "prosie chat retry <conversation-id>", true},
		{[]string{"chat", "--help"}, 0, "retry", true},
	} {
		cmd, out, errOut := newTestRootCmd(cfg, httpClient)
		if code := cmd.Execute(tc.args); code != tc.code {
			t.Fatalf("%v: code %d, stderr %q", tc.args, code, errOut.String())
		}
		if tc.help {
			if !strings.Contains(out.String(), tc.text) || errOut.Len() != 0 {
				t.Fatalf("%v: stdout %q, stderr %q", tc.args, out.String(), errOut.String())
			}
		} else if !strings.Contains(errOut.String(), tc.text) || out.Len() != 0 {
			t.Fatalf("%v: stdout %q, stderr %q", tc.args, out.String(), errOut.String())
		}
	}
}
