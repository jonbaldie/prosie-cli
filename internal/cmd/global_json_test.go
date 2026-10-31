package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestLeadingJSONPreservesChatMessageAfterSeparator(t *testing.T) {
	var received string
	_, cfgPath, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		received = body.Content
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"message":{"id":2,"conversation_id":777,"role":"assistant","content":"Reply"},"model":"test"}}`)
	})
	root, out, errOut := newTestRootCmd(cfgPath, httpClient)
	code := root.Execute([]string{"--json", "chat", "send", "--conversation", "777", "--", "Hello"})
	if code != 0 || errOut.Len() != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if received != "Hello" {
		t.Errorf("received message %q, want Hello", received)
	}
	var response map[string]any
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON output %q: %v", out.String(), err)
	}
	if response["conversation_id"] != float64(777) {
		t.Errorf("conversation_id = %v, want 777", response["conversation_id"])
	}
}
