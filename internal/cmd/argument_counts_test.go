package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFixedArgumentCounts(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{"book list", nil},
		{"book create", []string{"--title", "Title"}},
		{"book show", []string{"1"}},
		{"book update", []string{"1", "--title", "Title"}},
		{"book delete", []string{"1", "-y"}},
		{"book duplicate", []string{"1"}},
		{"book export", []string{"1"}},
		{"book import", []string{"-"}},
		{"chapter list", []string{"1"}},
		{"chapter create", []string{"1", "--title", "Title"}},
		{"chapter show", []string{"1"}},
		{"chapter update", []string{"1", "--title", "Title"}},
		{"chapter delete", []string{"1", "-y"}},
		{"chapter export", []string{"1"}},
		{"series list", nil},
		{"series create", []string{"--title", "Title"}},
		{"series show", []string{"1"}},
		{"series update", []string{"1", "--title", "Title"}},
		{"series delete", []string{"1", "-y"}},
		{"series attach", []string{"1", "2"}},
		{"series detach", []string{"1", "2"}},
		{"codex list", []string{"1"}},
		{"codex create", []string{"1", "--name", "Name", "--details", "Details"}},
		{"codex show", []string{"1"}},
		{"codex update", []string{"1", "--name", "Name"}},
		{"codex delete", []string{"1", "-y"}},
		{"chat list", []string{"1"}},
		{"chat show", []string{"1"}},
		{"chat delete", []string{"1", "-y"}},
		{"chat export", []string{"1"}},
		{"chat import", []string{"1", "-"}},
		{"generate continue", []string{"1"}},
		{"generate rewrite", []string{"1"}},
		{"generate summarize", []string{"1"}},
		{"generate reject", []string{"1"}},
		{"generate undo", []string{"1"}},
		{"llm show", nil},
		{"llm models", nil},
		{"llm update", []string{"--provider", "openai"}},
		{"auth login", []string{"--token", "test-token"}},
		{"auth status", nil},
		{"auth logout", nil},
		{"version", nil},
	}
	for _, tc := range commands {
		for _, jsonMode := range []bool{false, true} {
			name := tc.name
			if jsonMode {
				name += " --json"
			}
			t.Run(name, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
				}))
				defer server.Close()
				t.Setenv("PROSIE_API_URL", server.URL)
				t.Setenv("PROSIE_API_TOKEN", "test-token")

				args := append(strings.Fields(tc.name), tc.args...)
				if jsonMode {
					args = append(args, "--json")
				}
				args = append(args, "extra")
				cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), server.Client())
				code := cmd.Execute(args)
				if code != 1 {
					t.Errorf("exit = %d, want 1; stderr: %s", code, errOut.String())
				}
				unexpected := "unexpected argument \"extra\". Usage: prosie " + tc.name
				if strings.HasPrefix(tc.name, "llm ") {
					unexpected = "unexpected argument. Usage: prosie " + tc.name
				}
				if got := errOut.String(); !strings.Contains(got, unexpected) {
					t.Errorf("unexpected stderr: %q", got)
				}
				if out.Len() != 0 {
					t.Errorf("unexpected stdout: %s", out.String())
				}
				if got := requests.Load(); got != 0 {
					t.Errorf("sent %d API requests, want none", got)
				}
			})
		}
	}
}

func TestBookDeleteRejectsExtraArguments(t *testing.T) {
	cmd, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), nil)
	code := cmd.Execute([]string{"book", "delete", "1", "2", "3", "-y"})
	if code != 1 || out.Len() != 0 {
		t.Fatalf("exit = %d, stdout: %s; stderr: %s", code, out.String(), errOut.String())
	}
	if got := errOut.String(); got != "error: unexpected argument \"2\". Usage: prosie book delete <id> [flags]\n" {
		t.Errorf("unexpected stderr: %q", got)
	}
}
