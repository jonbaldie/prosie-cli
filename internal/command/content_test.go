package command_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/jonbaldie/prosie-cli/internal/command"
)

func TestReadContent(t *testing.T) {
	readErr := errors.New("input failed")
	for _, tc := range []struct {
		name, inline, path, want, errorText string
		in                                  io.Reader
	}{
		{"inline", "  Prose.\n", "", "  Prose.\n", "", iotest.ErrReader(readErr)},
		{"omitted", "", "", "", "", iotest.ErrReader(readErr)},
		{"standard input", "", command.StdinPath, "  Prose.\n", "", strings.NewReader("  Prose.\n")},
		{"empty standard input", "", command.StdinPath, "", "", strings.NewReader("")},
		{"read failure", "", command.StdinPath, "", "error reading content file: input failed", iotest.ErrReader(readErr)},
		{"conflict", "Inline prose.", command.StdinPath, "", "error: use either inline content or a content file, not both", iotest.ErrReader(readErr)},
		{"disk conflict", "Inline prose.", "/nonexistent/prosie-content.txt", "", "error: use either inline content or a content file, not both", iotest.ErrReader(readErr)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := command.ReadContent(&command.Environment{In: tc.in}, tc.inline, tc.path, "content")
			if tc.errorText != "" {
				if err == nil || err.Error() != tc.errorText || got != "" {
					t.Fatalf("text=%q error=%v", got, err)
				}
				if tc.name == "read failure" && !errors.Is(err, readErr) {
					t.Fatalf("read error not preserved: %v", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("text=%q error=%v, want %q", got, err, tc.want)
			}
		})
	}
}
