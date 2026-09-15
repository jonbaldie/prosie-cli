package command

import (
	"flag"
	"io"
	"testing"
)

func TestParseFlagsAndArgs(t *testing.T) {
	t.Run("flags before positional", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		title := fs.String("title", "", "")
		jsonFlag := fs.Bool("json", false, "")

		pos, err := ParseFlagsAndArgs(fs, []string{"--title", "Hello", "--json", "arg1", "arg2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *title != "Hello" || !*jsonFlag {
			t.Fatalf("flags not parsed: title=%s, json=%v", *title, *jsonFlag)
		}
		if len(pos) != 2 || pos[0] != "arg1" || pos[1] != "arg2" {
			t.Fatalf("unexpected pos args: %v", pos)
		}
	})

	t.Run("flags after positional", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		title := fs.String("title", "", "")
		jsonFlag := fs.Bool("json", false, "")

		pos, err := ParseFlagsAndArgs(fs, []string{"arg1", "--title", "Hello", "--json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *title != "Hello" || !*jsonFlag {
			t.Fatalf("flags not parsed: title=%s, json=%v", *title, *jsonFlag)
		}
		if len(pos) != 1 || pos[0] != "arg1" {
			t.Fatalf("unexpected pos args: %v", pos)
		}
	})

	t.Run("flags with equal sign", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		title := fs.String("title", "", "")

		pos, err := ParseFlagsAndArgs(fs, []string{"123", "--title=Hello World"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *title != "Hello World" {
			t.Fatalf("title not parsed: %s", *title)
		}
		if len(pos) != 1 || pos[0] != "123" {
			t.Fatalf("unexpected pos args: %v", pos)
		}
	})

	t.Run("double dash separator", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		jsonFlag := fs.Bool("json", false, "")

		pos, err := ParseFlagsAndArgs(fs, []string{"--json", "--", "--not-a-flag"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !*jsonFlag {
			t.Fatalf("json flag not parsed")
		}
		if len(pos) != 1 || pos[0] != "--not-a-flag" {
			t.Fatalf("unexpected pos args: %v", pos)
		}
	})
}
