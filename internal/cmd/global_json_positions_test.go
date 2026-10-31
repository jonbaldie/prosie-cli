package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/version"
)

func TestJSONPositionsByCommandFamily(t *testing.T) {
	t.Setenv("PROSIE_API_URL", "https://example.test")
	t.Setenv("PROSIE_API_TOKEN", "test-token")
	previousVersion := version.Version
	version.Version = "1.2.3-test"
	t.Cleanup(func() { version.Version = previousVersion })

	type fixture struct {
		method, response string
		body             map[string]any
	}
	for _, tc := range []struct {
		family, command string
		flags, operands []string
		requests        map[string]fixture
		field           string
		want            any
	}{
		{"auth", "status", nil, nil, map[string]fixture{"/api/user": {"GET", `{"id":7,"name":"Jane","email":"jane@example.com"}`, nil}}, "authenticated", true},
		{"book", "show", nil, []string{"1"}, map[string]fixture{"/api/stories/1": {"GET", `{"data":{"id":1,"title":"The Starlight Gate"}}`, nil}, "/api/stories/1/scenes": {"GET", `{"data":[]}`, nil}}, "title", "The Starlight Gate"},
		{"chapter", "show", nil, []string{"101"}, map[string]fixture{"/api/scenes/101": {"GET", `{"data":{"id":101,"name":"Chapter 1","content":"Prose"}}`, nil}}, "name", "Chapter 1"},
		{"chat", "send", []string{"--conversation", "777"}, []string{"Hello"}, map[string]fixture{"/api/conversations/777/messages": {"POST", `{"data":{"message":{"id":2,"conversation_id":777,"role":"assistant","content":"Reply"},"model":"test"}}`, map[string]any{"content": "Hello"}}}, "conversation_id", float64(777)},
		{"codex", "show", nil, []string{"10"}, map[string]fixture{"/api/codex-entries/10": {"GET", `{"data":{"id":10,"name":"Marcus Vance","category":"character","content":"Chief pilot"}}`, nil}}, "name", "Marcus Vance"},
		{"generate", "continue", []string{"--instruction", "Keep going"}, []string{"101"}, map[string]fixture{"/api/scenes/101/continue": {"POST", `{"data":{"prose":"Hello","persisted":true}}`, map[string]any{"instruction": "Keep going"}}}, "prose", "Hello"},
		{"llm", "show", nil, nil, map[string]fixture{"/api/llm-config": {"GET", `{"data":{"provider":"openrouter","model":"test-model","has_openrouter_key":true}}`, nil}}, "model", "test-model"},
		{"series", "show", nil, []string{"1"}, map[string]fixture{"/api/series/1": {"GET", `{"data":{"id":1,"name":"The Solar Cycle"}}`, nil}, "/api/stories": {"GET", `{"data":[]}`, nil}, "/api/series/1/codex-entries": {"GET", `{"data":[]}`, nil}}, "title", "The Solar Cycle"},
		{"version", "", nil, nil, nil, "version", "1.2.3-test"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			base := []string{tc.family}
			if tc.command != "" {
				base = append(base, tc.command)
			}
			with := func(parts ...[]string) []string {
				var result []string
				for _, part := range parts {
					result = append(result, part...)
				}
				return result
			}
			positions := map[string][]string{
				"leading":           with([]string{"--json"}, base, tc.flags, tc.operands),
				"leading_separator": with([]string{"--json"}, base, tc.flags, []string{"--"}, tc.operands),
				"trailing":          with(base, tc.flags, tc.operands, []string{"--json"}),
				"before_separator":  with(base, tc.flags, []string{"--json", "--"}, tc.operands),
			}
			if tc.family == "chat" {
				positions["separator_literal"] = with(base, tc.flags, []string{"--json", "--", "--json"})
			}
			if tc.command != "" {
				positions["family"] = with([]string{tc.family, "--json", tc.command}, tc.flags, tc.operands)
			}
			for name, args := range positions {
				t.Run(name, func(t *testing.T) {
					seen := make(map[string]int)
					client := &http.Client{Transport: deleteTransport(func(r *http.Request) (*http.Response, error) {
						f, ok := tc.requests[r.URL.Path]
						if !ok {
							t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
						}
						seen[r.URL.Path]++
						if r.Method != f.method || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer test-token" {
							t.Errorf("unexpected request: %s %s, authorization %q", r.Method, r.URL.String(), r.Header.Get("Authorization"))
						}
						if f.body != nil {
							var body map[string]any
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Errorf("decode request body: %v", err)
							}
							for key, want := range f.body {
								if name == "separator_literal" && key == "content" {
									want = "--json"
								}
								if !reflect.DeepEqual(body[key], want) {
									t.Errorf("body[%q] = %v, want %v", key, body[key], want)
								}
							}
						} else if r.Body != nil {
							data, err := io.ReadAll(r.Body)
							if err != nil || len(data) != 0 {
								t.Errorf("unexpected request body %q: %v", data, err)
							}
						}
						return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response))}, nil
					})}
					root, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), client)
					code := root.Execute(args)
					if code != 0 || errOut.Len() != 0 {
						t.Fatalf("%v: exit %d, stdout %q, stderr %q", args, code, out.String(), errOut.String())
					}
					var output map[string]any
					if err := json.Unmarshal(out.Bytes(), &output); err != nil {
						t.Fatalf("invalid JSON %q: %v", out.String(), err)
					}
					if !reflect.DeepEqual(output[tc.field], tc.want) {
						t.Errorf("output[%q] = %v, want %v; output %s", tc.field, output[tc.field], tc.want, out.String())
					}
					if tc.family == "chat" {
						message, ok := output["message"].(map[string]any)
						if !ok || message["content"] != "Reply" {
							t.Errorf("unexpected chat message: %v", output["message"])
						}
					}
					for path := range tc.requests {
						if seen[path] != 1 {
							t.Errorf("requests to %s = %d, want 1", path, seen[path])
						}
					}
				})
			}
		})
	}
}
