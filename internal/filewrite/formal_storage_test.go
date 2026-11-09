package filewrite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFormalStorageLinkBudget(t *testing.T) {
	for _, count := range []int{39, 40} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			d := t.TempDir()
			for i := 0; i < count; i++ {
				if err := os.Symlink(fmt.Sprint(i+1), filepath.Join(d, fmt.Sprint(i))); err != nil {
					t.Fatal(err)
				}
			}
			got, err := resolveLinks(filepath.Join(d, "0"))
			if count == 39 && (err != nil || got != filepath.Join(d, "39")) {
				t.Fatalf("got %q, %v", got, err)
			}
			if count == 40 && err == nil {
				t.Fatalf("40 links returned %q without error", got)
			}
		})
	}
}

func TestFormalStorageRenameFailureCleansTemp(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "directory")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := replaceRegular(target, []byte("new"), 0600); err == nil {
		t.Fatal("rename over directory succeeded")
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("destination lost: %v, %v", info, err)
	}
	matches, err := filepath.Glob(filepath.Join(d, ".directory.tmp-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files: %v, %v", matches, err)
	}
}

func TestFormalStorageReplaceMode(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "target")
	link := filepath.Join(d, "link")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", link); err != nil {
		t.Fatal(err)
	}
	if err := Replace(link, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatalf("target = %q, %v", data, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	info, err = os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link = %v, %v", info, err)
	}
}
