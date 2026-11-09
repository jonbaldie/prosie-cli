package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestFormalChatStreamOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		json   bool
		fail   bool
		tokens []string
		want   string
	}{
		{"success empty", false, false, nil, "\n"},
		{"success token", false, false, []string{"hello"}, "hello\n"},
		{"success two and trailing newline", false, false, []string{"hello", "world\n"}, "helloworld\n\n"},
		{"failure empty", false, true, nil, ""},
		{"failure token", false, true, []string{"hello"}, "hello\n"},
		{"failure trailing newline", false, true, []string{"hello\n"}, "hello\n\n"},
		{"json failure hides token", true, true, []string{"hello"}, ""},
		{"json success hides tokens", true, false, []string{"hello", "world\n"}, "json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			handler := func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				w.Header().Set("Content-Type", "text/event-stream")
				for _, token := range tc.tokens {
					encoded, _ := json.Marshal(token)
					fmt.Fprintf(w, "event: delta\ndata: {\"delta\":%s}\n\n", encoded)
				}
				if tc.fail {
					fmt.Fprint(w, "event: error\ndata: {\"message\":\"failed\"}\n\n")
				} else {
					fmt.Fprint(w, "event: done\ndata: {\"model\":\"m\"}\n\n")
				}
			}
			_, cfg, hc := setupTestChatEnv(t, handler)
			root, out, _ := newTestRootCmd(cfg, hc)
			args := []string{"chat", "stream", "--conversation", "7", "hi"}
			if tc.json {
				args = append(args, "--json")
			}
			code := root.Execute(args)
			validOutput := out.String() == tc.want
			if tc.want == "json" {
				var result map[string]any
				validOutput = json.Unmarshal(out.Bytes(), &result) == nil && result["model"] == "m" && !strings.Contains(out.String(), "hello")
			}
			if (code == 0) == tc.fail || !validOutput || path != "/api/conversations/7/messages/stream" {
				t.Fatalf("code=%d output=%q path=%q", code, out.String(), path)
			}
		})
	}
}
