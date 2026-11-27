package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestExportRejectsUnknownFormatBeforeRequest(t *testing.T) {
	for _, resource := range []string{"chapter", "book"} {
		for _, tc := range []struct {
			name       string
			format     string
			json       bool
			writeFile  bool
			priorBytes string
		}{
			{name: "pdf stdout", format: "pdf"},
			{name: "pdf new file", format: "pdf", writeFile: true},
			{name: "pdf existing file", format: "pdf", writeFile: true, priorBytes: "KEEP\n"},
			{name: "empty format", format: "", writeFile: true, priorBytes: "KEEP\n"},
			{name: "PDF json file", format: "PDF", json: true, writeFile: true, priorBytes: "KEEP\n"},
			{name: "doc", format: "doc"},
		} {
			t.Run(resource+"/"+tc.name, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					_, _ = w.Write([]byte("# Markdown\n\nProse\n"))
				}))
				t.Cleanup(server.Close)
				t.Setenv("PROSIE_API_URL", server.URL)
				t.Setenv("PROSIE_API_TOKEN", "fixture-token")

				dir := t.TempDir()
				dest := filepath.Join(dir, "out.pdf")
				if tc.priorBytes != "" {
					if err := os.WriteFile(dest, []byte(tc.priorBytes), 0644); err != nil {
						t.Fatal(err)
					}
				}
				root, out, errOut := newTestRootCmd(filepath.Join(dir, "config.json"), server.Client())
				args := []string{resource, "export", "41", "--format", tc.format}
				if tc.writeFile {
					args = append(args, "-o", dest)
				}
				if tc.json {
					args = append(args, "--json")
				}
				code := root.Execute(args)
				wantErr := fmt.Sprintf("error: invalid --format %q: must be markdown or docx\n", tc.format)
				if code != 1 || out.Len() != 0 || errOut.String() != wantErr || requests.Load() != 0 {
					t.Fatalf("code=%d stdout=%q stderr=%q requests=%d; want 1, empty stdout, %q, 0 requests", code, out.String(), errOut.String(), requests.Load(), wantErr)
				}
				info, err := os.Stat(dest)
				if tc.priorBytes == "" {
					if !os.IsNotExist(err) {
						t.Fatalf("output file exists or stat failed: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(dest)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != tc.priorBytes || info.Size() != int64(len(tc.priorBytes)) {
					t.Fatalf("file changed to %q", data)
				}
			})
		}
	}
}

func TestExportSendsNormalizedFormat(t *testing.T) {
	for _, resource := range []struct {
		name, path string
	}{
		{"chapter", "/api/scenes/41/export"},
		{"book", "/api/stories/41/export"},
	} {
		for _, tc := range []struct {
			name   string
			args   []string
			format string
		}{
			{name: "omitted", format: "markdown"},
			{name: "markdown", args: []string{"--format", "markdown"}, format: "markdown"},
			{name: "Markdown", args: []string{"--format", "Markdown"}, format: "markdown"},
			{name: "docx", args: []string{"--format", "docx"}, format: "docx"},
			{name: "DOCX", args: []string{"--format", "DOCX"}, format: "docx"},
		} {
			t.Run(resource.name+"/"+tc.name, func(t *testing.T) {
				var requests atomic.Int32
				var gotFormat atomic.Value
				gotFormat.Store("")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != resource.path {
						t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					}
					gotFormat.Store(r.URL.Query().Get("format"))
					_, _ = w.Write([]byte("# Markdown\n\nProse\n"))
				}))
				t.Cleanup(server.Close)
				t.Setenv("PROSIE_API_URL", server.URL)
				t.Setenv("PROSIE_API_TOKEN", "fixture-token")
				root, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), server.Client())
				args := append([]string{resource.name, "export", "41"}, tc.args...)
				if code := root.Execute(args); code != 0 || errOut.Len() != 0 {
					t.Fatalf("code=%d stderr=%q stdout=%q", code, errOut.String(), out.String())
				}
				if requests.Load() != 1 || gotFormat.Load() != tc.format {
					t.Fatalf("requests=%d format=%q; want 1 and %q", requests.Load(), gotFormat.Load(), tc.format)
				}
				if out.String() != "# Markdown\n\nProse\n" {
					t.Fatalf("stdout=%q", out.String())
				}
			})
		}
	}
}
