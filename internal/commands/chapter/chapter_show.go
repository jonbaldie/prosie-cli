package chaptercmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChapterShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", c.JSON, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chapter show <id> [flags]", Missing: "chapter ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChapterShowHelp)
	}

	id := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.Chapters().GetChapter(c.Ctx(), id)
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
