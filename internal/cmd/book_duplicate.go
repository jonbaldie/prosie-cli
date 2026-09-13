package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
)

func (c *RootCmd) executeBookDuplicate(args []string) int {
	fs := flag.NewFlagSet("duplicate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintBookDuplicateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie book duplicate <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.DuplicateBook(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error duplicating book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(book)
		return 0
	}

	fmt.Fprintf(c.Out, "Duplicated book %d to %q (ID: %d).\n", id, book.Title, book.ID)
	return 0
}

func (c *RootCmd) PrintBookDuplicateHelp() {
	help := `Duplicate a book.

Usage:
  prosie book duplicate <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
