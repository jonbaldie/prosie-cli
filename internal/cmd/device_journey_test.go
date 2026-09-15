package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestDeviceLoginDisplaysVerificationURLAndSavesIssuedScopes(t *testing.T) {
	for _, tc := range []struct{ name, fields, url string }{
		{"complete URI", `"verification_uri_complete":"https://example.test/approve?code=AB",`, "https://example.test/approve?code=AB"},
		{"complete URL alias", `"verification_url_complete":"https://example.test/approve/AB",`, "https://example.test/approve/AB"},
		{"base URI", `"verification_uri":"https://example.test/approve",`, "https://example.test/approve?user_code=A+B%2BC"},
		{"base URL alias", `"verification_url":"https://example.test/approve",`, "https://example.test/approve?user_code=A+B%2BC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Header.Get("Accept") != "application/json" || !strings.HasPrefix(r.Header.Get("User-Agent"), "prosie-cli/") {
					t.Errorf("request headers: %v", r.Header)
				}
				switch r.URL.Path {
				case "/oauth/device/code":
					checkDeviceRequest(t, r, `{"client_id":"prosie-cli","scope":"read write"}`)
					fmt.Fprintf(w, `{%s"device_code":"device-secret","user_code":"A B+C","expires_in":60,"interval":1}`, tc.fields)
				case "/oauth/token":
					checkDeviceRequest(t, r, `{"client_id":"prosie-cli","device_code":"device-secret","grant_type":"urn:ietf:params:oauth:grant-type:device_code"}`)
					fmt.Fprint(w, `{"access_token":"issued-token","token_type":"Bearer","scope":"read  write"}`)
				case "/api/user":
					if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer issued-token" {
						t.Errorf("user request: %s %v", r.Method, r.Header)
					}
					fmt.Fprint(w, `{"id":9,"name":"Device Writer","email":"device@example.test"}`)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "")
			cfg := filepath.Join(t.TempDir(), "config.json")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, cfg, server.Client()
			if code := root.Execute([]string{"auth", "login", "--no-browser", "--scopes", "read write"}); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			want := "First, copy your one-time code: A B+C\nOpen this URL in your browser to approve authorization:\n  " + tc.url + "\n\nWaiting for authorization in browser...\n\nSuccessfully authenticated to " + server.URL + "!\nLogged in as Device Writer (device@example.test)\n"
			if out.String() != want {
				t.Fatalf("output=%q; want=%q", out.String(), want)
			}
			reopened := cmd.NewRootCmd()
			out.Reset()
			errOut.Reset()
			reopened.Out, reopened.Err, reopened.ConfigPath, reopened.HTTPClient = out, errOut, cfg, server.Client()
			if code := reopened.Execute([]string{"auth", "status", "--json"}); code != 0 || errOut.Len() != 0 {
				t.Fatalf("status code=%d stderr=%q", code, errOut.String())
			}
			var status map[string]any
			if err := json.Unmarshal(out.Bytes(), &status); err != nil {
				t.Fatal(err)
			}
			expected := map[string]any{"authenticated": true, "api_url": server.URL, "token_source": "config", "scopes": []any{"read", "write"}, "user": map[string]any{"id": float64(9), "name": "Device Writer", "email": "device@example.test"}}
			if !reflect.DeepEqual(status, expected) {
				t.Fatalf("status=%#v; want=%#v", status, expected)
			}
		})
	}
}

func checkDeviceRequest(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("request=%s %v", r.Method, r.Header)
	}
	var got, want map[string]any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("request body=%#v; want=%#v", got, want)
	}
}

func TestDeviceLoginReportsAuthorizationFailures(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		status     int
		body, want string
	}{
		{"device rejection", "/oauth/device/code", 400, `{"error":"invalid_client","error_description":"Unknown app"}`, "failed to initiate device authorization: device authorization failed: invalid_client (Unknown app)\n"},
		{"device rejection without explanation", "/oauth/device/code", 400, `{"error":"invalid_client"}`, "failed to initiate device authorization: device authorization failed: invalid_client\n"},
		{"device server failure", "/oauth/device/code", 503, `Service unavailable`, "failed to initiate device authorization: device authorization failed with HTTP status 503: Service unavailable\n"},
		{"denied", "/oauth/token", 400, `{"error":"access_denied"}`, "authorization failed: authorization denied by user\n"},
		{"expired", "/oauth/token", 400, `{"error":"expired_token"}`, "authorization failed: device authorization code has expired\n"},
		{"unknown token failure", "/oauth/token", 400, `{"error":"invalid_grant","error_description":"Code was already used"}`, "authorization failed: oauth error: invalid_grant (Code was already used)\n"},
		{"token failure without explanation", "/oauth/token", 400, `{"error":"invalid_grant"}`, "authorization failed: oauth error: invalid_grant\n"},
		{"token server failure", "/oauth/token", 502, `Bad gateway`, "authorization failed: unexpected status 502: Bad gateway\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == tc.path {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.body)
					return
				}
				if r.URL.Path == "/oauth/device/code" {
					fmt.Fprint(w, `{"device_code":"device-secret","user_code":"AB","verification_uri":"https://example.test/approve","interval":1,"expires_in":60}`)
					return
				}
				t.Errorf("unexpected request %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			if code := root.Execute([]string{"auth", "login", "--no-browser", "--json"}); code != 1 {
				t.Fatalf("code=%d; want=1", code)
			}
			if out.Len() != 0 || errOut.String() != tc.want {
				t.Fatalf("stdout=%q stderr=%q; want stderr=%q", out.String(), errOut.String(), tc.want)
			}
		})
	}
}
