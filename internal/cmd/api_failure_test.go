package cmd_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestBookListReportsUsefulAPIFailures(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status        int
		body, message string
	}{
		{"message", 403, `{"message":"Book access denied"}`, "API error (403): Book access denied"},
		{"error code", 403, `{"error":"insufficient_scope"}`, "API error (403): insufficient_scope"},
		{"description", 403, `{"error":"insufficient_scope","error_description":"Read permission is required"}`, "API error (403): Read permission is required"},
		{"primary message", 403, `{"message":"Primary","error_description":"Fallback"}`, "API error (403): Primary"},
		{"validation", 422, `{"message":"Invalid input","errors":{"title":["Required","Too short"]}}`, "API error (422): Invalid input (title: Required, Too short)"},
		{"plain text", 503, "  Service unavailable\n", "API error (503): Service unavailable"},
		{"empty unauthorized", 401, "", "API error (401): unauthorized: invalid or missing authentication token"},
		{"empty server error", 500, "", "API request failed with status code 500"},
		{"unexpected JSON", 502, `{"message":42,"errors":false}`, "API request failed with status code 502"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			code := root.Execute([]string{"book", "list", "--json"})
			want := "error listing books: " + tc.message + "\n"
			if code != 1 || out.Len() != 0 || errOut.String() != want {
				t.Fatalf("code=%d stdout=%q stderr=%q; want 1 empty %q", code, out.String(), errOut.String(), want)
			}
		})
	}
}
