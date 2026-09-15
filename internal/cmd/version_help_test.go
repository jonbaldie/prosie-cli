package cmd_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func TestVersionHelpIsSuccessful(t *testing.T) {
	root := cmd.NewRootCmd()
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	root.Out, root.Err, root.ConfigPath = out, errOut, filepath.Join(t.TempDir(), "config.json")

	code := root.Execute([]string{"version", "--help"})

	if code != 0 || errOut.Len() != 0 {
		t.Fatalf("code=%d stderr=%q; want 0 empty", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Usage:") || !strings.Contains(out.String(), "prosie version") {
		t.Fatalf("stdout=%q; want usage text for prosie version", out.String())
	}
}
