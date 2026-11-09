package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormalReleaseExactTag(t *testing.T) {
	dir := t.TempDir()
	git := filepath.Join(dir, "git")
	script := "#!/bin/sh\ncase \"$3\" in\nstatus) printf '%s' \"$FORMAL_STATUS\"; exit \"${FORMAL_STATUS_EXIT:-0}\";;\ndescribe) printf '%s\\n' \"$FORMAL_TAG\";;\nesac\n"
	if err := os.WriteFile(git, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	for _, tc := range []struct {
		status, exit, tag, want string
		ok                      bool
	}{
		{"", "0", "v1.2.3", "1.2.3", true},
		{" M file", "0", "v1.2.3", "", false},
		{"", "1", "v1.2.3", "", false},
		{"", "0", "bad", "", false},
	} {
		t.Setenv("FORMAL_STATUS", tc.status)
		t.Setenv("FORMAL_STATUS_EXIT", tc.exit)
		t.Setenv("FORMAL_TAG", tc.tag)
		got, ok := exactTagVersion(dir)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("status=%q exit=%s tag=%q: got %q, %v", tc.status, tc.exit, tc.tag, got, ok)
		}
	}
}
