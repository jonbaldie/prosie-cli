package cmd

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestContinueLengthRequest(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		body  map[string]any
	}{
		{"word choice", []string{"--length", "w900"}, map[string]any{"persist": false, "length": "w900"}},
		{"line choice", []string{"--length", "12"}, map[string]any{"persist": false, "length": "12"}},
		{"empty", []string{"--length", ""}, map[string]any{"persist": false}},
		{"omitted", nil, map[string]any{"persist": false}},
		{"explicit limits", []string{"--length", "w900", "--words", "500", "--lines", "12"}, map[string]any{"persist": false, "length": "w900", "word_target": float64(500), "line_limit": float64(12)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			_, path, httpClient := setupTestGenerateEnv(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/scenes/1/continue" {
					t.Errorf("request: %s %s", r.Method, r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				_, _ = w.Write([]byte(`{"data":{"prose":"fixture prose","persisted":false}}`))
			})
			cmd, out, errOut := newTestRootCmd(path, httpClient)
			args := append([]string{"generate", "continue", "1", "--json", "--no-persist"}, tc.flags...)
			code := cmd.Execute(args)
			if code != 0 || errOut.Len() != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut.String())
			}
			if !reflect.DeepEqual(got, tc.body) {
				t.Errorf("body = %#v; want %#v", got, tc.body)
			}
			var response struct {
				Prose     string `json:"prose"`
				Persisted bool   `json:"persisted"`
			}
			if err := json.Unmarshal(out.Bytes(), &response); err != nil || response.Prose != "fixture prose" || response.Persisted {
				t.Errorf("response %q: %v", out.String(), err)
			}
		})
	}
}
