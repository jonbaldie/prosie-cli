package cmd

import (
	"bytes"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormalTransfersOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}
	var errors bytes.Buffer
	input := bytes.NewBufferString("stdin")
	env := &command.Environment{In: input, Err: &errors}
	file, closeFile, ok := command.OpenUpload(env, path)
	if !ok {
		t.Fatal(errors.String())
	}
	closeFile()
	if _, err := file.Reader.Read(make([]byte, 1)); err == nil {
		t.Fatal("opened file remains usable after cleanup")
	}
	stdin, closeStdin, ok := command.OpenUpload(env, "-")
	if !ok {
		t.Fatal(errors.String())
	}
	closeStdin()
	if b, err := io.ReadAll(stdin.Reader); err != nil || string(b) != "stdin" {
		t.Fatalf("stdin changed by cleanup: %q %v", b, err)
	}
}

type formalTransfersFailWriter struct{}

func (formalTransfersFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestFormalTransfersFetchAndReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export")
	_, cfg, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("destination exists before fetch: %v", err)
		}
		_, _ = io.WriteString(w, "raw bytes")
	})
	cmd, _, _ := newTestRootCmd(cfg, httpClient)
	cmd.Out = formalTransfersFailWriter{}
	if code := cmd.Execute([]string{"chat", "export", "100", "-o", path}); code == 0 {
		t.Fatal("output failure did not change exit status")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "raw bytes" {
		t.Fatalf("publication lost after output failure: %q %v", data, err)
	}
}

func TestFormalTransfersRouting(t *testing.T) {
	_, cfg, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not json")
	})
	path := filepath.Join(t.TempDir(), "export.json")
	cmd, _, errout := newTestRootCmd(cfg, httpClient)
	if code := cmd.Execute([]string{"chat", "export", "100", "--json", "-o", path}); code != 0 {
		t.Fatalf("file route rejected raw bytes: %d %s", code, errout.String())
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "not json" {
		t.Fatalf("file route lost raw bytes: %q %v", data, err)
	}
	cmd, _, _ = newTestRootCmd(cfg, httpClient)
	if code := cmd.Execute([]string{"chat", "export", "100", "--json"}); code == 0 {
		t.Fatal("stdout JSON accepted malformed bytes")
	}
	for _, payload := range []string{"not json", "not json\n", "{\"ok\":true}", "{\"ok\":true}\n"} {
		t.Run(payload, func(t *testing.T) {
			_, cfg, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, payload)
			})
			cmd, out, errout := newTestRootCmd(cfg, httpClient)
			if code := cmd.Execute([]string{"chat", "export", "100"}); code != 0 {
				t.Fatalf("text route failed: %d %s", code, errout.String())
			}
			want := payload
			if !strings.HasSuffix(payload, "\n") {
				want += "\n"
			}
			if out.String() != want {
				t.Fatalf("text route changed trailing newline: got %q want %q", out.String(), want)
			}
		})
	}
}

func TestFormalTransfersExport(t *testing.T) {
	var out, errout bytes.Buffer
	c := &command.Environment{Out: &out, Err: &errout}
	path := filepath.Join(t.TempDir(), "missing", "export")
	code := command.WriteExportFile(c, "book", 1, path, []byte("data"), false)
	if code == 0 || out.Len() != 0 || errout.Len() == 0 {
		t.Fatalf("failed export reported success: code=%d out=%q err=%q", code, out.String(), errout.String())
	}
}
