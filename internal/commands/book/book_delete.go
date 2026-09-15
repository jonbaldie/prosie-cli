package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
)

func executeBookDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printBookDeleteHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie book delete <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	if !command.ConfirmDeletion(c, *yesFlag, "book", id) {
		return 0
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.Books().DeleteBook(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, map[string]any{
			"id":      id,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted book %d.\n", id)
	return 0
}

func printBookDeleteHelp(c *command.Environment) {
	help := `Delete a book.

Usage:
  prosie book delete <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
  -y, --yes    Skip confirmation prompt
`
	fmt.Fprint(c.Out, help)
}
