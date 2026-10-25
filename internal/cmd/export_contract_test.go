package cmd_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestProseExportPreservesBytesAndReportsTheSavedFile(t *testing.T) {
	for _, resource := range []struct{ name, path string }{{"book", "/api/stories/41/export"}, {"chapter", "/api/scenes/41/export"}} {
		for _, mode := range []string{"stdout", "json", "file", "file-json"} {
			t.Run(resource.name+"/"+mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" || r.URL.Path != resource.path || r.URL.Query().Get("format") != "markdown" {
						t.Errorf("unexpected export request: %s %s", r.Method, r.URL)
					}
					fmt.Fprint(w, "é\n\nEnd.")
				}))
				defer server.Close()
				t.Setenv("PROSIE_API_URL", server.URL)
				t.Setenv("PROSIE_API_TOKEN", "fixture-token")
				root := cmd.NewRootCmd()
				out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
				dir := t.TempDir()
				dest := filepath.Join(dir, "export.md")
				root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(dir, "config.json"), server.Client()
				args := []string{resource.name, "export", "41"}
				if mode == "file" || mode == "file-json" {
					args = append(args, "--output", dest)
				}
				if mode == "json" || mode == "file-json" {
					args = append(args, "--json")
				}
				if code := root.Execute(args); code != 0 || errOut.Len() != 0 {
					t.Fatalf("code=%d stderr=%q", code, errOut.String())
				}
				if mode == "file" || mode == "file-json" {
					data, err := os.ReadFile(dest)
					if err != nil {
						t.Fatal(err)
					}
					if string(data) != "é\n\nEnd." {
						t.Fatalf("saved prose=%q", data)
					}
				}
				switch mode {
				case "stdout":
					if out.String() != "é\n\nEnd." {
						t.Fatalf("prose=%q", out.String())
					}
				case "file":
					want := fmt.Sprintf("Exported %s 41 to %s (8 bytes).\n", resource.name, dest)
					if out.String() != want {
						t.Fatalf("output=%q; want=%q", out.String(), want)
					}
				default:
					var got map[string]any
					if err := json.Unmarshal(out.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					want := map[string]any{"id": float64(41), "bytes": float64(8), "content": "é\n\nEnd."}
					if mode == "file-json" {
						want = map[string]any{"id": float64(41), "bytes": float64(8), "output": dest}
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("metadata=%#v; want=%#v", got, want)
					}
				}
			})
		}
	}
}

func TestDocxExportAsJSONCarriesTheExactBytesAsBase64(t *testing.T) {
	docx := []byte("PK\x03\x04\xff\xfe\x00\x80")
	for _, resource := range []struct{ name, path string }{{"book", "/api/stories/41/export"}, {"chapter", "/api/scenes/41/export"}} {
		t.Run(resource.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != resource.path || r.URL.Query().Get("format") != "docx" {
					t.Errorf("unexpected export request: %s %s", r.Method, r.URL)
				}
				_, _ = w.Write(docx)
			}))
			defer server.Close()
			t.Setenv("PROSIE_API_URL", server.URL)
			t.Setenv("PROSIE_API_TOKEN", "fixture-token")
			root := cmd.NewRootCmd()
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			root.Out, root.Err, root.ConfigPath, root.HTTPClient = out, errOut, filepath.Join(t.TempDir(), "config.json"), server.Client()
			if code := root.Execute([]string{resource.name, "export", "41", "--format", "docx", "--json"}); code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
			var got struct {
				ID       int    `json:"id"`
				Content  string `json:"content"`
				Encoding string `json:"encoding"`
				Bytes    int    `json:"bytes"`
			}
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			decoded, err := base64.StdEncoding.DecodeString(got.Content)
			if err != nil {
				t.Fatalf("content %q is not base64: %v", got.Content, err)
			}
			if got.ID != 41 || got.Encoding != "base64" || got.Bytes != len(docx) || !bytes.Equal(decoded, docx) {
				t.Fatalf("export=%+v decoded=%q; want encoding=base64 bytes=%d content=%q", got, decoded, len(docx), docx)
			}
		})
	}
}
