package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"text/tabwriter"
)

func executeBookList(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := command.ParseFlagsAndArgs(fs, args); err != nil {
		return command.FlagError(c, err, printBookListHelp)
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	books, err := cli.Books().ListBooks(context.Background())
	if err != nil {
		fmt.Fprintf(c.Err, "error listing books: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, books)
		return 0
	}

	if len(books) == 0 {
		fmt.Fprintln(c.Out, "No books found.")
		return 0
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tWORDS\tTARGET\tUPDATED")
	for _, b := range books {
		targetStr := "-"
		if b.TargetWordCount != nil && *b.TargetWordCount > 0 {
			targetStr = fmt.Sprintf("%d", *b.TargetWordCount)
		}
		updatedStr := command.FormatTimestamp(b.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\n", b.ID, b.Title, b.WordCount, targetStr, updatedStr)
	}
	_ = w.Flush()

	return 0
}

func printBookListHelp(c *command.Environment) {
	help := `List all books in your library.

Usage:
  prosie book list [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
