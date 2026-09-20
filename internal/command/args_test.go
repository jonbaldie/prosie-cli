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

	t.Run("bare dash positional argument", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		jsonFlag := fs.Bool("json", false, "")

		pos, err := ParseFlagsAndArgs(fs, []string{"7", "-", "1,2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *jsonFlag {
			t.Fatalf("json flag unexpectedly set")
		}
		expected := []string{"7", "-", "1,2"}
		if len(pos) != len(expected) {
			t.Fatalf("expected pos args %v, got %v", expected, pos)
		}
		for i, v := range expected {
			if pos[i] != v {
				t.Fatalf("expected pos args %v, got %v", expected, pos)
			}
		}
	})

	t.Run("never returns unsupplied token for bare dash cases", func(t *testing.T) {
		testCases := []struct {
			name     string
			args     []string
			expected []string
		}{
			{
				name:     "bare dash only",
				args:     []string{"-"},
				expected: []string{"-"},
			},
			{
				name:     "leading dash",
				args:     []string{"-", "pos1", "pos2"},
				expected: []string{"-", "pos1", "pos2"},
			},
			{
				name:     "trailing dash",
				args:     []string{"pos1", "pos2", "-"},
				expected: []string{"pos1", "pos2", "-"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fs := flag.NewFlagSet("test", flag.ContinueOnError)
				fs.SetOutput(io.Discard)

				pos, err := ParseFlagsAndArgs(fs, tc.args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(pos) != len(tc.expected) {
					t.Fatalf("expected pos args %v, got %v", tc.expected, pos)
				}
				for i, v := range tc.expected {
					if pos[i] != v {
						t.Fatalf("expected pos args %v, got %v", tc.expected, pos)
					}
				}
			})
		}
	})

	t.Run("dash as flag value", func(t *testing.T) {
		t.Run("flag before positional", func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			title := fs.String("title", "", "")

			pos, err := ParseFlagsAndArgs(fs, []string{"--title", "-", "pos1"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *title != "-" {
				t.Fatalf("expected title '-' but got %q", *title)
			}
			if len(pos) != 1 || pos[0] != "pos1" {
				t.Fatalf("expected pos [pos1], got %v", pos)
			}
		})

		t.Run("flag after positional", func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			title := fs.String("title", "", "")

			pos, err := ParseFlagsAndArgs(fs, []string{"pos1", "--title", "-"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *title != "-" {
				t.Fatalf("expected title '-' but got %q", *title)
			}
			if len(pos) != 1 || pos[0] != "pos1" {
				t.Fatalf("expected pos [pos1], got %v", pos)
			}
		})
	})

	t.Run("missing flag value reports the same error with and without positional arguments", func(t *testing.T) {
		testCases := []struct {
			name string
			args []string
		}{
			{name: "without positional argument", args: []string{"--title"}},
			{name: "after positional argument", args: []string{"123", "--title"}},
			{name: "after two positional arguments", args: []string{"123", "456", "--title"}},
			{name: "after bare dash positional argument", args: []string{"-", "--title"}},
			{name: "single dash flag after positional argument", args: []string{"123", "-title"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fs := flag.NewFlagSet("test", flag.ContinueOnError)
				fs.SetOutput(io.Discard)
				title := fs.String("title", "", "")

				pos, err := ParseFlagsAndArgs(fs, tc.args)
				if err == nil {
					t.Fatalf("expected an error, got title=%q pos=%v", *title, pos)
				}
				if err.Error() != "flag needs an argument: -title" {
					t.Fatalf("expected 'flag needs an argument: -title', got %q", err.Error())
				}
				if pos != nil {
					t.Fatalf("expected no pos args, got %v", pos)
				}
			})
		}
	})

	t.Run("missing integer flag value reports a missing argument", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		targetWords := fs.Int("target-words", 0, "")

		_, err := ParseFlagsAndArgs(fs, []string{"pos1", "--target-words"})
		if err == nil {
			t.Fatalf("expected an error, got target-words=%d", *targetWords)
		}
		if err.Error() != "flag needs an argument: -target-words" {
			t.Fatalf("expected 'flag needs an argument: -target-words', got %q", err.Error())
		}
	})

	t.Run("trailing boolean flag after positional arguments keeps positional arguments", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		jsonFlag := fs.Bool("json", false, "")

		pos, err := ParseFlagsAndArgs(fs, []string{"123", "--json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !*jsonFlag {
			t.Fatalf("json flag not parsed")
		}
		if len(pos) != 1 || pos[0] != "123" {
			t.Fatalf("expected pos [123], got %v", pos)
		}
	})

	t.Run("help flag returns flag.ErrHelp", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)

		_, err := ParseFlagsAndArgs(fs, []string{"--help"})
		if err != flag.ErrHelp {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})
}
