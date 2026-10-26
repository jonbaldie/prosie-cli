//go:build linux || darwin

package cmd_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestExportWriteLimitHelper(t *testing.T) {
	if os.Getenv("PROSIE_TEST_EXPORT_CHILD") != "1" {
		return
	}
	if os.Getenv("PROSIE_TEST_WRITE_LIMIT") == "1" {
		signal.Ignore(syscall.SIGXFSZ)
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 64, Max: 64}); err != nil {
			panic(err)
		}
	}
	root := cmd.NewRootCmd()
	os.Exit(root.Execute([]string{os.Getenv("PROSIE_TEST_RESOURCE"), "export", "41", "--output", os.Getenv("PROSIE_TEST_OUTPUT"), "--json"}))
}

func TestExportToDeviceStdout(t *testing.T) {
	const content = "export bytes\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer server.Close()
	for _, resource := range []string{"book", "chapter", "chat"} {
		t.Run(resource, func(t *testing.T) {
			child := exec.Command(os.Args[0], "-test.run=^TestExportWriteLimitHelper$")
			child.Env = append(os.Environ(), "PROSIE_TEST_EXPORT_CHILD=1", "PROSIE_TEST_WRITE_LIMIT=0", "PROSIE_TEST_RESOURCE="+resource, "PROSIE_TEST_OUTPUT=/dev/stdout", "PROSIE_API_URL="+server.URL, "PROSIE_API_TOKEN=fixture-token", "PROSIE_CONFIG_PATH="+filepath.Join(t.TempDir(), "absent.json"))
			var out, errOut bytes.Buffer
			child.Stdout, child.Stderr = &out, &errOut
			if err := child.Run(); err != nil || errOut.Len() != 0 {
				t.Fatalf("exit=%v stderr=%q", err, errOut.String())
			}
			id := "41"
			if resource == "chat" {
				id = `"41"`
			}
			want := content + fmt.Sprintf("{\n  \"bytes\": %d,\n  \"id\": %s,\n  \"output\": \"/dev/stdout\"\n}\n", len(content), id)
			if out.String() != want {
				t.Fatalf("stdout=%q want=%q", out.String(), want)
			}
		})
	}
}

func TestFailedExportKeepsPreviousFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("N", 4096))
	}))
	defer server.Close()
	for _, resource := range []string{"book", "chapter", "chat"} {
		t.Run(resource, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, "export.md")
			previous := bytes.Repeat([]byte("O"), 256)
			if err := os.WriteFile(dest, previous, 0644); err != nil {
				t.Fatal(err)
			}
			child := exec.Command(os.Args[0], "-test.run=^TestExportWriteLimitHelper$")
			child.Env = append(os.Environ(), "PROSIE_TEST_EXPORT_CHILD=1", "PROSIE_TEST_WRITE_LIMIT=1", "PROSIE_TEST_RESOURCE="+resource, "PROSIE_TEST_OUTPUT="+dest, "PROSIE_API_URL="+server.URL, "PROSIE_API_TOKEN=fixture-token", "PROSIE_CONFIG_PATH="+filepath.Join(dir, "absent.json"))
			var out, errOut bytes.Buffer
			child.Stdout, child.Stderr = &out, &errOut
			err := child.Run()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("exit=%v stderr=%q", err, errOut.String())
			}
			if out.Len() != 0 || !strings.Contains(errOut.String(), "error writing to file:") || !strings.Contains(errOut.String(), "file too large") {
				t.Fatalf("stdout=%q stderr=%q", out.String(), errOut.String())
			}
			got, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, previous) {
				t.Errorf("previous export lost: got %d bytes, want %d unchanged bytes", len(got), len(previous))
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Errorf("temporary files remain: %v", entries)
			}
		})
	}
}
