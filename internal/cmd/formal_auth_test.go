package cmd_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestFormalAuthMalformedConfigStopsLoginRequests(t *testing.T) {
	for _, args := range [][]string{{"auth", "login", "--token", "secret"}, {"auth", "login", "--no-browser"}} {
		t.Run(args[len(args)-1], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
				t.Fatal(err)
			}
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, path, server.Client()
			code := root.Execute(args)
			if code != 1 || calls != 0 || errOut.Len() == 0 {
				t.Fatalf("code=%d calls=%d stdout=%q stderr=%q", code, calls, out.String(), errOut.String())
			}
		})
	}
}
