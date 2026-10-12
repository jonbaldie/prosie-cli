package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
)

func TestProseInputHelp(t *testing.T) {
	for _, args := range [][]string{
		{"chapter", "create", "--help"},
		{"chapter", "update", "--help"},
		{"generate", "rewrite", "--help"},
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			cmd, out, errOut := newTestRootCmd("", nil)
			if code := cmd.Execute(args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			if !strings.Contains(out.String(), "'-' for standard input") || !strings.Contains(out.String(), "not both") {
				t.Fatalf("input rules missing from help: %s", out.String())
			}
		})
	}
}

func TestChapterContentPatch(t *testing.T) {
	for _, action := range []string{"create", "update"} {
		for _, tc := range []struct {
			name, input, want string
			flags             []string
			present           bool
		}{
			{"omitted", "Unused stdin.", "", nil, false},
			{"empty file flag", "Unused stdin.", "", []string{"--file", ""}, false},
			{"inline", "Unused stdin.", "Inline prose.", []string{"--content", "Inline prose."}, true},
			{"empty inline", "Unused stdin.", "", []string{"--content", ""}, true},
			{"empty stdin", "", "", []string{"--file", "-"}, true},
			{"empty inline with stdin", "File prose.", "File prose.", []string{"--content", "", "--file", "-"}, true},
			{"inline with empty file flag", "Unused stdin.", "Inline prose.", []string{"--content", "Inline prose.", "--file", ""}, true},
		} {
			t.Run(action+"/"+tc.name, func(t *testing.T) {
				var body map[string]any
				handler := func(w http.ResponseWriter, r *http.Request) {
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"data":{"id":101}}`))
				}
				_, cfgPath, httpClient := setupTestChapterEnv(t, handler)
				cmd, _, errOut := newTestRootCmd(cfgPath, httpClient)
				cmd.In = strings.NewReader(tc.input)
				args := append([]string{"chapter", action, "1", "--json"}, tc.flags...)
				if code := cmd.Execute(args); code != 0 || errOut.Len() != 0 {
					t.Fatalf("code=%d stderr=%q", code, errOut.String())
				}
				got, present := body["content"]
				if present != tc.present || (present && got != tc.want) {
					t.Fatalf("request body=%#v, want content=%q present=%v", body, tc.want, tc.present)
				}
			})
		}
	}
}

func TestRewriteEmptyStdin(t *testing.T) {
	for _, input := range []string{"", " \n\t"} {
		cmd, out, errOut := newTestRootCmd("", nil)
		cmd.In = strings.NewReader(input)
		if code := cmd.Execute([]string{"generate", "rewrite", "101", "--selection-file", "-", "--action", "tighten", "--json"}); code != 1 || out.Len() != 0 {
			t.Fatalf("code=%d stdout=%q", code, out.String())
		}
		if errOut.String() != "error: --selection or --selection-file is required\n" {
			t.Fatalf("unexpected stderr: %s", errOut.String())
		}
	}
}

func TestProseReadErrors(t *testing.T) {
	for _, tc := range []struct {
		name, flag, label string
		args              []string
	}{
		{"chapter create", "--file", "content", []string{"chapter", "create", "1"}},
		{"chapter update", "--file", "content", []string{"chapter", "update", "101"}},
		{"rewrite", "--selection-file", "selection", []string{"generate", "rewrite", "101", "--action", "tighten"}},
	} {
		for _, path := range []string{"-", "/nonexistent/prosie-content.txt"} {
			t.Run(tc.name+"/"+path, func(t *testing.T) {
				cmd, out, errOut := newTestRootCmd("", nil)
				cmd.In = iotest.ErrReader(errors.New("input failed"))
				args := append(append([]string{}, tc.args...), tc.flag, path, "--json")
				if code := cmd.Execute(args); code != 1 || out.Len() != 0 {
					t.Fatalf("code=%d stdout=%q", code, out.String())
				}
				if !strings.HasPrefix(errOut.String(), "error reading "+tc.label+" file: ") {
					t.Fatalf("unexpected stderr: %s", errOut.String())
				}
				if path == "-" && !strings.Contains(errOut.String(), "input failed") {
					t.Fatalf("read error missing from stderr: %s", errOut.String())
				}
			})
		}
	}
}

func TestProseConflictingInputs(t *testing.T) {
	for _, args := range [][]string{
		{"chapter", "create", "1", "--content", "Inline prose.", "--file", "-"},
		{"chapter", "update", "101", "--content", "Inline prose.", "--file", "-"},
		{"generate", "rewrite", "101", "--selection", "Inline prose.", "--selection-file", "-", "--action", "tighten"},
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			cmd, out, errOut := newTestRootCmd("", nil)
			cmd.In = iotest.ErrReader(errors.New("must not read input"))
			args = append(args, "--json")
			if code := cmd.Execute(args); code != 1 || out.Len() != 0 {
				t.Fatalf("code=%d stdout=%q", code, out.String())
			}
			if !strings.Contains(errOut.String(), "not both") || strings.Contains(errOut.String(), "must not read input") {
				t.Fatalf("unexpected stderr: %s", errOut.String())
			}
		})
	}
}

func TestProseFromStdin(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, key string
		args                    []string
	}{
		{"chapter create", http.MethodPost, "/api/stories/1/scenes", "content", []string{"chapter", "create", "1", "--file", "-", "--json"}},
		{"chapter update", http.MethodPatch, "/api/scenes/101", "content", []string{"chapter", "update", "101", "--file", "-", "--json"}},
		{"rewrite", http.MethodPost, "/api/scenes/101/rewrite", "selection", []string{"generate", "rewrite", "101", "--selection-file", "-", "--action", "tighten", "--json"}},
		{"stream rewrite", http.MethodPost, "/api/scenes/101/rewrite/stream", "selection", []string{"generate", "rewrite", "101", "--selection-file", "-", "--action", "tighten", "--stream"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if strings.HasSuffix(tc.path, "/stream") {
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"Rewritten prose.\"}\n\nevent: done\ndata: {\"prose\":\"Rewritten prose.\"}\n\n"))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"id":101,"prose":"Rewritten prose."}}`))
			}
			_, cfgPath, httpClient := setupTestChapterEnv(t, handler)
			cmd, out, errOut := newTestRootCmd(cfgPath, httpClient)
			cmd.In = strings.NewReader("  The stars shone.\n\nCafé.\n")
			if code := cmd.Execute(tc.args); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			if body[tc.key] != "  The stars shone.\n\nCafé.\n" {
				t.Fatalf("request body = %#v", body)
			}
			if strings.HasSuffix(tc.path, "/stream") {
				if out.String() != "Rewritten prose.\n" {
					t.Fatalf("unexpected stream output: %s", out.String())
				}
				return
			}
			if !json.Valid(out.Bytes()) {
				t.Fatalf("invalid JSON output: %s", out.String())
			}
		})
	}
}
