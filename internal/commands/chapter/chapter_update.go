package chaptercmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChapterUpdate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	input := newChapterInput(fs, "Chapter prose content")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chapter update <id> [flags]", Missing: "chapter ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChapterUpdateHelp)
	}

	id := posArgs[0]

	params, err := input.params(c, fs)
	if err != nil {
		fmt.Fprintln(c.Err, err)
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.Chapters().UpdateChapter(c.Ctx(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated chapter %d (%q).\n", chapter.ID, chapter.Display().Title)
	return 0
}

func printChapterUpdateHelp(c *command.Environment) {
	help := `Update an existing chapter.

Usage:
  prosie chapter update <id> [flags]

Flags:
      --character string Chapter character
      --content string   Chapter prose content
      --file string      Path to file containing chapter prose ('-' for standard input)
  -h, --help             Show help for command
      --json             Format output as JSON
      --order int        Chapter order
      --pov string       Chapter point of view
      --summary string   Chapter summary
      --title string     Chapter title (sent as name)

Use non-empty --content or --file, not both.
`
	fmt.Fprint(c.Out, help)
}
