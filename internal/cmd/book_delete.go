package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (c *RootCmd) executeBookDelete(args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintBookDeleteHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	if !*yesFlag {
		fmt.Fprintf(c.Out, "Are you sure you want to delete book %d? [y/N]: ", id)
		scanner := bufio.NewScanner(c.In)
		if scanner.Scan() {
			ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if ans != "y" && ans != "yes" {
				fmt.Fprintln(c.Out, "Deletion cancelled.")
				return 0
			}
		} else {
			fmt.Fprintln(c.Out, "Deletion cancelled.")
			return 0
		}
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.DeleteBook(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"id":      id,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted book %d.\n", id)
	return 0
}

func (c *RootCmd) PrintBookDeleteHelp() {
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
