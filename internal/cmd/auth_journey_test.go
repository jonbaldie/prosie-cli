package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

// A saved login must remain usable by a new invocation, and logout must remove it.
func TestSavedLoginSurvivesReopeningAndLogout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/user" || r.Header.Get("Authorization") != "Bearer saved-token" {
			http.Error(w, "unexpected authentication request", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":17,"name":"Ada Writer","email":"ada@example.test"}`)
	}))
	defer server.Close()
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "")
	configPath := filepath.Join(t.TempDir(), "config.json")
	run := func(wantCode int, args ...string) map[string]any {
		t.Helper()
		root := cmd.NewRootCmd()
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, configPath, server.Client()
		if code := root.Execute(args); code != wantCode || errOut.Len() != 0 {
			t.Fatalf("%v: code=%d stderr=%q stdout=%q", args, code, errOut.String(), out.String())
		}
		var result map[string]any
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	check := func(got, want map[string]any) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v; want %#v", got, want)
		}
	}
	user := map[string]any{"id": float64(17), "name": "Ada Writer", "email": "ada@example.test"}
	scopes := []any{"read", "write", "generate"}
	check(run(1, "auth", "status", "--json"), map[string]any{"authenticated": false, "api_url": server.URL, "token_source": "none"})
	check(run(0, "auth", "login", "--token", "saved-token", "--json"), map[string]any{"status": "authenticated", "api_url": server.URL, "user": user, "scopes": scopes})
	check(run(0, "auth", "status", "--json"), map[string]any{"authenticated": true, "api_url": server.URL, "token_source": "config", "user": user, "scopes": scopes})
	check(run(0, "auth", "logout", "--json"), map[string]any{"status": "logged_out", "message": "Logged out successfully", "env_token_override": false})
	check(run(1, "auth", "status", "--json"), map[string]any{"authenticated": false, "api_url": server.URL, "token_source": "none"})
}
