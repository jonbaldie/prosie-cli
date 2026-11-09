package config

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

func TestFormalStorageRelativeLink(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "link")
	if err := os.Symlink("target", p); err != nil {
		t.Fatal(err)
	}
	got, err := resolveLinks(p)
	if err != nil || got != filepath.Join(d, "target") {
		t.Fatalf("resolveLinks = %q, %v", got, err)
	}
}
