package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type deleteTransport func(*http.Request) (*http.Response, error)

func (f deleteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDeleteCommandsPreserveBehavior(t *testing.T) {
	for _, tc := range []struct {
		command, id, resource, path, jsonID string
	}{
		{"book", "8", "book", "/api/stories/8", "8"},
		{"chapter", "draft-7", "chapter", "/api/scenes/draft-7", `"draft-7"`},
		{"chat", "007", "conversation", "/api/conversations/007", "7"},
		{"codex", "8", "codex entry", "/api/codex-entries/8", "8"},
		{"series", "8", "series", "/api/series/8", "8"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			requests := 0
			t.Setenv("PROSIE_API_URL", "https://example.test")
			t.Setenv("PROSIE_API_TOKEN", "test-token")
			configPath := filepath.Join(t.TempDir(), "config.json")
			httpClient := &http.Client{Transport: deleteTransport(func(r *http.Request) (*http.Response, error) {
				if err := r.Context().Err(); err != nil {
					return nil, err
				}
				requests++
				if r.Method != http.MethodDelete || r.URL.Path != tc.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody}, nil
			})}
			root, out, errOut := newTestRootCmd(configPath, httpClient)
			root.In = strings.NewReader("no\n")
			code := root.Execute([]string{tc.command, "delete", tc.id, "--json"})
			prompt := fmt.Sprintf("Are you sure you want to delete %s %s? [y/N]: Deletion cancelled.\n", tc.resource, tc.id)
			if code != 0 || out.String() != "" || errOut.String() != prompt || requests != 0 {
				t.Fatalf("cancel: code=%d stdout=%q stderr=%q requests=%d", code, out.String(), errOut.String(), requests)
			}
			out.Reset()
			errOut.Reset()
			code = root.Execute([]string{tc.command, "delete", tc.id, "-y", "--json"})
			want := fmt.Sprintf("{\n  \"deleted\": true,\n  \"id\": %s\n}\n", tc.jsonID)
			if code != 0 || out.String() != want || errOut.String() != "" || requests != 1 {
				t.Fatalf("delete: code=%d stdout=%q stderr=%q requests=%d", code, out.String(), errOut.String(), requests)
			}
			out.Reset()
			errOut.Reset()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			root.Context = ctx
			code = root.Execute([]string{tc.command, "delete", tc.id, "-y"})
			if code != 1 || out.Len() != 0 || !strings.HasPrefix(errOut.String(), "error deleting "+tc.resource+": ") || !strings.Contains(errOut.String(), "context canceled") || requests != 1 {
				t.Fatalf("context: code=%d stdout=%q stderr=%q requests=%d", code, out.String(), errOut.String(), requests)
			}
		})
	}
}

func TestDeleteAuthenticationError(t *testing.T) {
	t.Setenv("PROSIE_API_TOKEN", "")
	root, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), nil)
	code := root.Execute([]string{"book", "delete", "1", "-y"})
	want := "authentication error: you are not logged in. Run 'prosie auth login' or set PROSIE_API_TOKEN\n"
	if code != 1 || out.Len() != 0 || errOut.String() != want {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
}
