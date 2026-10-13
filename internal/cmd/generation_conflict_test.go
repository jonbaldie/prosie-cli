package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestGenerationConflictKeepsCandidateInErrorOutput(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		stream bool
	}{
		{"continue", []string{"generate", "continue", "101", "--no-stream"}, false},
		{"rewrite", []string{"generate", "rewrite", "101", "--selection", "Original", "--action", "tighten", "--persist"}, false},
		{"summary", []string{"generate", "summarize", "101"}, false},
		{"stream continue", []string{"generate", "continue", "101"}, true},
		{"stream rewrite", []string{"generate", "rewrite", "101", "--selection", "Original", "--action", "tighten", "--persist", "--stream"}, true},
	}
	for _, tc := range cases {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: " text", true: " json"}[jsonOutput], func(t *testing.T) {
				handler := func(w http.ResponseWriter, r *http.Request) {
					payload := map[string]string{"message": "This chapter changed. Nothing was saved.", "error": "conflict", "generated_content": "Generated candidate."}
					if tc.stream && !jsonOutput {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"Generated \"}\n\n"))
						_, _ = w.Write([]byte("event: error\ndata: "))
						_ = json.NewEncoder(w).Encode(payload)
						_, _ = w.Write([]byte("\n"))
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(payload)
				}
				_, cfg, httpClient := setupTestGenerateEnv(t, handler)
				command, stdout, stderr := newTestRootCmd(cfg, httpClient)
				args := append([]string(nil), tc.args...)
				if jsonOutput {
					args = append(args, "--json")
				}
				if code := command.Execute(args); code != 1 {
					t.Fatalf("exit code %d, want 1", code)
				}
				wantOutput := ""
				if tc.stream && !jsonOutput {
					wantOutput = "Generated "
				}
				if stdout.String() != wantOutput {
					t.Fatalf("stdout %q, want provisional tokens %q", stdout.String(), wantOutput)
				}
				if !strings.Contains(stderr.String(), "Generated candidate.") || !strings.Contains(stderr.String(), "Nothing was saved.") {
					t.Fatalf("missing conflict or candidate in stderr %q", stderr.String())
				}
			})
		}
	}
}
