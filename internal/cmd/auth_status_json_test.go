package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestAuthStatusJSONWithInvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/user" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Unauthenticated."}`)
	}))
	defer server.Close()
	t.Setenv("PROSIE_API_URL", server.URL)
	configPath := filepath.Join(t.TempDir(), "config.json")

	run := func(t *testing.T, token string, args ...string) (int, string, string) {
		t.Helper()
		t.Setenv("PROSIE_API_TOKEN", token)
		root := cmd.NewRootCmd()
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, configPath, server.Client()
		code := root.Execute(args)
		return code, out.String(), errOut.String()
	}
	parse := func(t *testing.T, raw string) map[string]any {
		t.Helper()
		var result map[string]any
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatalf("stdout is not valid JSON: %v, raw: %q", err, raw)
		}
		return result
	}

	t.Run("invalid token with --json prints status JSON and reports the error", func(t *testing.T) {
		code, stdout, stderr := run(t, "bad-token", "auth", "status", "--json")
		if code != 1 {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		result := parse(t, stdout)
		if result["authenticated"] != false {
			t.Fatalf("authenticated=%v stdout=%q", result["authenticated"], stdout)
		}
		if result["api_url"] != server.URL {
			t.Fatalf("api_url=%v want %q", result["api_url"], server.URL)
		}
		if result["token_source"] != "environment" {
			t.Fatalf("token_source=%v want environment", result["token_source"])
		}
		if !strings.Contains(stderr, "authentication error") || !strings.Contains(stderr, "401") {
			t.Fatalf("stderr=%q want authentication error with 401", stderr)
		}
	})

	t.Run("no token with --json keeps the same JSON shape", func(t *testing.T) {
		code, stdout, stderr := run(t, "", "auth", "status", "--json")
		if code != 1 {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		result := parse(t, stdout)
		if result["authenticated"] != false || result["token_source"] != "none" || result["api_url"] != server.URL {
			t.Fatalf("result=%v stdout=%q", result, stdout)
		}
	})

	t.Run("invalid token without --json keeps text output", func(t *testing.T) {
		code, stdout, stderr := run(t, "bad-token", "auth", "status")
		if code != 1 {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		if stdout != "" {
			t.Fatalf("stdout=%q want empty", stdout)
		}
		if !strings.Contains(stderr, "authentication error") {
			t.Fatalf("stderr=%q", stderr)
		}
	})

	t.Run("malformed config with --json keeps current error behavior", func(t *testing.T) {
		badPath := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(badPath, []byte("{broken json"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PROSIE_API_TOKEN", "")
		root := cmd.NewRootCmd()
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, badPath, server.Client()
		code := root.Execute([]string{"auth", "status", "--json"})
		if code != 1 {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
		}
		if out.String() != "" {
			t.Fatalf("stdout=%q want empty when no status result exists", out.String())
		}
		if !strings.Contains(errOut.String(), "authentication error") {
			t.Fatalf("stderr=%q", errOut.String())
		}
	})
}
