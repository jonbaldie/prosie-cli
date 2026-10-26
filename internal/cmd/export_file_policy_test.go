package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestExportFileReplacementPolicy(t *testing.T) {
	const content = "complete new export\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, content) }))
	defer server.Close()
	t.Setenv("PROSIE_API_URL", server.URL)
	t.Setenv("PROSIE_API_TOKEN", "fixture-token")
	for _, resource := range []string{"book", "chapter", "chat"} {
		for _, policy := range []string{"new", "existing", "link", "dangling-link"} {
			t.Run(resource+"/"+policy, func(t *testing.T) {
				dir := t.TempDir()
				dest, target := filepath.Join(dir, "export.md"), filepath.Join(dir, "export.md")
				wantMode := os.FileMode(0644)
				if policy == "link" || policy == "dangling-link" {
					target = filepath.Join(dir, "target.md")
					if err := os.Symlink("target.md", dest); err != nil {
						t.Fatal(err)
					}
				}
				if policy == "existing" || policy == "link" {
					if err := os.WriteFile(target, bytes.Repeat([]byte("old"), 100), 0600); err != nil {
						t.Fatal(err)
					}
					// Set the mode explicitly so the test does not depend on the umask.
					wantMode = 0640
					if err := os.Chmod(target, wantMode); err != nil {
						t.Fatal(err)
					}
				}
				root := cmd.NewRootCmd()
				var out, errOut bytes.Buffer
				root.Out, root.Err, root.ConfigPath = &out, &errOut, filepath.Join(dir, "absent.json")
				if code := root.Execute([]string{resource, "export", "41", "--output", dest, "--json"}); code != 0 || errOut.Len() != 0 {
					t.Fatalf("code=%d stderr=%q", code, errOut.String())
				}
				data, err := os.ReadFile(target)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != content {
					t.Fatalf("saved=%q", data)
				}
				info, err := os.Stat(target)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != wantMode {
					t.Errorf("mode=%o want=%o", info.Mode().Perm(), wantMode)
				}
				if policy == "link" || policy == "dangling-link" {
					link, err := os.Readlink(dest)
					if err != nil || link != "target.md" {
						t.Errorf("link=%q err=%v", link, err)
					}
				}
				var metadata struct {
					ID     any
					Output string
					Bytes  int
				}
				if err := json.Unmarshal(out.Bytes(), &metadata); err != nil {
					t.Fatal(err)
				}
				wantID := any(float64(41))
				if resource == "chat" {
					wantID = "41"
				}
				if metadata.ID != wantID || metadata.Output != dest || metadata.Bytes != len(content) {
					t.Errorf("metadata=%+v", metadata)
				}
				entries, err := os.ReadDir(dir)
				if err != nil {
					t.Fatal(err)
				}
				wantCount := 1
				if policy == "link" || policy == "dangling-link" {
					wantCount = 2
				}
				if len(entries) != wantCount {
					t.Errorf("extra files: %v", entries)
				}
			})
		}
	}
}
