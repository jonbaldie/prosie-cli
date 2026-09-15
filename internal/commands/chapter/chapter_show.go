package chaptercmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChapterShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChapterShowHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter show <id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.Chapters().GetChapter(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapter)
		return 0
	}

	if chapter.Content == "" {
		fmt.Fprintln(c.Out, "No content.")
	} else {
		fmt.Fprintln(c.Out, chapter.Content)
	}

	return 0
}

func printChapterShowHelp(c *command.Environment) {
	help := `Show chapter prose content.

Usage:
  prosie chapter show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
