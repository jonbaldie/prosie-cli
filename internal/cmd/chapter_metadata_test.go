package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func chapterMetadataEnv(t *testing.T, handler http.HandlerFunc) (string, *http.Client) {
	t.Helper()
	t.Setenv("PROSIE_API_URL", "https://example.test")
	t.Setenv("PROSIE_API_TOKEN", "test-token")
	client := &http.Client{Transport: deleteTransport(func(r *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		handler(response, r)
		return response.Result(), nil
	})}
	return filepath.Join(t.TempDir(), "config.json"), client
}

func TestChapterMetadataRequests(t *testing.T) {
	for _, command := range []struct {
		name, id, method, path string
	}{
		{"create", "1", http.MethodPost, "/api/stories/1/scenes"},
		{"update", "101", http.MethodPatch, "/api/scenes/101"},
	} {
		t.Run(command.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "chapter.txt")
			if err := os.WriteFile(file, []byte("Prose from file."), 0600); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name  string
				flags []string
				want  map[string]any
			}{
				{"omitted", nil, map[string]any{}},
				{"all", []string{"--pov", "first", "--character", "Mara", "--order", "3"}, map[string]any{"custom_pov": "first", "custom_character": "Mara", "order": float64(3)}},
				{"pov alone", []string{"--pov", "third"}, map[string]any{"custom_pov": "third"}},
				{"character alone", []string{"--character", "Mara"}, map[string]any{"custom_character": "Mara"}},
				{"order alone", []string{"--order", "2"}, map[string]any{"order": float64(2)}},
				{"explicit empty and zero", []string{"--pov=", "--character=", "--order=0"}, map[string]any{"custom_pov": "", "custom_character": "", "order": float64(0)}},
				{"existing content", []string{"--title", "Arrival", "--content", "Docking.", "--summary", "At the station.", "--pov", "first"}, map[string]any{"name": "Arrival", "content": "Docking.", "summary": "At the station.", "custom_pov": "first"}},
				{"file content", []string{"--file", file, "--character", "Mara", "--order", "0"}, map[string]any{"content": "Prose from file.", "custom_character": "Mara", "order": float64(0)}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var got map[string]any
					var requests atomic.Int32
					cfgPath, httpClient := chapterMetadataEnv(t, func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						if r.Method != command.method || r.URL.Path != command.path {
							t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, command.method, command.path)
						}
						if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
							t.Errorf("invalid request JSON: %v", err)
						}
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"data":{"id":101,"name":"Arrival"}}`))
					})
					cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
					args := append([]string{"chapter", command.name, command.id}, tc.flags...)
					code := cmd.Execute(append(args, "--json"))
					if code != 0 || errOut.Len() != 0 {
						t.Fatalf("exit = %d, stderr = %q; want 0 and empty stderr", code, errOut.String())
					}
					if requests.Load() != 1 || !reflect.DeepEqual(got, tc.want) {
						t.Fatalf("requests = %d, body = %#v; want 1, %#v", requests.Load(), got, tc.want)
					}
					if !json.Valid(out.Bytes()) {
						t.Fatalf("invalid output JSON: %q", out.String())
					}
				})
			}
		})
	}
}

func TestChapterMetadataInvalidOrder(t *testing.T) {
	for _, command := range []struct{ name, id string }{{"create", "1"}, {"update", "101"}} {
		t.Run(command.name, func(t *testing.T) {
			var requests atomic.Int32
			cfgPath, httpClient := chapterMetadataEnv(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
			})
			cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
			code := cmd.Execute([]string{"chapter", command.name, command.id, "--order", "first"})
			if code != 1 || out.Len() != 0 || !strings.Contains(errOut.String(), `invalid value "first" for flag -order`) || requests.Load() != 0 {
				t.Fatalf("exit = %d, stdout = %q, stderr = %q, requests = %d", code, out.String(), errOut.String(), requests.Load())
			}
		})
	}
}

func TestChapterMetadataHelp(t *testing.T) {
	for _, command := range []string{"create", "update"} {
		t.Run(command, func(t *testing.T) {
			cmd, out, errOut := newTestRootCmd("", nil)
			code := cmd.Execute([]string{"chapter", command, "--help"})
			if code != 0 || errOut.Len() != 0 {
				t.Fatalf("exit = %d, stderr = %q", code, errOut.String())
			}
			for _, flag := range []string{"--pov", "--character", "--order"} {
				if !strings.Contains(out.String(), flag) {
					t.Errorf("help does not include %s: %s", flag, out.String())
				}
			}
		})
	}
}
