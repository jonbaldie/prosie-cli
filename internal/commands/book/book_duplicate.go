package bookcmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
)

func executeBookDuplicate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("duplicate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie book duplicate <id> [flags]", Missing: "book ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printBookDuplicateHelp)
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.Books().DuplicateBook(c.Ctx(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error duplicating book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, book)
		return 0
	}

	fmt.Fprintf(c.Out, "Duplicated book %d to %q (ID: %d).\n", id, book.Title, book.ID)
	return 0
}

func printBookDuplicateHelp(c *command.Environment) {
	help := `Duplicate a book.

Usage:
  prosie book duplicate <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
