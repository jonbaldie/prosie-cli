package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearRemovesSymbolicLinkTargetAndKeepsLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	if err := os.WriteFile(target, []byte(`{"token":"old-token"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Clear(link); err != nil {
		t.Fatalf("clear: %v", err)
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
}

func TestClearResolvesRelativeSymbolicLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real", "config.json")
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"token":"old-token"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(filepath.Join("real", "config.json"), link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Clear(link); err != nil {
		t.Fatalf("clear: %v", err)
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
}

func TestClearDanglingSymbolicLinkIsNotError(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(filepath.Join(dir, "missing.json"), link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Clear(link); err != nil {
		t.Fatalf("clear dangling link: %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("expected link to stay: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to stay a symbolic link, mode=%v", link, info.Mode())
	}
}

func TestClearRejectsSymbolicLinkLoop(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(link, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Clear(link); err == nil {
		t.Fatal("expected clear through a symbolic link loop to fail")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("expected link to stay: %v", err)
	}
}
