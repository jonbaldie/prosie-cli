package chaptercmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func executeChapterCreate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	input := newChapterInput(fs, "Initial chapter prose content")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chapter create <book-id> [flags]", Missing: "book ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChapterCreateHelp)
	}

	bookID := posArgs[0]

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

	chapter, err := cli.Chapters().CreateChapter(c.Ctx(), bookID, client.CreateChapterParams(params))
	if err != nil {
		fmt.Fprintf(c.Err, "error creating chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Created chapter %q (ID: %d).\n", chapter.Display().Title, chapter.ID)
	return 0
}

func printChapterCreateHelp(c *command.Environment) {
	help := `Create a new chapter in a book.

Usage:
  prosie chapter create <book-id> [flags]

Flags:
      --character string Chapter character
      --content string   Initial chapter prose content
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
