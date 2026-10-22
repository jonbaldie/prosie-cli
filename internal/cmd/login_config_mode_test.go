package cmd_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

// Both login paths must replace an existing world-readable config.json with a private file.
func TestLoginReplacesPermissiveConfigWithPrivateFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/device/code":
			fmt.Fprint(w, `{"device_code":"device-secret","user_code":"ABCD","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`)
		case "/oauth/token":
			fmt.Fprint(w, `{"access_token":"new-token","token_type":"Bearer","scope":"read write"}`)
		case "/api/user":
			if r.Header.Get("Authorization") != "Bearer new-token" {
				http.Error(w, "unexpected token", http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"id":3,"name":"Mode Writer","email":"mode@example.test"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "")

	for name, args := range map[string][]string{
		"token":  {"auth", "login", "--token", "new-token", "--json"},
		"device": {"auth", "login", "--no-browser", "--json"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := filepath.Join(dir, "config.json")
			if err := os.WriteFile(cfg, []byte(`{"token":"old-token"}`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(cfg, 0644); err != nil {
				t.Fatal(err)
			}
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, cfg, server.Client()
			if code := root.Execute(args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q stdout=%q", code, errOut.String(), out.String())
			}

			saved, err := os.ReadFile(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(saved), `"token": "new-token"`) {
				t.Fatalf("saved config=%q", saved)
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(cfg)
				if err != nil {
					t.Fatal(err)
				}
				if perm := info.Mode().Perm(); perm != 0600 {
					t.Fatalf("config mode=%04o; want 0600", perm)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("directory has %d entries; want only config.json", len(entries))
			}
		})
	}
}
