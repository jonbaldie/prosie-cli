package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/config"
)

func TestAuthLogoutRemovesSymbolicLinkTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	if err := os.WriteFile(target, []byte(`{"api_url":"https://example.test/api","token":"old-token","scopes":["read"]}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	t.Setenv("PROSIE_API_TOKEN", "")

	cmd, out, errOut := newTestRootCmd(link, nil)
	code := cmd.Execute([]string{"auth", "logout"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Logged out from Prosie") {
		t.Fatalf("expected logout message on stdout, got: %s", out.String())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected target to be removed, stat err: %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("expected link to stay: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to stay a symbolic link, mode=%v", link, info.Mode())
	}
	// After logout, the dangling link reads as no token.
	loaded, err := config.Load(link)
	if err != nil {
		t.Fatalf("load after logout: %v", err)
	}
	// ResolveToken reads the environment; ensure empty observed source.
	t.Setenv("PROSIE_API_TOKEN", "")
	token, source := config.ResolveToken(loaded)
	if token != "" || source != "none" {
		t.Fatalf("expected no token after logout, got %q from %q", token, source)
	}
}

func TestAuthLogoutSymbolicLinkLoopFails(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(link, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	t.Setenv("PROSIE_API_TOKEN", "")

	cmd, out, errOut := newTestRootCmd(link, nil)
	code := cmd.Execute([]string{"auth", "logout"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d. stdout: %s stderr: %s", code, out.String(), errOut.String())
	}
	if strings.Contains(out.String(), "Logged out from Prosie") {
		t.Fatalf("expected no success message on stdout, got: %s", out.String())
	}
	if errOut.Len() == 0 {
		t.Fatal("expected error on stderr, got empty stderr")
	}
}
