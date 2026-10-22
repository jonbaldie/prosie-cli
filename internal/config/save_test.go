package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func assertPrivate(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected 0600 permissions, got %04o", perm)
	}
}

func assertLoadedToken(t *testing.T, path, want string) {
	t.Helper()
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load after save: %v", err)
	}
	if loaded.Token != want {
		t.Fatalf("token=%q; want %q", loaded.Token, want)
	}
}

func assertOnlyEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name()] = true
	}
	if len(got) != len(want) {
		t.Fatalf("directory entries=%v; want %v", got, want)
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("directory entries=%v; want %v", got, want)
		}
	}
}

func TestSaveMakesExistingPermissiveConfigPrivate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	if err := Save(path, &Config{Token: "new-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	assertPrivate(t, path)
	assertLoadedToken(t, path, "new-token")
	assertOnlyEntries(t, dir, "config.json")
}

func TestSaveThroughSymbolicLinkUpdatesTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real", "config.json")
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"token":"old-token"}`), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(filepath.Join("real", "config.json"), link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Save(link, &Config{Token: "new-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to stay a symbolic link, mode=%v", link, info.Mode())
	}
	assertLoadedToken(t, target, "new-token")
	assertLoadedToken(t, link, "new-token")
	assertPrivate(t, target)
	assertOnlyEntries(t, filepath.Dir(target), "config.json")
}

func TestSaveThroughDanglingSymbolicLinkCreatesTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config-target.json")
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink("config-target.json", link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Save(link, &Config{Token: "new-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to stay a symbolic link, mode=%v", link, info.Mode())
	}
	assertLoadedToken(t, target, "new-token")
	assertPrivate(t, target)
}

func TestSaveRejectsSymbolicLinkLoop(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(link, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if err := Save(link, &Config{Token: "new-token"}); err == nil {
		t.Fatal("expected save through a symbolic link loop to fail")
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to stay a symbolic link, err=%v", link, err)
	}
	assertOnlyEntries(t, dir, "config.json")
}

type failingFile struct {
	tempFile
	failWrite, failSync, failClose bool
}

func (f *failingFile) Write(p []byte) (int, error) {
	if f.failWrite {
		return 0, os.ErrInvalid
	}
	return f.tempFile.Write(p)
}

func (f *failingFile) Sync() error {
	if f.failSync {
		return os.ErrInvalid
	}
	return f.tempFile.Sync()
}

func (f *failingFile) Close() error {
	err := f.tempFile.Close()
	if f.failClose {
		return os.ErrInvalid
	}
	return err
}

func TestSaveFailureKeepsPreviousConfig(t *testing.T) {
	failures := map[string]func(t *testing.T){
		"create": func(t *testing.T) {
			original := createTemp
			createTemp = func(string, string) (tempFile, error) { return nil, os.ErrPermission }
			t.Cleanup(func() { createTemp = original })
		},
		"write": func(t *testing.T) { injectFile(t, &failingFile{failWrite: true}) },
		"sync":  func(t *testing.T) { injectFile(t, &failingFile{failSync: true}) },
		"close": func(t *testing.T) { injectFile(t, &failingFile{failClose: true}) },
		"rename": func(t *testing.T) {
			original := rename
			rename = func(string, string) error { return os.ErrInvalid }
			t.Cleanup(func() { rename = original })
		},
	}
	for name, inject := range failures {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			old := []byte(`{"token":"old-token"}` + "\n")
			if err := os.WriteFile(path, old, 0600); err != nil {
				t.Fatal(err)
			}
			inject(t)

			if err := Save(path, &Config{Token: "new-token"}); err == nil {
				t.Fatal("expected save to fail")
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(old) {
				t.Fatalf("config changed to %q", got)
			}
			assertLoadedToken(t, path, "old-token")
			assertOnlyEntries(t, dir, "config.json")
		})
	}
}

func injectFile(t *testing.T, wrapper *failingFile) {
	t.Helper()
	original := createTemp
	createTemp = func(dir, pattern string) (tempFile, error) {
		f, err := original(dir, pattern)
		if err != nil {
			return nil, err
		}
		wrapper.tempFile = f
		return wrapper, nil
	}
	t.Cleanup(func() { createTemp = original })
}
